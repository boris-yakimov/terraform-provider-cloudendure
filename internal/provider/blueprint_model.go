package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/boris-yakimov/terraform-provider-cloudendure/internal/client"
)

// blueprintModel is shared by the cloudendure_blueprint resource and data source.
type blueprintModel struct {
	ID        types.String `tfsdk:"id"`
	MachineID types.String `tfsdk:"machine_id"`

	MachineName             types.String `tfsdk:"machine_name"`
	Region                  types.String `tfsdk:"region"`
	InstanceType            types.String `tfsdk:"instance_type"`
	RecommendedInstanceType types.String `tfsdk:"recommended_instance_type"`
	RecommendedPrivateIP    types.String `tfsdk:"recommended_private_ip"`
	IAMRole                 types.String `tfsdk:"iam_role"`
	PlacementGroup          types.String `tfsdk:"placement_group"`
	Tenancy                 types.String `tfsdk:"tenancy"`
	DedicatedHostIdentifier types.String `tfsdk:"dedicated_host_identifier"`
	LaunchOnInstanceID      types.String `tfsdk:"launch_on_instance_id"`

	ByolOnDedicatedInstance types.Bool `tfsdk:"byol_on_dedicated_instance"`
	ForceUEFI               types.Bool `tfsdk:"force_uefi"`
	RunAfterLaunch          types.Bool `tfsdk:"run_after_launch"`
	UseSharedRAM            types.Bool `tfsdk:"use_shared_ram"`

	SubnetIDs           types.List   `tfsdk:"subnet_ids"`
	SecurityGroupIDs    types.List   `tfsdk:"security_group_ids"`
	SecurityGroupAction types.String `tfsdk:"security_group_action"`
	PrivateIPAction     types.String `tfsdk:"private_ip_action"`
	PrivateIPs          types.List   `tfsdk:"private_ips"`
	PublicIPAction      types.String `tfsdk:"public_ip_action"`
	StaticIP            types.String `tfsdk:"static_ip"`
	StaticIPAction      types.String `tfsdk:"static_ip_action"`
	NetworkInterface    types.String `tfsdk:"network_interface"`
	SubnetsHostProject  types.String `tfsdk:"subnets_host_project"`

	ScsiAdapterType    types.String `tfsdk:"scsi_adapter_type"`
	NetworkAdapterType types.String `tfsdk:"network_adapter_type"`
	Cpus               types.Int64  `tfsdk:"cpus"`
	CoresPerCPU        types.Int64  `tfsdk:"cores_per_cpu"`
	MbRAM              types.Int64  `tfsdk:"mb_ram"`
	ComputeLocationID  types.String `tfsdk:"compute_location_id"`
	LogicalLocationID  types.String `tfsdk:"logical_location_id"`

	Tags  types.Map  `tfsdk:"tags"`
	Disks types.List `tfsdk:"disks"`
}

type diskModel struct {
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	Iops       types.Int64  `tfsdk:"iops"`
	Throughput types.Int64  `tfsdk:"throughput"`
}

var diskObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"name":       types.StringType,
	"type":       types.StringType,
	"iops":       types.Int64Type,
	"throughput": types.Int64Type,
}}

// toAPI builds an API request body from the model. Null or unknown values
// are left out so that the API keeps (or computes) its current value.
func (m *blueprintModel) toAPI(ctx context.Context, diags *diag.Diagnostics) *client.Blueprint {
	bp := &client.Blueprint{
		ID:        stringValue(m.ID),
		MachineID: stringValue(m.MachineID),

		MachineName:             stringValue(m.MachineName),
		InstanceType:            stringValue(m.InstanceType),
		IAMRole:                 stringValue(m.IAMRole),
		PlacementGroup:          stringValue(m.PlacementGroup),
		Tenancy:                 stringValue(m.Tenancy),
		DedicatedHostIdentifier: stringValue(m.DedicatedHostIdentifier),
		LaunchOnInstanceID:      stringValue(m.LaunchOnInstanceID),

		ByolOnDedicatedInstance: boolPointer(m.ByolOnDedicatedInstance),
		ForceUEFI:               boolPointer(m.ForceUEFI),
		RunAfterLaunch:          boolPointer(m.RunAfterLaunch),
		UseSharedRAM:            boolPointer(m.UseSharedRAM),

		SubnetIDs:           stringSlice(ctx, m.SubnetIDs, diags),
		SecurityGroupIDs:    stringSlice(ctx, m.SecurityGroupIDs, diags),
		SecurityGroupAction: stringValue(m.SecurityGroupAction),
		PrivateIPAction:     stringValue(m.PrivateIPAction),
		PrivateIPs:          stringSlice(ctx, m.PrivateIPs, diags),
		PublicIPAction:      stringValue(m.PublicIPAction),
		StaticIP:            stringValue(m.StaticIP),
		StaticIPAction:      stringValue(m.StaticIPAction),
		NetworkInterface:    stringValue(m.NetworkInterface),
		SubnetsHostProject:  stringValue(m.SubnetsHostProject),

		ScsiAdapterType:    stringValue(m.ScsiAdapterType),
		NetworkAdapterType: stringValue(m.NetworkAdapterType),
		Cpus:               int64Pointer(m.Cpus),
		CoresPerCPU:        int64Pointer(m.CoresPerCPU),
		MbRAM:              int64Pointer(m.MbRAM),
		ComputeLocationID:  stringValue(m.ComputeLocationID),
		LogicalLocationID:  stringValue(m.LogicalLocationID),
	}

	if tags := stringMap(ctx, m.Tags, diags); tags != nil {
		list := make([]client.Tag, 0, len(tags))
		for k, v := range tags {
			list = append(list, client.Tag{Key: k, Value: v})
		}
		bp.Tags = &list
	}

	if !m.Disks.IsNull() && !m.Disks.IsUnknown() {
		var disks []diskModel
		diags.Append(m.Disks.ElementsAs(ctx, &disks, false)...)
		list := make([]client.Disk, 0, len(disks))
		for _, d := range disks {
			list = append(list, client.Disk{
				Name:       stringValue(d.Name),
				Type:       stringValue(d.Type),
				Iops:       int64Pointer(d.Iops),
				Throughput: int64Pointer(d.Throughput),
			})
		}
		bp.Disks = &list
	}

	return bp
}

