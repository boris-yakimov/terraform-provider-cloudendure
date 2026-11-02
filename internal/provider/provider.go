// Package provider implements the CloudEndure Terraform provider using the
// Terraform Plugin Framework.
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/boris-yakimov/terraform-provider-cloudendure/internal/client"
)

const (
	envAPIKey    = "CLOUDENDURE_API_KEY"
	envAPIURL    = "CLOUDENDURE_API_URL"
	envProjectID = "CLOUDENDURE_PROJECT_ID"
)

var _ provider.Provider = (*cloudEndureProvider)(nil)

// New returns a provider factory for the given version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &cloudEndureProvider{version: version}
	}
}

type cloudEndureProvider struct {
	version string
}

type providerModel struct {
	APIKey    types.String `tfsdk:"api_key"`
	APIURL    types.String `tfsdk:"api_url"`
	ProjectID types.String `tfsdk:"project_id"`
}

// providerData is handed to every resource and data source.
type providerData struct {
	client    *client.Client
	projectID string
}

func (p *cloudEndureProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cloudendure"
	resp.Version = p.version
}

func (p *cloudEndureProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		DeprecationMessage: deprecationMessage,
		Description:        "Manages CloudEndure Disaster Recovery / Migration resources.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "CloudEndure user API token. Can also be set with the `" + envAPIKey + "` environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"api_url": schema.StringAttribute{
				Description: "CloudEndure API base URL. Defaults to `" + client.DefaultBaseURL + "`. Can also be set with the `" + envAPIURL + "` environment variable.",
				Optional:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "ID of the CloudEndure project to manage. Can also be set with the `" + envProjectID + "` environment variable.",
				Optional:    true,
			},
		},
	}
	resp.Schema.MarkdownDescription = deprecationNotice + resp.Schema.Description
}

func (p *cloudEndureProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for attr, v := range map[string]types.String{"api_key": cfg.APIKey, "api_url": cfg.APIURL, "project_id": cfg.ProjectID} {
		if v.IsUnknown() {
			resp.Diagnostics.AddAttributeError(path.Root(attr), "Unknown provider configuration value",
				"The provider cannot be configured with a value that is only known after apply. "+
					"Set it statically or use the corresponding environment variable.")
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := valueOrEnv(cfg.APIKey, envAPIKey)
	apiURL := valueOrEnv(cfg.APIURL, envAPIURL)
	projectID := valueOrEnv(cfg.ProjectID, envProjectID)

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"), "Missing CloudEndure API key",
			"Set the api_key provider attribute or the "+envAPIKey+" environment variable.")
	}
	if projectID == "" {
		resp.Diagnostics.AddAttributeError(path.Root("project_id"), "Missing CloudEndure project ID",
			"Set the project_id provider attribute or the "+envProjectID+" environment variable.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	c, err := client.New(client.Config{
		BaseURL:   apiURL,
		APIToken:  apiKey,
		UserAgent: "terraform-provider-cloudendure/" + p.version,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create CloudEndure client", err.Error())
		return
	}

	tflog.Debug(ctx, "configured CloudEndure client", map[string]any{"api_url": apiURL, "project_id": projectID})

	data := &providerData{client: c, projectID: projectID}
	resp.DataSourceData = data
	resp.ResourceData = data
}

func (p *cloudEndureProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBlueprintResource,
		NewRecoveryPlanResource,
	}
}

func (p *cloudEndureProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewBlueprintDataSource,
		NewMachineDataSource,
	}
}

func valueOrEnv(v types.String, env string) string {
	if !v.IsNull() && v.ValueString() != "" {
		return v.ValueString()
	}
	return os.Getenv(env)
}
