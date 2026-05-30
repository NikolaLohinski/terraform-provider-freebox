package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
)

var (
	_ datasource.DataSource = &dhcpLeasesDataSource{}
)

func NewDhcpLeasesDataSource() datasource.DataSource {
	return &dhcpLeasesDataSource{}
}

// dhcpLeaseResource defines the resource implementation.
type dhcpLeasesDataSource struct {
	client client.Client
}

type DhcpLeasesModel struct {
	Leases types.List `tfsdk:"leases"`
}

func (v *dhcpLeasesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dhcp_leases"
}

func (v *dhcpLeasesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get the list of DHCP static leases.",
		Attributes: map[string]schema.Attribute{
			"leases": schema.ListAttribute{
				Computed:            true,
				MarkdownDescription: "List of DHCP leases",
				ElementType: types.ObjectType{
					AttrTypes: dhcpLeaseModel{}.AttrTypes(),
				},
			},
		},
	}
}

func (v *dhcpLeasesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	v.client = client
}

func (v *dhcpLeasesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model DhcpLeasesModel

	if diags := req.Config.Get(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	dhcpLeases, err := v.client.ListDHCPStaticLease(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get DHCP lease",
			err.Error(),
		)
		return
	}

	leases := make([]attr.Value, len(dhcpLeases))
	for i, dhcpLease := range dhcpLeases {
		var lease dhcpLeaseModel
		diags := lease.fromDHCPStaticLeaseInfo(dhcpLease)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		leases[i] = lease.ToObjectValue()
	}

	var diags diag.Diagnostics

	model.Leases, diags = basetypes.NewListValueFrom(ctx, types.ObjectType{
		AttrTypes: dhcpLeaseModel{}.AttrTypes(),
	}, leases)

	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if diags := resp.State.Set(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
}
