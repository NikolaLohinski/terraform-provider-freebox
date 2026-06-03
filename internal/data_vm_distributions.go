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

var _ datasource.DataSource = &vmDistributionsDataSource{}

func NewVMDistributionsDataSource() datasource.DataSource {
	return &vmDistributionsDataSource{}
}

type vmDistributionsDataSource struct {
	client client.Client
}

type vmDistributionsModel struct {
	Distributions types.List `tfsdk:"distributions"`
}

type vmDistributionModel struct {
	Hash types.String `tfsdk:"hash"`
	OS   types.String `tfsdk:"os"`
	URL  types.String `tfsdk:"url"`
	Name types.String `tfsdk:"name"`
}

func (m vmDistributionModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"hash": types.StringType,
		"os":   types.StringType,
		"url":  types.StringType,
		"name": types.StringType,
	}
}

func (d *vmDistributionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine_distributions"
}

func (d *vmDistributionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List the OS distributions available for virtual machine provisioning.",
		Attributes: map[string]schema.Attribute{
			"distributions": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of available OS distributions",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"hash": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "SHA256 hash of the distribution image",
						},
						"os": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Operating system identifier (e.g. `debian`, `ubuntu`)",
						},
						"url": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Download URL for the distribution image",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Human-readable name of the distribution",
						},
					},
				},
			},
		},
	}
}

func (d *vmDistributionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *vmDistributionsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	distributions, err := d.client.GetVirtualMachineDistributions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list VM distributions", fmt.Sprintf("Failed to list VM distributions: %s", err))
		return
	}

	attrTypes := vmDistributionModel{}.attrTypes()
	items := make([]attr.Value, len(distributions))
	for i, dist := range distributions {
		items[i] = basetypes.NewObjectValueMust(attrTypes, map[string]attr.Value{
			"hash": types.StringValue(dist.Hash),
			"os":   types.StringValue(dist.OS),
			"url":  types.StringValue(dist.URL),
			"name": types.StringValue(dist.Name),
		})
	}

	list, diags := basetypes.NewListValue(types.ObjectType{AttrTypes: attrTypes}, items)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &vmDistributionsModel{Distributions: list})...)
}
