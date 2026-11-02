package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The blueprint attributes are described once in these tables and used to
// build both the resource schema (configurable) and the data source schema
// (computed only), keeping the two in sync with blueprintModel.

type attrSpec struct {
	description string
	// oneOf restricts a string attribute to the listed values.
	oneOf []string
}

var blueprintStringAttrs = map[string]attrSpec{
	"machine_name":              {description: "Name of the target machine."},
	"instance_type":             {description: "Target instance type. Possible values come from the Region object, plus the special values `COPY_ORIGIN` and `CUSTOM`."},
	"iam_role":                  {description: "AWS only. IAM instance profile attached to the target machine."},
	"placement_group":           {description: "AWS only. Placement group of the target machine."},
	"tenancy":                   {description: "Instance tenancy.", oneOf: []string{"SHARED", "DEDICATED", "HOST"}},
	"dedicated_host_identifier": {description: "Dedicated host identifier, used when `tenancy` is `HOST`."},
	"launch_on_instance_id":     {description: "Instance ID of a target machine managed by AMS."},
	"security_group_action":     {description: "How security groups are assigned to the target machine, e.g. `FROM_POLICY`."},
	"private_ip_action":         {description: "How the private IP is assigned.", oneOf: []string{"CREATE_NEW", "COPY_ORIGIN", "CUSTOM_IP", "USE_NETWORK_INTERFACE"}},
	"public_ip_action":          {description: "Whether to allocate an ephemeral public IP. `AS_SUBNET` copies the subnet setting.", oneOf: []string{"ALLOCATE", "DONT_ALLOCATE", "AS_SUBNET"}},
	"static_ip":                 {description: "Static (elastic) IP to attach. Possible values come from the Region object."},
	"static_ip_action":          {description: "How the static IP is handled.", oneOf: []string{"EXISTING", "DONT_CREATE", "CREATE_NEW", "IF_IN_ORIGIN"}},
	"network_interface":         {description: "Network interface to attach, used when `private_ip_action` is `USE_NETWORK_INTERFACE`."},
	"subnets_host_project":      {description: "GCP only. Host project for cross-project network subnets."},
	"scsi_adapter_type":         {description: "vCenter only. SCSI adapter type."},
	"network_adapter_type":      {description: "vCenter only. Network adapter type."},
	"compute_location_id":       {description: "vCenter only. Compute location ID."},
	"logical_location_id":       {description: "vCenter only. Logical location (VM folder) ID."},
}

var blueprintComputedStringAttrs = map[string]string{
	"region":                    "ID of the target region.",
	"recommended_instance_type": "Instance type recommended by CloudEndure rightsizing.",
	"recommended_private_ip":    "Private IP recommended for the target machine.",
}

var blueprintBoolAttrs = map[string]string{
	"byol_on_dedicated_instance": "Use a BYOL Windows license when dedicated tenancy is selected.",
	"force_uefi":                 "Force UEFI boot on the target machine.",
	"run_after_launch":           "AWS only. Power on the target machine after launch.",
	"use_shared_ram":             "vCenter only. Use shared RAM.",
}

var blueprintInt64Attrs = map[string]string{
	"cpus":          "vCenter only. Number of CPUs of the target machine.",
	"cores_per_cpu": "vCenter only. Number of cores per CPU of the target machine.",
	"mb_ram":        "vCenter only. RAM of the target machine in MB.",
}

var blueprintStringListAttrs = map[string]string{
	"subnet_ids":         "AWS only. Subnets the target machine's network interface is placed in.",
	"security_group_ids": "AWS only. Security groups applied to the target machine.",
	"private_ips":        "Private IPs to assign when `private_ip_action` is `CUSTOM_IP`.",
}

const (
	blueprintTagsDescription  = "AWS only. Tags applied to the target machine."
	blueprintDisksDescription = "AWS only. Target machine disk properties, matched to source disks by name."
)

