package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	"github.com/nikolalohinski/terraform-provider-freebox/internal/models"
)

var _ datasource.DataSource = &LanInterfaceHostsDataSource{}

func NewLanInterfaceHostsDataSource() datasource.DataSource {
	return &LanInterfaceHostsDataSource{}
}

type LanInterfaceHostsDataSource struct {
	client client.Client
}

type LanInterfaceHostsDataSourceModel struct {
	InterfaceName  types.String `tfsdk:"interface"`
	Hosts types.Set `tfsdk:"hosts"`
}

func (a *LanInterfaceHostsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lan_interface_hosts"
}

func (a *LanInterfaceHostsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get information about the hosts of a LAN interface.",
		Attributes: map[string]schema.Attribute{
			"interface": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the interface",
			},
			"hosts": schema.SetAttribute{
				Computed:            true,
				MarkdownDescription: "List of hosts",
				ElementType:         types.ObjectType{
					AttrTypes: models.LanHostL2IdentModel{}.AttrTypes(),
				},
			},
		},
	}
}

func (a *LanInterfaceHostsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (a *LanInterfaceHostsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LanInterfaceHostsDataSourceModel

	resp.Diagnostics.Append(resp.State.Get(ctx, &data)...)

	name := data.InterfaceName.ValueString()

	hosts, err := a.client.GetLanInterface(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get hosts of a LAN interface",
			fmt.Sprintf("Failed to get hosts of a LAN interface at %q: %s", name, err),
		)
		return
	}

	hostsElements := []attr.Value{}
	for _, host := range hosts {
		hostModel := LanInterfaceHostDataSourceModel{}
		hostModel.fromClientType(host)
		hostsElements = append(hostsElements, hostModel.ToObjectValue())
	}
	data.Hosts = basetypes.NewSetValueMust(basetypes.ObjectType{
		AttrTypes: models.LanHostL2IdentModel{}.AttrTypes(),
	}, hostsElements)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
