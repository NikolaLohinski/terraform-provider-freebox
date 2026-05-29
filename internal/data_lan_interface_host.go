package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/nikolalohinski/terraform-provider-freebox/internal/models"
)

var _ datasource.DataSource = &LanInterfaceHostDataSource{}

func NewLanInterfaceHostDataSource() datasource.DataSource {
	return &LanInterfaceHostDataSource{}
}

type LanInterfaceHostDataSource struct {
	client client.Client
}

type LanInterfaceHostDataSourceModel struct {
	Interface types.String `tfsdk:"interface"`
	HostId    types.String `tfsdk:"host_id"`
	L2Ident   types.Object `tfsdk:"l2ident"`
}

func (o LanInterfaceHostDataSourceModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"interface": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Name of the interface",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"host_id": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "ID of the host",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"l2ident": schema.ObjectAttribute{
			Required:            true,
			MarkdownDescription: "L2 ident of the interface",
			AttributeTypes: models.LanHostL2IdentModel{}.AttrTypes(),
		},
	}
}

func (o LanInterfaceHostDataSourceModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"interface": types.StringType,
		"host_id":   types.StringType,
		"l2ident":   types.ObjectType{}.WithAttributeTypes(models.LanHostL2IdentModel{}.AttrTypes()),
	}
}

func (a *LanInterfaceHostDataSourceModel) fromClientType(host freeboxTypes.LanInterfaceHost) {
	a.Interface = types.StringValue(host.Interface)
	a.HostId = types.StringValue(host.ID)
	a.L2Ident = models.LanHostL2IdentModel{}.FromClientType(host.L2Ident)
}

func (a *LanInterfaceHostDataSourceModel) ToObjectValue() basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(a.AttrTypes(), map[string]attr.Value{
		"interface": a.Interface,
		"host_id":   a.HostId,
		"l2ident":   a.L2Ident,
	})
}

func (a *LanInterfaceHostDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lan_interface_host"
}

func (a *LanInterfaceHostDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get information about a host of a LAN interface.",
		Attributes: LanInterfaceHostDataSourceModel{}.ResourceAttributes(),
	}
}

func (a *LanInterfaceHostDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (a *LanInterfaceHostDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LanInterfaceHostDataSourceModel

	resp.Diagnostics.Append(resp.State.Get(ctx, &data)...)

	interfaceName := data.Interface.ValueString()
	hostId := data.HostId.ValueString()

	host, err := a.client.GetLanInterfaceHost(ctx, interfaceName, hostId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get host of a LAN interface",
			fmt.Sprintf("Failed to get host of a LAN interface at %q: %s", interfaceName, hostId, err),
		)
		return
	}

	data.L2Ident = models.LanHostL2IdentModel{}.FromClientType(host.L2Ident)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
