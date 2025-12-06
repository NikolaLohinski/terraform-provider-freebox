package internal

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

var (
	_ resource.ResourceWithImportState    = &portForwardingResource{}
	_ resource.ResourceWithValidateConfig = &portForwardingResource{}
)

func NewPortForwardingResource() resource.Resource {
	return &portForwardingResource{}
}

// virtualMachineResource defines the resource implementation.
type portForwardingResource struct {
	client client.Client
}

type portForwardingLanHostL2IdentModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

func (o portForwardingLanHostL2IdentModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "ID of the L2 ident",
		},
		"type": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Type of the L2 ident",
		},
	}
}

func (o portForwardingLanHostL2IdentModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   types.StringType,
		"type": types.StringType,
	}
}

func (o portForwardingLanHostL2IdentModel) FromClientType(l2ident freeboxTypes.L2Ident) basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(o.AttrTypes(), map[string]attr.Value{
		"id":   basetypes.NewStringValue(l2ident.ID),
		"type": basetypes.NewStringValue(string(l2ident.Type)),
	})
}

type portForwardingLanHostL3ConnectivityModel struct {
	Address   types.String `tfsdk:"address"`
	Active    types.Bool   `tfsdk:"active"`
	Reachable types.Bool   `tfsdk:"reachable"`
	Type      types.String `tfsdk:"type"`
}

func (o portForwardingLanHostL3ConnectivityModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"address": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Address of the L3 connectivity",
		},
		"active": schema.BoolAttribute{
			Optional:            true,
			MarkdownDescription: "Whether the L3 connectivity is active",
		},
		"reachable": schema.BoolAttribute{
			Optional:            true,
			MarkdownDescription: "Whether the L3 connectivity is reachable",
		},
		"type": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Type of the L3 connectivity",
		},
	}
}

func (o portForwardingLanHostL3ConnectivityModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"address":   types.StringType,
		"active":    types.BoolType,
		"reachable": types.BoolType,
		"type":      types.StringType,
	}
}

func (o portForwardingLanHostL3ConnectivityModel) FromClientType(connectivity freeboxTypes.LanHostL3Connectivity) basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(o.AttrTypes(), map[string]attr.Value{
		"address":   basetypes.NewStringValue(connectivity.Address),
		"active":    basetypes.NewBoolValue(connectivity.Active),
		"reachable": basetypes.NewBoolValue(connectivity.Reachable),
		"type":      basetypes.NewStringValue(string(connectivity.Type)),
	})
}

type portForwardingLanHostNameModel struct {
	Name   types.String `tfsdk:"name"`
	Source types.String `tfsdk:"source"`
}

func (o portForwardingLanHostNameModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Name of the host",
		},
		"source": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Source of the host",
		},
	}
}

func (o portForwardingLanHostNameModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":   types.StringType,
		"source": types.StringType,
	}
}

func (o portForwardingLanHostNameModel) FromClientType(name freeboxTypes.HostName) basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(o.AttrTypes(), map[string]attr.Value{
		"name":   basetypes.NewStringValue(name.Name),
		"source": basetypes.NewStringValue(name.Source),
	})
}

type portForwardingLanHostNetworkControlModel struct {
	ProfileID   types.Int64  `tfsdk:"profile_id"`
	Name        types.String `tfsdk:"name"`
	CurrentMode types.String `tfsdk:"current_mode"`
}

func (o portForwardingLanHostNetworkControlModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"profile_id": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "ID of the profile this device is associated with",
		},
		"name": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Name of the profile this device is associated with",
		},
		"current_mode": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Mode described in Network Control Object",
		},
	}
}

func (o portForwardingLanHostNetworkControlModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"profile_id":   types.Int64Type,
		"name":         types.StringType,
		"current_mode": types.StringType,
	}
}

func (o portForwardingLanHostNetworkControlModel) FromClientType(networkControl freeboxTypes.LanHostNetworkControl) basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(o.AttrTypes(), map[string]attr.Value{
		"profile_id":   basetypes.NewInt64Value(int64(networkControl.ProfileID)),
		"name":         basetypes.NewStringValue(networkControl.Name),
		"current_mode": basetypes.NewStringValue(networkControl.CurrentMode),
	})
}