func blueprintResourceSchema() schema.Schema {
	attrs := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description:   "Blueprint ID.",
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"machine_id": schema.StringAttribute{
			Description:   "ID of the source machine the blueprint belongs to. Changing it forces a new resource.",
			Required:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"tags": schema.MapAttribute{
			Description:   blueprintTagsDescription,
			ElementType:   types.StringType,
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown()},
		},
		"disks": schema.ListNestedAttribute{
			Description:   blueprintDisksDescription,
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"name":       schema.StringAttribute{Description: "Source disk name.", Required: true},
					"type":       schema.StringAttribute{Description: "Disk type, e.g. `SSD`, `STANDARD`, `PROVISIONED_SSD`, `COPY_ORIGIN`.", Optional: true, Computed: true},
					"iops":       schema.Int64Attribute{Description: "Provisioned IOPS.", Optional: true, Computed: true},
					"throughput": schema.Int64Attribute{Description: "Provisioned throughput in MiB/s.", Optional: true, Computed: true},
				},
			},
		},
	}

	for name, spec := range blueprintStringAttrs {
		validators := []validator.String{stringvalidator.LengthAtLeast(1)}
		desc := spec.description
		if len(spec.oneOf) > 0 {
			validators = append(validators, stringvalidator.OneOf(spec.oneOf...))
			desc += " Valid values: " + backtickJoin(spec.oneOf) + "."
		}
		attrs[name] = schema.StringAttribute{
			Description:   desc,
			Optional:      true,
			Computed:      true,
			Validators:    validators,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		}
	}
	for name, desc := range blueprintComputedStringAttrs {
		attrs[name] = schema.StringAttribute{
			Description:   desc,
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		}
	}
	for name, desc := range blueprintBoolAttrs {
		attrs[name] = schema.BoolAttribute{
			Description:   desc,
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
		}
	}
	for name, desc := range blueprintInt64Attrs {
		attrs[name] = schema.Int64Attribute{
			Description:   desc,
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		}
	}
	for name, desc := range blueprintStringListAttrs {
		attrs[name] = schema.ListAttribute{
			Description:   desc,
			ElementType:   types.StringType,
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
		}
	}

	return schema.Schema{
		Description: "Manages the launch blueprint of a CloudEndure machine.\n\n" +
			"Every replicated machine has exactly one blueprint. On create the provider " +
			"adopts the machine's existing blueprint (if any) and applies the configured " +
			"settings; CloudEndure has no API to delete a blueprint, so destroying this " +
			"resource only removes it from Terraform state. Attributes that are removed " +
			"from the configuration keep their last value in CloudEndure.",
		Attributes: attrs,
	}
}

func blueprintDataSourceSchema() dsschema.Schema {
	attrs := map[string]dsschema.Attribute{
		"id": dsschema.StringAttribute{
			Description: "Blueprint ID. Exactly one of `id` or `machine_id` must be set.",
			Optional:    true,
			Computed:    true,
		},
		"machine_id": dsschema.StringAttribute{
			Description: "ID of the machine whose blueprint to look up. Exactly one of `id` or `machine_id` must be set.",
			Optional:    true,
			Computed:    true,
		},
		"tags": dsschema.MapAttribute{Description: blueprintTagsDescription, ElementType: types.StringType, Computed: true},
		"disks": dsschema.ListNestedAttribute{
			Description: blueprintDisksDescription,
			Computed:    true,
			NestedObject: dsschema.NestedAttributeObject{
				Attributes: map[string]dsschema.Attribute{
					"name":       dsschema.StringAttribute{Description: "Source disk name.", Computed: true},
					"type":       dsschema.StringAttribute{Description: "Disk type.", Computed: true},
					"iops":       dsschema.Int64Attribute{Description: "Provisioned IOPS.", Computed: true},
					"throughput": dsschema.Int64Attribute{Description: "Provisioned throughput in MiB/s.", Computed: true},
				},
			},
		},
	}
	for name, spec := range blueprintStringAttrs {
		attrs[name] = dsschema.StringAttribute{Description: spec.description, Computed: true}
	}
	for name, desc := range blueprintComputedStringAttrs {
		attrs[name] = dsschema.StringAttribute{Description: desc, Computed: true}
	}
	for name, desc := range blueprintBoolAttrs {
		attrs[name] = dsschema.BoolAttribute{Description: desc, Computed: true}
	}
	for name, desc := range blueprintInt64Attrs {
		attrs[name] = dsschema.Int64Attribute{Description: desc, Computed: true}
	}
	for name, desc := range blueprintStringListAttrs {
		attrs[name] = dsschema.ListAttribute{Description: desc, ElementType: types.StringType, Computed: true}
	}

	return dsschema.Schema{
		Description: "Looks up the blueprint of a CloudEndure machine.",
		Attributes:  attrs,
	}
}

func backtickJoin(values []string) string {
	out := ""
	for i, v := range values {
		if i > 0 {
			out += ", "
		}
		out += "`" + v + "`"
	}
	return out
}
