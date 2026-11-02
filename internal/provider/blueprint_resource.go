package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/boris-yakimov/terraform-provider-cloudendure/internal/client"
)

var (
	_ resource.Resource                = (*blueprintResource)(nil)
	_ resource.ResourceWithConfigure   = (*blueprintResource)(nil)
	_ resource.ResourceWithImportState = (*blueprintResource)(nil)
)

// NewBlueprintResource returns the cloudendure_blueprint resource.
func NewBlueprintResource() resource.Resource {
	return &blueprintResource{}
}

type blueprintResource struct {
	data *providerData
}

func (r *blueprintResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprint"
}

func (r *blueprintResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = blueprintResourceSchema()
	resp.Schema.MarkdownDescription = deprecationNotice + resp.Schema.Description
}

func (r *blueprintResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.data = providerDataFrom(req.ProviderData, &resp.Diagnostics)
}

func (r *blueprintResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan blueprintModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := plan.toAPI(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	c, projectID := r.data.client, r.data.projectID

	// CloudEndure creates a blueprint automatically for every machine, so
	// adopt it when present instead of failing on a duplicate.
	existing, err := c.FindBlueprintByMachineID(ctx, projectID, body.MachineID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list CloudEndure blueprints", err.Error())
		return
	}

	var result *client.Blueprint
	if existing != nil {
		tflog.Info(ctx, "adopting existing blueprint", map[string]any{"blueprint_id": existing.ID, "machine_id": body.MachineID})
		body.ID = existing.ID
		result, err = c.UpdateBlueprint(ctx, projectID, existing.ID, body)
	} else {
		result, err = c.CreateBlueprint(ctx, projectID, body)
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to create CloudEndure blueprint", err.Error())
		return
	}

	plan.fromAPI(result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *blueprintResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state blueprintModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bp, err := r.data.client.GetBlueprint(ctx, r.data.projectID, state.ID.ValueString())
	if client.IsNotFound(err) {
		tflog.Warn(ctx, "blueprint not found, removing from state", map[string]any{"blueprint_id": state.ID.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read CloudEndure blueprint", err.Error())
		return
	}

	state.fromAPI(bp, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *blueprintResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan blueprintModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := plan.toAPI(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.data.client.UpdateBlueprint(ctx, r.data.projectID, body.ID, body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update CloudEndure blueprint", err.Error())
		return
	}

	plan.fromAPI(result, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete only removes the blueprint from state: the CloudEndure API has no
// delete operation for blueprints, which live as long as their machine.
func (r *blueprintResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state blueprintModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.AddWarning("Blueprint not deleted in CloudEndure",
		fmt.Sprintf("CloudEndure does not support deleting blueprints. Blueprint %s was removed from "+
			"Terraform state but still exists, with its last applied settings, in CloudEndure.", state.ID.ValueString()))
}

func (r *blueprintResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