type portForwardingLanHostModel struct {
	ID                   types.String  `tfsdk:"id"`
	Active               types.Bool    `tfsdk:"active"`
	Reachable            types.Bool    `tfsdk:"reachable"`
	Persistent           types.Bool    `tfsdk:"persistent"`
	PrimaryNameManual    types.Bool    `tfsdk:"primary_name_manual"`
	VendorName           types.String  `tfsdk:"vendor_name"`
	HostType             types.String  `tfsdk:"host_type"`
	Interface            types.String  `tfsdk:"interface"`
	FirstActivitySeconds types.Float64 `tfsdk:"first_activity_seconds"`
	PrimaryName          types.String  `tfsdk:"primary_name"`
	DefaultName          types.String  `tfsdk:"default_name"`
	L2Ident              types.Object  `tfsdk:"l2ident"`
	L3Connectivities     types.List    `tfsdk:"l3connectivities"`
	Names                types.List    `tfsdk:"names"`
	NetworkControl       types.Object  `tfsdk:"network_control"`
}

func (o portForwardingLanHostModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "ID of the host",
		},
		"active": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "If true the host sends traffic to the Freebox",
		},
		"reachable": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "If true the host can receive traffic from the Freebox",
		},
		"persistent": schema.BoolAttribute{
			Optional:            true,
			MarkdownDescription: "If true the host is always shown even if it has not been active since the Freebox startup",
		},
		"primary_name_manual": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "If true the primary name has been set manually",
		},
		"vendor_name": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Host vendor name (from the mac address)",
		},
		"host_type": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "When possible, the Freebox will try to guess the host_type, but you can manually override this to the correct value",
		},
		"interface": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Interface of the host",
		},
		"first_activity_seconds": schema.Float64Attribute{
			Computed:            true,
			MarkdownDescription: "First time the host sent traffic, or null if it wasn’t seen before this field was added.",
		},
		"primary_name": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Host primary name (chosen from the list of available names, or manually set by user)",
		},
		"default_name": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Default name of the host",
		},
		"l2ident": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Layer 2 network id and its type",
			Attributes:          portForwardingLanHostL2IdentModel{}.ResourceAttributes(),
		},
		"l3connectivities": schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "List of available layer 3 network connections",
			NestedObject: schema.NestedAttributeObject{
				Attributes: portForwardingLanHostL3ConnectivityModel{}.ResourceAttributes(),
			},
		},
		"names": schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "List of available names, and their source",
			NestedObject: schema.NestedAttributeObject{
				Attributes: portForwardingLanHostNameModel{}.ResourceAttributes(),
			},
		},
		"network_control": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "If device is associated with a profile, contains profile summary.",
			Attributes:          portForwardingLanHostNetworkControlModel{}.ResourceAttributes(),
		},
	}
}

func (o portForwardingLanHostModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                     types.StringType,
		"active":                 types.BoolType,
		"reachable":              types.BoolType,
		"persistent":             types.BoolType,
		"primary_name_manual":    types.BoolType,
		"vendor_name":            types.StringType,
		"host_type":              types.StringType,
		"interface":              types.StringType,
		"first_activity_seconds": types.Float64Type,
		"primary_name":           types.StringType,
		"default_name":           types.StringType,
		"l2ident":                types.ObjectType{}.WithAttributeTypes(portForwardingLanHostL2IdentModel{}.AttrTypes()),
		"l3connectivities":       types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(portForwardingLanHostL3ConnectivityModel{}.AttrTypes())),
		"names":                  types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(portForwardingLanHostNameModel{}.AttrTypes())),
		"network_control":        types.ObjectType{}.WithAttributeTypes(portForwardingLanHostNetworkControlModel{}.AttrTypes()),
	}
}

