package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nikolalohinski/free-go/client"
)

var _ datasource.DataSource = &NetshareSambaConfigurationDataSource{}

func NewNetshareSambaConfigurationDataSource() datasource.DataSource {
	return &NetshareSambaConfigurationDataSource{}
}

type NetshareSambaConfigurationDataSource struct {
	client client.Client
}

type NetshareSambaConfigurationDataSourceModel struct {
	FileShareEnabled  types.Bool   `tfsdk:"file_share_enabled"`
	PrintShareEnabled types.Bool   `tfsdk:"print_share_enabled"`
	LogonEnabled      types.Bool   `tfsdk:"logon_enabled"`
	LogonUser         types.String `tfsdk:"logon_user"`
	Workgroup         types.String `tfsdk:"workgroup"`
	V2Enabled         types.Bool   `tfsdk:"v2_enabled"`
}

func (a *NetshareSambaConfigurationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_netshare_samba_configuration"
}

func (a *NetshareSambaConfigurationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get information about the Samba configuration.",
		Attributes: map[string]schema.Attribute{
			"file_share_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Is file sharing enabled",
			},
			"print_share_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Is printer sharing enabled",
			},
			"logon_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Is login/password required to access shares",
			},
			"logon_user": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Login user name",
			},
			"workgroup": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Workgroup name",
			},
			"v2_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Is SMBv2/v3 enabled",
			},
		},
	}
}

func (a *NetshareSambaConfigurationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (a *NetshareSambaConfigurationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NetshareSambaConfigurationDataSourceModel

	resp.Diagnostics.Append(resp.State.Get(ctx, &data)...)

	sambaConfiguration, err := a.client.GetSambaConfiguration(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get Samba configuration",
			fmt.Sprintf("Failed to get Samba configuration: %s", err),
		)
		return
	}

	data.FileShareEnabled = types.BoolValue(sambaConfiguration.FileShareEnabled)
	data.PrintShareEnabled = types.BoolValue(sambaConfiguration.PrintShareEnabled)
	data.LogonEnabled = types.BoolValue(sambaConfiguration.LogonEnabled)
	data.LogonUser = types.StringValue(sambaConfiguration.LogonUser)
	data.Workgroup = types.StringValue(sambaConfiguration.Workgroup)
	data.V2Enabled = types.BoolValue(sambaConfiguration.V2Enabled)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
