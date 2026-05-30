package internal

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/nikolalohinski/free-go/client"
)

var (
	_ datasource.DataSource = &dhcpLeaseDataSource{}
)

func NewDhcpLeaseDataSource() datasource.DataSource {
	return &dhcpLeaseDataSource{}
}

// dhcpLeaseResource defines the resource implementation.
type dhcpLeaseDataSource struct {
	client client.Client
}

func (v *dhcpLeaseDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dhcp_lease"
}

func (v *dhcpLeaseDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a virtual machine instance within a Freebox. See the [Freebox blog](https://dev.freebox.fr/blog/?p=5450) for additional details",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the DHCP lease",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"ip": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "IP address to assign to the target device",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`), "Must be a valid IPv4 address"),
				},
			},
			"mac": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "MAC address of the target device",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"hostname": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Hostname of the target device",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"comment": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Comment of the DHCP lease",
			},
		},
	}
}

func (v *dhcpLeaseDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (v *dhcpLeaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model dhcpLeaseModel

	if diags := req.Config.Get(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	dhcpLease, err := v.client.GetDHCPStaticLease(ctx, strings.ToLower(model.Mac.ValueString()))
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.Code == "noent" {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Failed to get DHCP lease",
			err.Error(),
		)
		return
	}

	if diags := model.fromDHCPStaticLeaseInfo(dhcpLease); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if diags := resp.State.Set(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
}