func (o portForwardingLanHostModel) FromClientType(host freeboxTypes.LanInterfaceHost) basetypes.ObjectValue {
	l3connectivities := make([]attr.Value, len(host.L3Connectivities))
	for i, connectivity := range host.L3Connectivities {
		l3connectivities[i] = portForwardingLanHostL3ConnectivityModel{}.FromClientType(connectivity)
	}

	names := make([]attr.Value, len(host.Names))
	for i, name := range host.Names {
		names[i] = portForwardingLanHostNameModel{}.FromClientType(name)
	}

	var firstActivityValue basetypes.Float64Value

	if host.FirstActivity.IsZero() {
		firstActivityValue = basetypes.NewFloat64Null()
	} else {
		firstActivityValue = basetypes.NewFloat64Value(float64(host.FirstActivity.UnixMicro() / 1000000))
	}

	var networkControlValue basetypes.ObjectValue
	if host.NetworkControl != nil {
		networkControlValue = portForwardingLanHostNetworkControlModel{}.FromClientType(*host.NetworkControl)
	} else {
		networkControlValue = basetypes.NewObjectNull(portForwardingLanHostNetworkControlModel{}.AttrTypes())
	}

	return basetypes.NewObjectValueMust(o.AttrTypes(), map[string]attr.Value{
		"id":                     basetypes.NewStringValue(host.ID),
		"active":                 basetypes.NewBoolValue(host.Active),
		"reachable":              basetypes.NewBoolValue(host.Reachable),
		"persistent":             basetypes.NewBoolValue(host.Persistent),
		"primary_name_manual":    basetypes.NewBoolValue(host.PrimaryNameManual),
		"vendor_name":            basetypes.NewStringValue(host.VendorName),
		"host_type":              basetypes.NewStringValue(string(host.Type)),
		"interface":              basetypes.NewStringValue(host.Interface),
		"first_activity_seconds": firstActivityValue,
		"primary_name":           basetypes.NewStringValue(host.PrimaryName),
		"default_name":           basetypes.NewStringValue(host.DefaultName),
		"l2ident":                portForwardingLanHostL2IdentModel{}.FromClientType(host.L2Ident),
		"l3connectivities":       basetypes.NewListValueMust(types.ObjectType{}.WithAttributeTypes(portForwardingLanHostL3ConnectivityModel{}.AttrTypes()), l3connectivities),
		"names":                  basetypes.NewListValueMust(types.ObjectType{}.WithAttributeTypes(portForwardingLanHostNameModel{}.AttrTypes()), names),
		"network_control":        networkControlValue,
	})
}

