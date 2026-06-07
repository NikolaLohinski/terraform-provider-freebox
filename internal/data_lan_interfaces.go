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
)

var _ datasource.DataSource = &lanInterfacesDataSource{}

func NewLanInterfacesDataSource() datasource.DataSource {
	return &lanInterfacesDataSource{}
}

type lanInterfacesDataSource struct {
	client client.Client
}

type lanInterfacesModel struct {
	Interfaces types.List `tfsdk:"interfaces"`
}

type lanInfoModel struct {
	Name      types.String `tfsdk:"name"`
	HostCount types.Int64  `tfsdk:"host_count"`
}

func (m lanInfoModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":       types.StringType,
		"host_count": types.Int64Type,
	}
}

func (d *lanInterfacesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lan_interfaces"
}

func (d *lanInterfacesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List the LAN interfaces available on the Freebox.",
		Attributes: map[string]schema.Attribute{
			"interfaces": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of LAN interfaces",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Interface name (e.g. `pub`, `priv`)",
						},
						"host_count": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Number of hosts seen on this interface",
						},
					},
				},
			},
		},
	}
}

func (d *lanInterfacesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *lanInterfacesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	lanInfos, err := d.client.ListLanInterfaceInfo(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list LAN interfaces", fmt.Sprintf("Failed to list LAN interfaces: %s", err))
		return
	}

	attrTypes := lanInfoModel{}.attrTypes()
	items := make([]attr.Value, len(lanInfos))
	for i, info := range lanInfos {
		items[i] = basetypes.NewObjectValueMust(attrTypes, map[string]attr.Value{
			"name":       types.StringValue(info.Name),
			"host_count": types.Int64Value(int64(info.HostCount)),
		})
	}

	list, diags := basetypes.NewListValue(types.ObjectType{AttrTypes: attrTypes}, items)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &lanInterfacesModel{Interfaces: list})...)
}
