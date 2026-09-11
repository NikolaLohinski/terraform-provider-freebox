package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

var (
	_ resource.ResourceWithImportState = &vpnServerResource{}
)

func NewVPNServerResource() resource.Resource {
	return &vpnServerResource{}
}

type vpnServerResource struct {
	client client.Client
}

type vpnServerModel struct {
	ID              types.String `tfsdk:"id"`
	VPNID           types.String `tfsdk:"vpn_id"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	EnableIPv4      types.Bool   `tfsdk:"enable_ipv4"`
	EnableIPv6      types.Bool   `tfsdk:"enable_ipv6"`
	Port            types.Int64  `tfsdk:"port"`
	MinPort         types.Int64  `tfsdk:"min_port"`
	MaxPort         types.Int64  `tfsdk:"max_port"`
	Cipher          types.String `tfsdk:"cipher"`
	DisableFragment types.Bool   `tfsdk:"disable_fragment"`
	UseTCP          types.Bool   `tfsdk:"use_tcp"`
	MTU             types.Int64  `tfsdk:"mtu"`
	IPStart         types.String `tfsdk:"ip_start"`
	IPEnd           types.String `tfsdk:"ip_end"`
	IP6Start        types.String `tfsdk:"ip6_start"`
	IP6End          types.String `tfsdk:"ip6_end"`
}

func (m *vpnServerModel) toPayload() freeboxTypes.VPNServerConfig {
	payload := freeboxTypes.VPNServerConfig{
		Enabled:    m.Enabled.ValueBool(),
		EnableIPv4: m.EnableIPv4.ValueBool(),
		EnableIPv6: m.EnableIPv6.ValueBool(),
		Port:       m.Port.ValueInt64(),
	}
	if m.VPNID.ValueString() == string(freeboxTypes.VPNServerIDOpenVPNRouted) || m.VPNID.ValueString() == string(freeboxTypes.VPNServerIDOpenVPNBridge) {
		payload.ConfOpenVPN = &freeboxTypes.OpenVPNConfig{
			Cipher:          freeboxTypes.OpenVPNCipher(m.Cipher.ValueString()),
			DisableFragment: m.DisableFragment.ValueBool(),
			UseTCP:          m.UseTCP.ValueBool(),
		}
	}
	if m.VPNID.ValueString() == string(freeboxTypes.VPNServerIDWireguard) {
		payload.ConfWireguard = &freeboxTypes.WireguardConfig{
			MTU: m.MTU.ValueInt64(),
		}
	}
	return payload
}

func (m *vpnServerModel) fromClientType(id freeboxTypes.VPNServerID, config freeboxTypes.VPNServerConfig) {
	m.ID = basetypes.NewStringValue(string(id))
	m.VPNID = basetypes.NewStringValue(string(id))
	m.Enabled = basetypes.NewBoolValue(config.Enabled)
	m.EnableIPv4 = basetypes.NewBoolValue(config.EnableIPv4)
	m.EnableIPv6 = basetypes.NewBoolValue(config.EnableIPv6)
	m.Port = basetypes.NewInt64Value(config.Port)
	m.MinPort = basetypes.NewInt64Value(config.MinPort)
	m.MaxPort = basetypes.NewInt64Value(config.MaxPort)
	if config.ConfOpenVPN != nil {
		m.Cipher = basetypes.NewStringValue(string(config.ConfOpenVPN.Cipher))
		m.DisableFragment = basetypes.NewBoolValue(config.ConfOpenVPN.DisableFragment)
		m.UseTCP = basetypes.NewBoolValue(config.ConfOpenVPN.UseTCP)
	} else {
		m.Cipher = basetypes.NewStringNull()
		m.DisableFragment = basetypes.NewBoolNull()
		m.UseTCP = basetypes.NewBoolNull()
	}
	if config.ConfWireguard != nil {
		m.MTU = basetypes.NewInt64Value(config.ConfWireguard.MTU)
	} else {
		m.MTU = basetypes.NewInt64Null()
	}
	m.IPStart = basetypes.NewStringValue(config.IPStart)
	m.IPEnd = basetypes.NewStringValue(config.IPEnd)
	m.IP6Start = basetypes.NewStringValue(config.IP6Start)
	m.IP6End = basetypes.NewStringValue(config.IP6End)
}

func (v *vpnServerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_server"
}

func (v *vpnServerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages one of the built-in VPN server configurations on the Freebox (`pptp`, `openvpn_routed`, `openvpn_bridge` or `wireguard`). This is a singleton resource per `vpn_id`: it configures an existing server slot rather than creating a new one. Destroying this resource disables that VPN server.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of the VPN server, same value as `vpn_id`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vpn_id": schema.StringAttribute{
				MarkdownDescription: "Which VPN server to manage: `pptp`, `openvpn_routed`, `openvpn_bridge` or `wireguard`",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(string(freeboxTypes.VPNServerIDOpenVPNRouted)),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the VPN server is enabled",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"enable_ipv4": schema.BoolAttribute{
				MarkdownDescription: "Enable IPv4 (not relevant for openvpn_bridge and pptp)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"enable_ipv6": schema.BoolAttribute{
				MarkdownDescription: "Enable IPv6 (not relevant for openvpn_bridge and pptp)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "Server port (only editable when vpn_id is openvpn_routed or openvpn_bridge)",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1194),
			},
			"min_port": schema.Int64Attribute{
				MarkdownDescription: "Read-only lower bound of the available port range",
				Computed:            true,
			},
			"max_port": schema.Int64Attribute{
				MarkdownDescription: "Read-only upper bound of the available port range",
				Computed:            true,
			},
			"cipher": schema.StringAttribute{
				MarkdownDescription: "Cipher used by the OpenVPN server: `blowfish`, `aes128` or `aes256` (only relevant when vpn_id is openvpn_routed or openvpn_bridge)",
				Optional:            true,
				Computed:            true,
			},
			"disable_fragment": schema.BoolAttribute{
				MarkdownDescription: "Disable the fragment configuration option (only relevant when vpn_id is openvpn_routed or openvpn_bridge)",
				Optional:            true,
				Computed:            true,
			},
			"use_tcp": schema.BoolAttribute{
				MarkdownDescription: "Use TCP instead of UDP (only relevant when vpn_id is openvpn_routed or openvpn_bridge)",
				Optional:            true,
				Computed:            true,
			},
			"mtu": schema.Int64Attribute{
				MarkdownDescription: "Maximum transmission unit of the WireGuard interface (only relevant when vpn_id is wireguard)",
				Optional:            true,
				Computed:            true,
			},
			"ip_start": schema.StringAttribute{
				MarkdownDescription: "Read-only IPv4 pool range start for clients",
				Computed:            true,
			},
			"ip_end": schema.StringAttribute{
				MarkdownDescription: "Read-only IPv4 pool range end for clients",
				Computed:            true,
			},
			"ip6_start": schema.StringAttribute{
				MarkdownDescription: "Read-only IPv6 pool range start for clients",
				Computed:            true,
			},
			"ip6_end": schema.StringAttribute{
				MarkdownDescription: "Read-only IPv6 pool range end for clients",
				Computed:            true,
			},
		},
	}
}

func (v *vpnServerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (v *vpnServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model vpnServerModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := freeboxTypes.VPNServerID(model.VPNID.ValueString())

	response, err := v.client.UpdateVPNServerConfig(ctx, id, model.toPayload())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to configure VPN server",
			err.Error(),
		)
		return
	}

	model.fromClientType(id, response)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *vpnServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model vpnServerModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := freeboxTypes.VPNServerID(model.VPNID.ValueString())

	response, err := v.client.GetVPNServerConfig(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read VPN server config",
			err.Error(),
		)
		return
	}

	model.fromClientType(id, response)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *vpnServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model vpnServerModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := freeboxTypes.VPNServerID(model.VPNID.ValueString())

	response, err := v.client.UpdateVPNServerConfig(ctx, id, model.toPayload())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update VPN server config",
			err.Error(),
		)
		return
	}

	model.fromClientType(id, response)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *vpnServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model vpnServerModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := freeboxTypes.VPNServerID(model.VPNID.ValueString())

	payload := model.toPayload()
	payload.Enabled = false

	if _, err := v.client.UpdateVPNServerConfig(ctx, id, payload); err != nil {
		resp.Diagnostics.AddError(
			"Failed to disable VPN server",
			err.Error(),
		)
	}
}

func (v *vpnServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vpn_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
