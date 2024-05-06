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
				MarkdownDescription: "The address of the Freebox [env: `FREEBOX_ENDPOINT`] [default: `http://mafreebox.freebox.fr`]",
			},
			"api_version": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The version of the API to use [env: `FREEBOX_VERSION`] [default: `latest`]",
			},
			"app_id": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The ID of the application you created to authenticate to the Freebox (see [the login documentation](https://dev.freebox.fr/sdk/os/login/)) [env: `FREEBOX_APP_ID`]",
			},
			"token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The private token to authenticate to the Freebox (see [the login documentation](https://dev.freebox.fr/sdk/os/login/)) [env: `FREEBOX_TOKEN`]",
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

	endpoint, ok := os.LookupEnv("FREEBOX_ENDPOINT")
	if !ok {
		endpoint = "http://mafreebox.freebox.fr"
	}
	if !data.Endpoint.IsNull() {
		endpoint = data.Endpoint.String()
	}

	version, ok := os.LookupEnv("FREEBOX_VERSION")
	if !ok {
		version = "latest"
	}
	if !data.APIVersion.IsNull() {
		version = data.APIVersion.String()
	}

	client, err := client.New(endpoint, version)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create freebox client", err.Error())
		return
	}

	if !data.AppID.IsNull() {
		client = client.WithAppID(data.AppID.String())
	} else if appID, ok := os.LookupEnv("FREEBOX_APP_ID"); ok {
		client = client.WithAppID(appID)
	}

	if !data.Token.IsNull() {
		client = client.WithPrivateToken(data.Token.String())
	} else if token, ok := os.LookupEnv("FREEBOX_TOKEN"); ok {
		client = client.WithPrivateToken(token)
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *freeboxProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewVirtualMachineResource,
	}
}

func (p *freeboxProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAPIVersionDataSource,
	}
}

func NewProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &freeboxProvider{
			version: version,
		}
	}
}
