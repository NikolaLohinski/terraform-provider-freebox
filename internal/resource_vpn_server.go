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
	ID         types.String `tfsdk:"id"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	ServerPort types.Int64  `tfsdk:"server_port"`
	ServerIP   types.String `tfsdk:"server_ip"`
	ServerMask types.String `tfsdk:"server_mask"`
	PushDHCP   types.Bool   `tfsdk:"push_dhcp"`
	CA         types.String `tfsdk:"ca"`
}

func (m *vpnServerModel) toPayload() freeboxTypes.OpenVPNServerConfig {
	payload := freeboxTypes.OpenVPNServerConfig{
		Enabled:    m.Enabled.ValueBool(),
		ServerPort: m.ServerPort.ValueInt64(),
		PushDHCP:   m.PushDHCP.ValueBool(),
	}
	if !m.ServerIP.IsNull() && !m.ServerIP.IsUnknown() {
		payload.ServerIP = m.ServerIP.ValueString()
	}
	if !m.ServerMask.IsNull() && !m.ServerMask.IsUnknown() {
		payload.ServerMask = m.ServerMask.ValueString()
	}
	return payload
}

func (m *vpnServerModel) fromClientType(config freeboxTypes.OpenVPNServerConfig) {
	m.ID = basetypes.NewStringValue("openvpn")
	m.Enabled = basetypes.NewBoolValue(config.Enabled)
	m.ServerPort = basetypes.NewInt64Value(config.ServerPort)
	m.ServerIP = basetypes.NewStringValue(config.ServerIP)
	m.ServerMask = basetypes.NewStringValue(config.ServerMask)
	m.PushDHCP = basetypes.NewBoolValue(config.PushDHCP)
	m.CA = basetypes.NewStringValue(config.CA)
}

func (v *vpnServerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpn_server"
}

func (v *vpnServerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the OpenVPN server configuration on the Freebox. This is a singleton resource: only one OpenVPN server exists per Freebox. Destroying this resource disables the VPN server.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Fixed identifier for the singleton OpenVPN server resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the OpenVPN server is enabled",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"server_port": schema.Int64Attribute{
				MarkdownDescription: "UDP port the OpenVPN server listens on (default 1194)",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1194),
			},
			"server_ip": schema.StringAttribute{
				MarkdownDescription: "VPN subnet IP address (e.g. \"10.8.0.0\")",
				Optional:            true,
				Computed:            true,
			},
			"server_mask": schema.StringAttribute{
				MarkdownDescription: "VPN subnet mask (e.g. \"255.255.255.0\")",
				Optional:            true,
				Computed:            true,
			},
			"push_dhcp": schema.BoolAttribute{
				MarkdownDescription: "Whether to push DHCP settings to clients",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"ca": schema.StringAttribute{
				MarkdownDescription: "CA certificate in PEM format (read-only, set by Freebox)",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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

	response, err := v.client.UpdateOpenVPNServerConfig(ctx, model.toPayload())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to configure OpenVPN server",
			err.Error(),
		)
		return
	}

	model.fromClientType(response)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *vpnServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model vpnServerModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := v.client.GetOpenVPNServerConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read OpenVPN server config",
			err.Error(),
		)
		return
	}

	model.fromClientType(response)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *vpnServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model vpnServerModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := v.client.UpdateOpenVPNServerConfig(ctx, model.toPayload())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update OpenVPN server config",
			err.Error(),
		)
		return
	}

	model.fromClientType(response)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *vpnServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model vpnServerModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := model.toPayload()
	payload.Enabled = false

	if _, err := v.client.UpdateOpenVPNServerConfig(ctx, payload); err != nil {
		resp.Diagnostics.AddError(
			"Failed to disable OpenVPN server",
			err.Error(),
		)
	}
}

func (v *vpnServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), "openvpn")...)
}
