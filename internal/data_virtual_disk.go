package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nikolalohinski/free-go/client"
	"github.com/nikolalohinski/terraform-provider-freebox/internal/models"
)

var _ datasource.DataSource = &VirtualDiskDataSource{}

func NewVirtualDiskDataSource() datasource.DataSource {
	return &VirtualDiskDataSource{}
}

type VirtualDiskDataSource struct {
	client client.Client
}

type VirtualDiskDataSourceModel struct {
	Path        types.String `tfsdk:"path"`
	Type        types.String `tfsdk:"type"`
	ActualSize  types.Int64  `tfsdk:"actual_size"`  // Space used by virtual image on disk. This is how much filesystem space is consumed on the box.
	VirtualSize types.Int64  `tfsdk:"virtual_size"` // Size of virtual disk. This is the size the disk will appear inside the VM.
}

func (a *VirtualDiskDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_disk"
}

func (a *VirtualDiskDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get information about a virtual disk.",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Path to the virtual disk",
				Validators: []validator.String{
					models.FilePathValidator(),
				},
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Type of virtual disk",
				Validators: []validator.String{
					models.DiskTypeValidator(),
				},
			},
			"actual_size": schema.Int64Attribute{
				Computed:           true,
				MarkdownDescription: "Space in bytes used by the virtual image on disk. This is how much filesystem space is consumed on the box.",
				Validators: []validator.Int64{
					models.DiskSizeValidator(),
				},
			},
			"virtual_size": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Size in bytes of the virtual disk. This is the size the disk will appear inside the VM.",
				Validators: []validator.Int64{
					models.VirtualDiskSizeValidator(),
				},
			},
		},
	}
}

func (a *VirtualDiskDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (a *VirtualDiskDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VirtualDiskDataSourceModel

	resp.Diagnostics.Append(resp.State.Get(ctx, &data)...)

	path := data.Path.ValueString()

	diskInfo, err := a.client.GetVirtualDiskInfo(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get virtual disk info",
			fmt.Sprintf("Failed to get virtual disk info at %q: %s", path, err),
		)
		return
	}

	data.Type = types.StringValue(diskInfo.Type)
	data.ActualSize = types.Int64Value(diskInfo.ActualSize)
	data.VirtualSize = types.Int64Value(diskInfo.VirtualSize)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