// virtualMachineModel describes the resource data model.
type portForwardingModel struct {
	ID             types.Int64  `tfsdk:"id"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	IPProtocol     types.String `tfsdk:"ip_protocol"`
	PortRangeStart types.Int64  `tfsdk:"port_range_start"`
	PortRangeEnd   types.Int64  `tfsdk:"port_range_end"`
	TargetPort     types.Int64  `tfsdk:"target_port"`
	SourceIP       types.String `tfsdk:"source_ip"`
	TargetIP       types.String `tfsdk:"target_ip"`
	Comment        types.String `tfsdk:"comment"`
	Hostname       types.String `tfsdk:"hostname"`
	LanHost        types.Object `tfsdk:"host"`
}

func (p *portForwardingModel) toPayload() freeboxTypes.PortForwardingRulePayload {
	enabled := p.Enabled.ValueBool()
	payload := freeboxTypes.PortForwardingRulePayload{
		Enabled:      &enabled,
		IPProtocol:   p.IPProtocol.ValueString(),
		LanIP:        p.TargetIP.ValueString(),
		WanPortStart: p.PortRangeStart.ValueInt64(),
	}
	if p.PortRangeEnd.IsNull() || p.PortRangeEnd.IsUnknown() {
		payload.WanPortEnd = p.PortRangeStart.ValueInt64() // Default to the same value as the start
	} else {
		payload.WanPortEnd = p.PortRangeEnd.ValueInt64()
	}

	if p.TargetPort.IsNull() || p.TargetPort.IsUnknown() {
		payload.LanPort = p.PortRangeStart.ValueInt64() // Default to the same value as the start
	} else {
		payload.LanPort = p.TargetPort.ValueInt64()
	}

	if !p.SourceIP.IsNull() && !p.SourceIP.IsUnknown() {
		payload.SourceIP = p.SourceIP.ValueString()
	}

	if !p.Comment.IsNull() && !p.Comment.IsUnknown() {
		payload.Comment = p.Comment.ValueString()
	}

	return payload
}

func (p *portForwardingModel) fromClientType(rule freeboxTypes.PortForwardingRule) {
	p.ID = basetypes.NewInt64Value(rule.ID)
	if rule.Enabled != nil {
		p.Enabled = basetypes.NewBoolValue(*rule.Enabled)
	} else {
		p.Enabled = basetypes.NewBoolValue(false)
	}
	p.IPProtocol = basetypes.NewStringValue(rule.IPProtocol)
	p.TargetIP = basetypes.NewStringValue(rule.LanIP)

	p.PortRangeStart = basetypes.NewInt64Value(rule.WanPortStart)
	p.PortRangeEnd = basetypes.NewInt64Value(rule.WanPortEnd)
	p.TargetPort = basetypes.NewInt64Value(rule.LanPort)

	if rule.Comment != "" {
		p.Comment = basetypes.NewStringValue(rule.Comment)
	}

	if rule.SourceIP != "" {
		p.SourceIP = basetypes.NewStringValue(rule.SourceIP)
	}

	p.Hostname = basetypes.NewStringValue(rule.Hostname)

	if rule.Host != nil {
		p.LanHost = portForwardingLanHostModel{}.FromClientType(*rule.Host)
	} else {
		p.LanHost = basetypes.NewObjectNull(portForwardingLanHostModel{}.AttrTypes())
	}
}

func (v *portForwardingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_forwarding"
}

func (v *portForwardingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a port forwarding between a local network host and the Freebox Internet Gateway.\n\nNote: To have access to the full port range, you need to enable the Full Stack IP (see [the Freebox documentation (in French)](https://assistance.free.fr/articles/1758)).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the port forwarding",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Status of the forwarding",
				Required:            true,
			},
			"ip_protocol": schema.StringAttribute{
				MarkdownDescription: "Protocol to handle",
				Required:            true,
				Validators: []validator.String{stringvalidator.OneOf(
					string(freeboxTypes.TCP),
					string(freeboxTypes.UDP),
				)},
			},
			"port_range_start": schema.Int64Attribute{
				MarkdownDescription: "Start boundary of the port range to forward. The range is inclusive.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 65353),
				},
			},
			"port_range_end": schema.Int64Attribute{
				MarkdownDescription: "End boundary of the port range to forward. If not set, it will default to the same value as `port_range_start`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 65353),
				},
			},
			"target_port": schema.Int64Attribute{
				MarkdownDescription: "The target port range to forward to. If not set, it will default to the same value as `port_range_start`. Only available for a range of 1 port.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 65353),
				},
			},
			"source_ip": schema.StringAttribute{
				MarkdownDescription: "Local IP of the local port forwarding target. If left unset or set to 0.0.0.0, the rule will apply to any incoming IP",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("0.0.0.0"),
			},
			"target_ip": schema.StringAttribute{
				MarkdownDescription: "Local IP of the local port forwarding target",
				Required:            true,
			},
			"comment": schema.StringAttribute{
				MarkdownDescription: "Additional comment associated with the rule",
				Optional:            true,
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Name of the target host in the local network",
				Computed:            true,
			},
			"host": schema.SingleNestedAttribute{
				MarkdownDescription: "LAN host information",
				Computed:            true,
				Attributes:          portForwardingLanHostModel{}.ResourceAttributes(),
			},
		},
	}
}

func (v *portForwardingResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data portForwardingModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !data.TargetPort.IsNull() && !data.TargetPort.IsUnknown() {
		end := data.PortRangeEnd.ValueInt64()
		start := data.PortRangeStart.ValueInt64()
		if end != 0 && end != start {
			if data.TargetPort.ValueInt64() != start {
				resp.Diagnostics.AddError(
					"Invalid target port", "Using a target port is only allowed when the port range is a single port")
				return
			}
		}
	}
}

func (v *portForwardingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (v *portForwardingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model portForwardingModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := v.client.CreatePortForwardingRule(ctx, model.toPayload())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create port forwarding rule",
			err.Error(),
		)
		return
	}

	model.fromClientType(response)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *portForwardingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model portForwardingModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	response, err := v.client.GetPortForwardingRule(ctx, model.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get port forwarding rule",
			err.Error(),
		)
		return
	}
	model.fromClientType(response)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *portForwardingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model portForwardingModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	response, err := v.client.UpdatePortForwardingRule(ctx, model.ID.ValueInt64(), model.toPayload())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update port forwarding rule",
			err.Error(),
		)
		return
	}

	model.fromClientType(response)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *portForwardingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model portForwardingModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)

	if err := v.client.DeletePortForwardingRule(ctx, model.ID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete port forwarding rule",
			err.Error(),
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (v *portForwardingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.Atoi(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected the import identifier with to be an int64 but got: %s", req.ID),
		)
	}
	attrPath := path.Root("id")
	if attrPath.Equal(path.Empty()) {
		resp.Diagnostics.AddError(
			"Resource import passthrough missing attribute path",
			"This is always an error in the provider. Please report the following to the provider developer:\n\n"+
				"Resource ImportState method call to ImportStatePassthroughIntID path must be set to a valid attribute path that can accept a int value.",
		)
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, attrPath, id)...)
}
