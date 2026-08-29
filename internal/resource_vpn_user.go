package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
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
	Login           types.String `tfsdk:"login"`
	Type            types.String `tfsdk:"type"`
	Password        types.String `tfsdk:"password"`
	IPReservation   types.String `tfsdk:"ip_reservation"`
	Keepalive       types.Int64  `tfsdk:"keepalive"`
	PSK             types.Bool   `tfsdk:"psk"`
	OVPNConfig      types.String `tfsdk:"ovpn_config"`
	WireguardConfig types.String `tfsdk:"wireguard_config"`
}

func (m *vpnUserModel) toPayload() freeboxTypes.VPNUserPayload {
	payload := freeboxTypes.VPNUserPayload{
		Login:         m.Login.ValueString(),
		Type:          freeboxTypes.VPNUserType(m.Type.ValueString()),
		Password:      m.Password.ValueString(),
		IPReservation: m.IPReservation.ValueString(),
	}
	if m.Type.ValueString() == string(freeboxTypes.VPNUserTypeWireguard) {
		payload.ConfWireguard = &freeboxTypes.VPNUserWireguardConfig{
			Keepalive: m.Keepalive.ValueInt64(),
			PSK:       m.PSK.ValueBool(),
		}
	}
	return payload
}

// fromClientType does not copy Password; it is write-only on the Freebox API,
// so the value from config/state is left untouched.
func (m *vpnUserModel) fromClientType(user freeboxTypes.VPNUser) {
	m.Login = basetypes.NewStringValue(user.Login)
	m.Type = basetypes.NewStringValue(string(user.Type))
	m.IPReservation = basetypes.NewStringValue(user.IPReservation)
	if user.ConfWireguard != nil {
		m.Keepalive = basetypes.NewInt64Value(user.ConfWireguard.Keepalive)
		m.PSK = basetypes.NewBoolValue(user.ConfWireguard.PSK)
	} else {
		m.Keepalive = basetypes.NewInt64Null()
		m.PSK = basetypes.NewBoolNull()
	}
}

func (v *vpnUserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_user"
}

func (v *vpnUserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a VPN user account on the Freebox (`standard` or `wireguard`). The `ovpn_config` attribute contains the ready-to-use OpenVPN client configuration file content and is only populated for `standard` users.",
		Attributes: map[string]schema.Attribute{
			"login": schema.StringAttribute{
				MarkdownDescription: "VPN username (immutable after creation)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "VPN user type: `standard` or `wireguard` (immutable after creation)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(string(freeboxTypes.VPNUserTypeStandard)),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "VPN password (not relevant when type is wireguard)",
				Optional:            true,
				Sensitive:           true,
			},
			"ip_reservation": schema.StringAttribute{
				MarkdownDescription: "Reserved IPv4 for this VPN user within the VPN range. Required when type is wireguard",
				Optional:            true,
				Computed:            true,
			},
			"keepalive": schema.Int64Attribute{
				MarkdownDescription: "Interval in seconds at which keepalive packets are sent (only relevant when type is wireguard)",
				Optional:            true,
				Computed:            true,
			},
			"psk": schema.BoolAttribute{
				MarkdownDescription: "Enable optional preshared-key (only relevant when type is wireguard)",
				Optional:            true,
				Computed:            true,
			},
			"ovpn_config": schema.StringAttribute{
				MarkdownDescription: "OpenVPN client configuration file content (.ovpn format). Ready to import into any OpenVPN client. Only populated when type is standard.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"wireguard_config": schema.StringAttribute{
				MarkdownDescription: "WireGuard client configuration file content (wg-quick .conf format). Ready to import into any WireGuard client. Only populated when type is wireguard. Requires the app to have the 'settings' permission.",
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

	switch model.Type.ValueString() {
	case string(freeboxTypes.VPNUserTypeStandard):
		ovpnConfig, err := v.client.GetVPNUserClientConfig(ctx, model.Login.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to get VPN client config",
				err.Error(),
			)
			return
		}

		model.OVPNConfig = basetypes.NewStringValue(ovpnConfig)
		model.WireguardConfig = basetypes.NewStringNull()
	case string(freeboxTypes.VPNUserTypeWireguard):
		wireguardConfig, err := v.client.GetVPNUserWireguardConfig(ctx, model.Login.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to get VPN wireguard config",
				err.Error(),
			)
			return
		}

		model.OVPNConfig = basetypes.NewStringNull()
		model.WireguardConfig = basetypes.NewStringValue(wireguardConfig)
	default:
		model.OVPNConfig = basetypes.NewStringNull()
		model.WireguardConfig = basetypes.NewStringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *vpnUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model vpnUserModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	existingOVPNConfig := model.OVPNConfig
	existingWireguardConfig := model.WireguardConfig

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

	// The Freebox API embeds a freshly generated certificate/key in the
	// response on every call to GetVPNUserClientConfig/GetVPNUserWireguardConfig,
	// so re-fetching here would make the config look like it changes on every
	// refresh even though the server-side config is stable (it only changes
	// when the user is recreated). Preserve the value already in state, same
	// as Update does, and only fetch it when state doesn't have one yet (e.g.
	// after import).
	switch model.Type.ValueString() {
	case string(freeboxTypes.VPNUserTypeStandard):
		model.WireguardConfig = basetypes.NewStringNull()
		if existingOVPNConfig.ValueString() != "" {
			model.OVPNConfig = existingOVPNConfig
		} else {
			ovpnConfig, err := v.client.GetVPNUserClientConfig(ctx, model.Login.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Failed to get VPN client config",
					err.Error(),
				)
				return
			}

			model.OVPNConfig = basetypes.NewStringValue(ovpnConfig)
		}
	case string(freeboxTypes.VPNUserTypeWireguard):
		model.OVPNConfig = basetypes.NewStringNull()
		if existingWireguardConfig.ValueString() != "" {
			model.WireguardConfig = existingWireguardConfig
		} else {
			wireguardConfig, err := v.client.GetVPNUserWireguardConfig(ctx, model.Login.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Failed to get VPN wireguard config",
					err.Error(),
				)
				return
			}

			model.WireguardConfig = basetypes.NewStringValue(wireguardConfig)
		}
	default:
		model.OVPNConfig = basetypes.NewStringNull()
		model.WireguardConfig = basetypes.NewStringNull()
	}

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
	model.WireguardConfig = state.WireguardConfig

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
