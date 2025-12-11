package internal

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

var _ datasource.DataSource = &LanConfigDataSource{}

func NewLanConfigDataSource() datasource.DataSource {
	return &LanConfigDataSource{}
}

type LanConfigDataSource struct {
	client client.Client
}

type LanConfigDataSourceModel struct {
	IP          types.String `tfsdk:"ip"`           // Freebox Server IPv4 address
	Name        types.String `tfsdk:"name"`         // Freebox Server name
	NameDNS     types.String `tfsdk:"name_dns"`     // Freebox Server DNS name
	NameMDNS    types.String `tfsdk:"name_mdns"`    // Freebox Server mDNS name
	NameNetBIOS types.String `tfsdk:"name_netbios"` // Freebox Server netbios name
	Mode        types.String `tfsdk:"mode"`         // LAN mode (router or bridge)
}

func (a *LanConfigDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lan_config"
}

func (a *LanConfigDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get information about the LAN configuration.",
		Attributes: map[string]schema.Attribute{
			"ip": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Freebox Server IPv4 address",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$`),
						"Must be a valid IPv4 address",
					),
				},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Freebox Server name",
			},
			"name_dns": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Freebox Server DNS name",
			},
			"name_mdns": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Freebox Server mDNS name",
			},
			"name_netbios": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Freebox Server netbios name",
				Validators: []validator.String{
					stringvalidator.All(
						stringvalidator.LengthAtMost(16),
						stringvalidator.RegexMatches(
							regexp.MustCompile(`[^\/:*?"<>|]`),
							`Cannot contain forbidden characters : \ / : * ? < > |`,
						),
					),
				},
			},
			"mode": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "LAN mode",
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(freeboxTypes.LanModeBridge),
						string(freeboxTypes.LanModeRouter),
					),
				},
			},
		},
	}
}

func (a *LanConfigDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (a *LanConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LanConfigDataSourceModel

	resp.Diagnostics.Append(resp.State.Get(ctx, &data)...)

	lanConfig, err := a.client.GetLanConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get LAN configuration",
			fmt.Sprintf("Failed to get LAN configuration: %s", err),
		)
		return
	}

	data.IP = types.StringValue(lanConfig.IP)
	data.Name = types.StringValue(lanConfig.Name)
	data.NameDNS = types.StringValue(lanConfig.NameDNS)
	data.NameMDNS = types.StringValue(lanConfig.NameMDNS)
	data.NameNetBIOS = types.StringValue(lanConfig.NameNetBIOS)
	data.Mode = types.StringValue(lanConfig.Mode)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
