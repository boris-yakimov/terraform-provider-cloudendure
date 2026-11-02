package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/boris-yakimov/terraform-provider-cloudendure/internal/client"
)

var (
	_ resource.Resource                = (*recoveryPlanResource)(nil)
	_ resource.ResourceWithConfigure   = (*recoveryPlanResource)(nil)
	_ resource.ResourceWithImportState = (*recoveryPlanResource)(nil)
)

// NewRecoveryPlanResource returns the cloudendure_recovery_plan resource.
func NewRecoveryPlanResource() resource.Resource {
	return &recoveryPlanResource{}
}

type recoveryPlanResource struct {
	data *providerData
}

type recoveryPlanModel struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Steps types.List   `tfsdk:"steps"`
}

type recoveryPlanStepModel struct {
	Name       types.String `tfsdk:"name"`
	MachineIDs types.List   `tfsdk:"machine_ids"`
	WaitBefore types.String `tfsdk:"wait_before"`
}

var recoveryPlanStepType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"name":        types.StringType,
	"machine_ids": types.ListType{ElemType: types.StringType},
	"wait_before": types.StringType,
}}

func (r *recoveryPlanResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recovery_plan"
}

func (r *recoveryPlanResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		DeprecationMessage: deprecationMessage,
		Description:        "Manages a CloudEndure recovery plan: an ordered list of steps, each launching a group of machines.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "Recovery plan ID.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "Recovery plan name.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"steps": schema.ListNestedAttribute{
				Description: "Ordered launch steps.",
				Required:    true,
				Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Step name.",
							Optional:    true,
							Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
						},
						"machine_ids": schema.ListAttribute{
							Description: "IDs of the machines launched in this step.",
							ElementType: types.StringType,
							Required:    true,
							Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
						},
						"wait_before": schema.StringAttribute{
							Description: "Time to wait before launching this step, passed to the API as-is.",
							Optional:    true,
							Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
						},
					},
				},
			},
		},
	}
	resp.Schema.MarkdownDescription = deprecationNotice + resp.Schema.Description
}

func (r *recoveryPlanResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.data = providerDataFrom(req.ProviderData, &resp.Diagnostics)
}

func (r *recoveryPlanResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan recoveryPlanModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := plan.toAPI(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	rp, err := r.data.client.CreateRecoveryPlan(ctx, r.data.projectID, body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create CloudEndure recovery plan", err.Error())
		return
	}

	plan.fromAPI(rp, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recoveryPlanResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state recoveryPlanModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rp, err := r.data.client.GetRecoveryPlan(ctx, r.data.projectID, state.ID.ValueString())
	if client.IsNotFound(err) {
		tflog.Warn(ctx, "recovery plan not found, removing from state", map[string]any{"id": state.ID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read CloudEndure recovery plan", err.Error())
		return
	}

	state.fromAPI(rp, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *recoveryPlanResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan recoveryPlanModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := plan.toAPI(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	rp, err := r.data.client.UpdateRecoveryPlan(ctx, r.data.projectID, plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update CloudEndure recovery plan", err.Error())
		return
	}

	plan.fromAPI(rp, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recoveryPlanResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state recoveryPlanModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.data.client.DeleteRecoveryPlan(ctx, r.data.projectID, state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete CloudEndure recovery plan", err.Error())
	}
}

func (r *recoveryPlanResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (m *recoveryPlanModel) toAPI(ctx context.Context, diags *diag.Diagnostics) *client.RecoveryPlan {
	var steps []recoveryPlanStepModel
	diags.Append(m.Steps.ElementsAs(ctx, &steps, false)...)

	rp := &client.RecoveryPlan{
		ID:    stringValue(m.ID),
		Name:  m.Name.ValueString(),
		Steps: client.RecoveryPlanSteps{Items: make([]client.RecoveryPlanStep, 0, len(steps))},
	}
	for _, s := range steps {
		var machineIDs []string
		diags.Append(s.MachineIDs.ElementsAs(ctx, &machineIDs, false)...)
		rp.Steps.Items = append(rp.Steps.Items, client.RecoveryPlanStep{
			Name:       stringValue(s.Name),
			MachineIDs: machineIDs,
			WaitBefore: stringValue(s.WaitBefore),
		})
	}
	return rp
}

func (m *recoveryPlanModel) fromAPI(rp *client.RecoveryPlan, diags *diag.Diagnostics) {
	m.ID = types.StringValue(rp.ID)
	m.Name = types.StringValue(rp.Name)

	steps := make([]attr.Value, 0, len(rp.Steps.Items))
	for _, s := range rp.Steps.Items {
		obj, d := types.ObjectValue(recoveryPlanStepType.AttrTypes, map[string]attr.Value{
			"name":        stringOrNull(s.Name),
			"machine_ids": stringList(s.MachineIDs),
			"wait_before": stringOrNull(s.WaitBefore),
		})
		diags.Append(d...)
		steps = append(steps, obj)
	}
	m.Steps = types.ListValueMust(recoveryPlanStepType, steps)
}