// fromAPI overwrites the model with the values returned by the API.
func (m *blueprintModel) fromAPI(bp *client.Blueprint, diags *diag.Diagnostics) {
	m.ID = types.StringValue(bp.ID)
	m.MachineID = types.StringValue(bp.MachineID)

	m.MachineName = stringOrNull(bp.MachineName)
	m.Region = stringOrNull(bp.Region)
	m.InstanceType = stringOrNull(bp.InstanceType)
	m.RecommendedInstanceType = stringOrNull(bp.RecommendedInstanceType)
	m.RecommendedPrivateIP = stringOrNull(bp.RecommendedPrivateIP)
	m.IAMRole = stringOrNull(bp.IAMRole)
	m.PlacementGroup = stringOrNull(bp.PlacementGroup)
	m.Tenancy = stringOrNull(bp.Tenancy)
	m.DedicatedHostIdentifier = stringOrNull(bp.DedicatedHostIdentifier)
	m.LaunchOnInstanceID = stringOrNull(bp.LaunchOnInstanceID)

	m.ByolOnDedicatedInstance = types.BoolPointerValue(bp.ByolOnDedicatedInstance)
	m.ForceUEFI = types.BoolPointerValue(bp.ForceUEFI)
	m.RunAfterLaunch = types.BoolPointerValue(bp.RunAfterLaunch)
	m.UseSharedRAM = types.BoolPointerValue(bp.UseSharedRAM)

	m.SubnetIDs = stringListPtr(bp.SubnetIDs)
	m.SecurityGroupIDs = stringListPtr(bp.SecurityGroupIDs)
	m.SecurityGroupAction = stringOrNull(bp.SecurityGroupAction)
	m.PrivateIPAction = stringOrNull(bp.PrivateIPAction)
	m.PrivateIPs = stringListPtr(bp.PrivateIPs)
	m.PublicIPAction = stringOrNull(bp.PublicIPAction)
	m.StaticIP = stringOrNull(bp.StaticIP)
	m.StaticIPAction = stringOrNull(bp.StaticIPAction)
	m.NetworkInterface = stringOrNull(bp.NetworkInterface)
	m.SubnetsHostProject = stringOrNull(bp.SubnetsHostProject)

	m.ScsiAdapterType = stringOrNull(bp.ScsiAdapterType)
	m.NetworkAdapterType = stringOrNull(bp.NetworkAdapterType)
	m.Cpus = types.Int64PointerValue(bp.Cpus)
	m.CoresPerCPU = types.Int64PointerValue(bp.CoresPerCPU)
	m.MbRAM = types.Int64PointerValue(bp.MbRAM)
	m.ComputeLocationID = stringOrNull(bp.ComputeLocationID)
	m.LogicalLocationID = stringOrNull(bp.LogicalLocationID)

	tags := map[string]attr.Value{}
	if bp.Tags != nil {
		for _, t := range *bp.Tags {
			tags[t.Key] = types.StringValue(t.Value)
		}
	}
	m.Tags = types.MapValueMust(types.StringType, tags)

	disks := []attr.Value{}
	if bp.Disks != nil {
		for _, d := range *bp.Disks {
			obj, objDiags := types.ObjectValue(diskObjectType.AttrTypes, map[string]attr.Value{
				"name":       types.StringValue(d.Name),
				"type":       stringOrNull(d.Type),
				"iops":       types.Int64PointerValue(d.Iops),
				"throughput": types.Int64PointerValue(d.Throughput),
			})
			diags.Append(objDiags...)
			disks = append(disks, obj)
		}
	}
	m.Disks = types.ListValueMust(diskObjectType, disks)
}
