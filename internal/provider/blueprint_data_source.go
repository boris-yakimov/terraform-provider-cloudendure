package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/boris-yakimov/terraform-provider-cloudendure/internal/client"
)

var (
	_ datasource.DataSource                     = (*blueprintDataSource)(nil)
	_ datasource.DataSourceWithConfigure        = (*blueprintDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*blueprintDataSource)(nil)
)

// NewBlueprintDataSource returns the cloudendure_blueprint data source.
func NewBlueprintDataSource() datasource.DataSource {
	return &blueprintDataSource{}
}

type blueprintDataSource struct {
	data *providerData
}

func (d *blueprintDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprint"
}

func (d *blueprintDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = blueprintDataSourceSchema()
}

func (d *blueprintDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("machine_id")),
	}
}

func (d *blueprintDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.data = providerDataFrom(req.ProviderData, &resp.Diagnostics)
}

func (d *blueprintDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg blueprintModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, projectID := d.data.client, d.data.projectID

	var (
		bp  *client.Blueprint
		err error
	)
	if id := stringValue(cfg.ID); id != "" {
		bp, err = c.GetBlueprint(ctx, projectID, id)
	} else {
		machineID := stringValue(cfg.MachineID)
		bp, err = c.FindBlueprintByMachineID(ctx, projectID, machineID)
		if err == nil && bp == nil {
			err = fmt.Errorf("no blueprint found for machine %q", machineID)
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read CloudEndure blueprint", err.Error())
		return
	}

	cfg.fromAPI(bp, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
