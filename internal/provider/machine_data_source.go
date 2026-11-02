package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/boris-yakimov/terraform-provider-cloudendure/internal/client"
)

var (
	_ datasource.DataSource                     = (*machineDataSource)(nil)
	_ datasource.DataSourceWithConfigure        = (*machineDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*machineDataSource)(nil)
)

// NewMachineDataSource returns the cloudendure_machine data source.
func NewMachineDataSource() datasource.DataSource {
	return &machineDataSource{}
}

type machineDataSource struct {
	data *providerData
}

type machineModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	OS                types.String `tfsdk:"os"`
	MachineCloudID    types.String `tfsdk:"machine_cloud_id"`
	ReplicationStatus types.String `tfsdk:"replication_status"`
	Replica           types.String `tfsdk:"replica"`
	IsAgentInstalled  types.Bool   `tfsdk:"is_agent_installed"`
}

func (d *machineDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine"
}

func (d *machineDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a source machine in the CloudEndure project by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Machine ID. Exactly one of `id` or `name` must be set.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Source machine name (hostname). Exactly one of `id` or `name` must be set. Must be unique in the project.",
				Optional:    true,
				Computed:    true,
			},
			"os":                 schema.StringAttribute{Description: "Source machine operating system.", Computed: true},
			"machine_cloud_id":   schema.StringAttribute{Description: "ID of the source machine in its cloud, if any.", Computed: true},
			"replication_status": schema.StringAttribute{Description: "Replication status, e.g. `STARTED`, `PAUSED`, `STOPPED`.", Computed: true},
			"replica":            schema.StringAttribute{Description: "ID of the most recently launched target machine, if any.", Computed: true},
			"is_agent_installed": schema.BoolAttribute{Description: "Whether the CloudEndure agent is installed on the machine.", Computed: true},
		},
	}
}

func (d *machineDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *machineDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.data = providerDataFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *machineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg machineModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m, err := d.lookup(ctx, stringValue(cfg.ID), stringValue(cfg.Name))
	if err != nil {
		resp.Diagnostics.AddError("Unable to read CloudEndure machine", err.Error())
		return
	}

	state := machineModel{
		ID:                types.StringValue(m.ID),
		Name:              types.StringValue(m.SourceProperties.Name),
		OS:                stringOrNull(m.SourceProperties.OS),
		MachineCloudID:    stringOrNull(m.SourceProperties.MachineCloudID),
		ReplicationStatus: stringOrNull(m.ReplicationStatus),
		Replica:           stringOrNull(m.Replica),
		IsAgentInstalled:  types.BoolValue(m.IsAgentInstalled),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *machineDataSource) lookup(ctx context.Context, id, name string) (*client.Machine, error) {
	if id != "" {
		return d.data.client.GetMachine(ctx, d.data.projectID, id)
	}

	machines, err := d.data.client.ListMachines(ctx, d.data.projectID)
	if err != nil {
		return nil, err
	}
	var match *client.Machine
	for i := range machines {
		if machines[i].SourceProperties.Name != name {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("multiple machines named %q found; look the machine up by id instead", name)
		}
		match = &machines[i]
	}
	if match == nil {
		return nil, fmt.Errorf("no machine named %q found in project %s", name, d.data.projectID)
	}
	return match, nil
}
