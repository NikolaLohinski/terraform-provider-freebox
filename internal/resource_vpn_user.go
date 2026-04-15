package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

var (
	_ resource.ResourceWithImportState = &vpnUserResource{}
)

func NewVPNUserResource() resource.Resource {
	return &vpnUserResource{}
}

type vpnUserResource struct {
	client client.Client
}

type vpnUserModel struct {
	Login       types.String `tfsdk:"login"`
	Password    types.String `tfsdk:"password"`
	Description types.String `tfsdk:"description"`
	OVPNConfig  types.String `tfsdk:"ovpn_config"`
}

func (m *vpnUserModel) toPayload() freeboxTypes.VPNUserPayload {
	return freeboxTypes.VPNUserPayload{
		Login:       m.Login.ValueString(),
		Password:    m.Password.ValueString(),
		Description: m.Description.ValueString(),
	}
}

func (m *vpnUserModel) fromClientType(user freeboxTypes.VPNUser) {
	m.Login = basetypes.NewStringValue(user.Login)
	// Password is write-only on the Freebox API; preserve the value from config
	if user.Description != "" {
		m.Description = basetypes.NewStringValue(user.Description)
	} else {
		m.Description = basetypes.NewStringNull()
	}
}

func (v *vpnUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_user"
}

func (v *vpnUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a VPN user account on the Freebox OpenVPN server. The `ovpn_config` attribute contains the ready-to-use OpenVPN client configuration file content.",
		Attributes: map[string]schema.Attribute{
			"login": schema.StringAttribute{
				MarkdownDescription: "VPN username (immutable after creation)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "VPN password",
				Required:            true,
				Sensitive:           true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the VPN user",
				Optional:            true,
			},
			"ovpn_config": schema.StringAttribute{
				MarkdownDescription: "OpenVPN client configuration file content (.ovpn format). Ready to import into any OpenVPN client.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (v *vpnUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	v.client = c
}

func (v *vpnUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model vpnUserModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := v.client.CreateVPNUser(ctx, model.toPayload())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create VPN user",
			err.Error(),
		)
		return
	}

	model.fromClientType(user)

	ovpnConfig, err := v.client.GetVPNUserClientConfig(ctx, model.Login.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get VPN client config",
			err.Error(),
		)
		return
	}

	model.OVPNConfig = basetypes.NewStringValue(ovpnConfig)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *vpnUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model vpnUserModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := v.client.GetVPNUser(ctx, model.Login.ValueString())
	if err != nil {
		if err == client.ErrVPNUserNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Failed to read VPN user",
			err.Error(),
		)
		return
	}

	model.fromClientType(user)

	ovpnConfig, err := v.client.GetVPNUserClientConfig(ctx, model.Login.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get VPN client config",
			err.Error(),
		)
		return
	}

	model.OVPNConfig = basetypes.NewStringValue(ovpnConfig)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *vpnUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model vpnUserModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the existing ovpn_config from state (it only changes when the server config changes)
	var state vpnUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := v.client.UpdateVPNUser(ctx, model.Login.ValueString(), model.toPayload())
	if err != nil {
		if err == client.ErrVPNUserNotFound {
			resp.Diagnostics.AddError(
				"VPN user not found",
				fmt.Sprintf("VPN user %q was not found on the Freebox. It may have been deleted outside of Terraform.", model.Login.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Failed to update VPN user",
			err.Error(),
		)
		return
	}

	model.fromClientType(user)
	model.OVPNConfig = state.OVPNConfig

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *vpnUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model vpnUserModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := v.client.DeleteVPNUser(ctx, model.Login.ValueString()); err != nil {
		if err == client.ErrVPNUserNotFound {
			return // Already deleted, nothing to do
		}
		resp.Diagnostics.AddError(
			"Failed to delete VPN user",
			err.Error(),
		)
	}
}

func (v *vpnUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("login"), req.ID)...)
}
