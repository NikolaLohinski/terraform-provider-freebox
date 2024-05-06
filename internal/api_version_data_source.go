package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nikolalohinski/free-go/client"
)

var _ datasource.DataSource = &APIVersionDataSource{}

func NewAPIVersionDataSource() datasource.DataSource {
	return &APIVersionDataSource{}
}

type APIVersionDataSource struct {
	client client.Client
}

type APIVersionDataSourceModel struct {
	UID            types.String `tfsdk:"uid"`
	APIVersion     types.String `tfsdk:"api_version"`
	APIDomain      types.String `tfsdk:"api_domain"`
	APIBaseURL     types.String `tfsdk:"api_base_url"`
	BoxModelName   types.String `tfsdk:"box_model_name"`
	BoxModel       types.String `tfsdk:"box_model"`
	HTTPSPort      types.Int64  `tfsdk:"https_port"`
	HTTPSAvailable types.Bool   `tfsdk:"https_available"`
}

func (a *APIVersionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_version"
}

func (a *APIVersionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Discovery of the Freebox over HTTP(S)",
		Attributes: map[string]schema.Attribute{
			"box_model_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Box model display name",
			},
			"box_model": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Box model",
			},
			"api_domain": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The domain to use in place of hardcoded Freebox IP",
			},
			"api_base_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The API root path on the HTTP server",
			},
			"api_version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The current API version on the Freebox",
			},
			"https_port": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Port to use for remote https access to the Freebox API",
			},
			"https_available": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Tells if HTTPS has been configured on the Freebox",
			},
			"uid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The device unique id",
			},
		},
	}
}

func (a *APIVersionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	a.client = client
}

func (a *APIVersionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data APIVersionDataSourceModel

	apiVersion, err := a.client.APIVersion(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get API version",
			err.Error(),
		)
		return
	}

	data.UID = types.StringValue(apiVersion.UID)
	data.APIBaseURL = types.StringValue(apiVersion.APIBaseURL)
	data.APIDomain = types.StringValue(apiVersion.APIDomain)
	data.APIVersion = types.StringValue(apiVersion.APIVersion)
	data.BoxModel = types.StringValue(apiVersion.BoxModel)
	data.BoxModelName = types.StringValue(apiVersion.BoxModelName)
	data.HTTPSAvailable = types.BoolValue(apiVersion.HTTPSAvailable)
	data.HTTPSPort = types.Int64Value(int64(apiVersion.HTTPSPort))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
