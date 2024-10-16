package internal

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nikolalohinski/free-go/client"
)

const (
	defaultEndpoint = "http://mafreebox.freebox.fr"
	defaultVersion  = "latest"

	environmentVariableEndpoint = "FREEBOX_ENDPOINT"
	environmentVariableVersion  = "FREEBOX_VERSION"
	environmentVariableAppID    = "FREEBOX_APP_ID"
	environmentVariableToken    = "FREEBOX_TOKEN"
)

// Ensure freeboxProvider satisfies various provider interfaces.
var _ provider.Provider = &freeboxProvider{}

// freeboxProvider defines the provider implementation.
type freeboxProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// freeboxProviderModel describes the provider data model.
type freeboxProviderModel struct {
	Endpoint   types.String `tfsdk:"endpoint"`
	APIVersion types.String `tfsdk:"api_version"`
	AppID      types.String `tfsdk:"app_id"`
	Token      types.String `tfsdk:"token"`
}

func (p *freeboxProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "freebox"
	resp.Version = p.version
}

func (p *freeboxProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The address of the Freebox (env: `" + environmentVariableEndpoint + "`) (default: `\"" + defaultEndpoint + "\"`)",
			},
			"api_version": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The version of the API to use (env: `" + environmentVariableVersion + "`) (default: `\"" + defaultVersion + "\"`)",
			},
			"app_id": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The ID of the application you created to authenticate to the Freebox (see [the login documentation](https://dev.freebox.fr/sdk/os/login/)) (env: `" + environmentVariableAppID + "`)",
			},
			"token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The private token to authenticate to the Freebox (see [the login documentation](https://dev.freebox.fr/sdk/os/login/)) (env: `" + environmentVariableToken + "`)",
			},
		},
	}
}

func (p *freeboxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data freeboxProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	endpoint, ok := os.LookupEnv(environmentVariableEndpoint)
	if !ok {
		endpoint = defaultEndpoint
	}
	if !data.Endpoint.IsNull() && !data.Endpoint.IsUnknown() {
		endpoint = data.Endpoint.ValueString()
	}

	version, ok := os.LookupEnv(environmentVariableVersion)
	if !ok {
		version = defaultVersion
	}
	if !data.APIVersion.IsNull() && !data.APIVersion.IsUnknown() {
		version = data.APIVersion.ValueString()
	}

	client, err := client.New(endpoint, version)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create freebox client", err.Error())
		return
	}

	if !data.AppID.IsNull() && !data.AppID.IsUnknown() {
		client = client.WithAppID(data.AppID.ValueString())
	} else if appID, ok := os.LookupEnv(environmentVariableAppID); ok {
		client = client.WithAppID(appID)
	}

	if !data.Token.IsNull() && !data.Token.IsUnknown() {
		client = client.WithPrivateToken(data.Token.ValueString())
	} else if token, ok := os.LookupEnv(environmentVariableToken); ok {
		client = client.WithPrivateToken(token)
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *freeboxProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewVirtualMachineResource,
		NewRemoteFileResource,
	}
}

func (p *freeboxProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAPIVersionDataSource,
		NewVirtualDiskDataSource,
	}
}

func NewProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &freeboxProvider{
			version: version,
		}
	}
}
