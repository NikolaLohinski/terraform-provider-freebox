package internal

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

var (
	_ resource.Resource                = &dhcpLeaseResource{}
	_ resource.ResourceWithImportState = &dhcpLeaseResource{}
)

func NewDhcpLeaseResource() resource.Resource {
	return &dhcpLeaseResource{}
}

// dhcpLeaseResource defines the resource implementation.
type dhcpLeaseResource struct {
	client client.Client
}

// dhcpLeaseModel describes the resource data model.
type dhcpLeaseModel struct {
	ID       types.String          `tfsdk:"id"`
	Hostname types.String          `tfsdk:"hostname"`
	IP       types.String          `tfsdk:"ip"`
	Comment  types.String          `tfsdk:"comment"`
	Mac      basetypes.StringValue `tfsdk:"mac"`
}

func (v *dhcpLeaseModel) fromLanInterfaceHost(ctx context.Context, c client.Client, lanInterfaceHost freeboxTypes.LanInterfaceHost) (diagnostics diag.Diagnostics) {
	dhcpLease, err := c.GetDHCPStaticLease(ctx, lanInterfaceHost.ID)
	if err != nil {
		diagnostics.AddError("Failed to read DHCP lease after write", err.Error())
		return
	}
	return v.fromDHCPStaticLeaseInfo(dhcpLease)
}

func (v *dhcpLeaseModel) fromDHCPStaticLeaseInfo(dhcpLeaseInfo freeboxTypes.DHCPStaticLeaseInfo) (diagnostics diag.Diagnostics) {
	v.ID = basetypes.NewStringValue(dhcpLeaseInfo.ID)
	v.Mac = basetypes.NewStringValue(dhcpLeaseInfo.Mac)
	v.Hostname = basetypes.NewStringValue(dhcpLeaseInfo.Hostname)
	v.IP = basetypes.NewStringValue(dhcpLeaseInfo.IP)
	v.Comment = basetypes.NewStringValue(dhcpLeaseInfo.Comment)
	return diagnostics
}

func (v dhcpLeaseModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":       types.StringType,
		"hostname": types.StringType,
		"ip":       types.StringType,
		"comment":  types.StringType,
		"mac":      types.StringType,
	}
}

func (v *dhcpLeaseModel) ToObjectValue() basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(v.AttrTypes(), map[string]attr.Value{
		"id":       v.ID,
		"hostname": v.Hostname,
		"ip":       v.IP,
		"comment":  v.Comment,
		"mac":      v.Mac,
	})
}

func (v *dhcpLeaseModel) toClientPayload() (payload freeboxTypes.DHCPStaticLeasePayload, diagnostics diag.Diagnostics) {
	payload.Mac = v.Mac.ValueString()
	payload.Hostname = v.Hostname.ValueString()
	payload.IP = v.IP.ValueString()
	payload.Comment = v.Comment.ValueString()
	return payload, nil
}

func (v *dhcpLeaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dhcp_lease"
}

func (v *dhcpLeaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DHCP static lease on the Freebox.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the DHCP lease",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
						var id basetypes.StringValue
						resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
						if id.ValueString() == "" {
							var mac basetypes.StringValue
							resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("mac"), &mac)...)
							if mac.IsNull() || mac.IsUnknown() {
								return
							}
							req.Plan.SetAttribute(ctx, path.Root("id"), basetypes.NewStringValue(strings.ToUpper(mac.ValueString())))
						}
					}, "", "If the state value is empty, set the id from the mac address"),
				},
			},
			"ip": schema.StringAttribute{
				Required:            true,
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hostname": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "Hostname of the target device",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"comment": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Comment of the DHCP lease",
			},
		},
	}
}

func (v *dhcpLeaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (v *dhcpLeaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model dhcpLeaseModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}

	payload, diagnostics := model.toClientPayload()
	if diagnostics.HasError() {
		resp.Diagnostics.Append(diagnostics...)
		return
	}

	lanInterfaceHost, err := v.client.CreateDHCPStaticLease(ctx, payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create DHCP static lease",
			err.Error(),
		)
		return
	}

	if d := model.fromLanInterfaceHost(ctx, v.client, lanInterfaceHost); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *dhcpLeaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model dhcpLeaseModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}

	dhcpLease, err := v.client.GetDHCPStaticLease(ctx, model.ID.ValueString())
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

	if d := model.fromDHCPStaticLeaseInfo(dhcpLease); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *dhcpLeaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var oldModel, newModel dhcpLeaseModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &newModel)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &oldModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, diagnostics := newModel.toClientPayload()
	if diagnostics.HasError() {
		resp.Diagnostics.Append(diagnostics...)
		return
	}

	lanInterfaceHost, err := v.client.UpdateDHCPStaticLease(ctx, oldModel.ID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update DHCP lease",
			err.Error(),
		)
		return
	}

	if d := newModel.fromLanInterfaceHost(ctx, v.client, lanInterfaceHost); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newModel)...)
}

func (v *dhcpLeaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model dhcpLeaseModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if err := v.client.DeleteDHCPStaticLease(ctx, model.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete DHCP lease",
			err.Error(),
		)
		return
	}
}

func (v *dhcpLeaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
