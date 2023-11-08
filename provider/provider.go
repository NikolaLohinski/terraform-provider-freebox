package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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
}

func (p *freeboxProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "freebox"
	resp.Version = p.version
}

func (p *freeboxProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{},
	}
}

func (p *freeboxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data freeboxProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO

	resp.DataSourceData = nil
	resp.ResourceData = nil
}

func (p *freeboxProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSomeResource,
	}
}

func (p *freeboxProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// NewDeploymentDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &freeboxProvider{
			version: version,
		}
	}
}
