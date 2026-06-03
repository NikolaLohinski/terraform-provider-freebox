package models

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)


type LanHostL2IdentModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

func (o LanHostL2IdentModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "ID of the L2 ident",
		},
		"type": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Type of the L2 ident",
		},
	}
}

func (o LanHostL2IdentModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   types.StringType,
		"type": types.StringType,
	}
}

func (o LanHostL2IdentModel) FromClientType(l2ident freeboxTypes.L2Ident) basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(o.AttrTypes(), map[string]attr.Value{
		"id":   basetypes.NewStringValue(l2ident.ID),
		"type": basetypes.NewStringValue(string(l2ident.Type)),
	})
}

type LanHostModel struct {
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

func (o LanHostModel) ResourceAttributes() map[string]schema.Attribute {
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
			Attributes:          LanHostL2IdentModel{}.ResourceAttributes(),
		},
		"l3connectivities": schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "List of available layer 3 network connections",
			NestedObject: schema.NestedAttributeObject{
				Attributes: LanHostL3ConnectivityModel{}.ResourceAttributes(),
			},
		},
		"names": schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "List of available names, and their source",
			NestedObject: schema.NestedAttributeObject{
				Attributes: LanHostNameModel{}.ResourceAttributes(),
			},
		},
		"network_control": schema.SingleNestedAttribute{
			Computed:            true,
			MarkdownDescription: "If device is associated with a profile, contains profile summary.",
			Attributes:          LanHostNetworkControlModel{}.ResourceAttributes(),
		},
	}
}

func (o LanHostModel) AttrTypes() map[string]attr.Type {
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
		"l2ident":                types.ObjectType{}.WithAttributeTypes(LanHostL2IdentModel{}.AttrTypes()),
		"l3connectivities":       types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(LanHostL3ConnectivityModel{}.AttrTypes())),
		"names":                  types.ListType{}.WithElementType(types.ObjectType{}.WithAttributeTypes(LanHostNameModel{}.AttrTypes())),
		"network_control":        types.ObjectType{}.WithAttributeTypes(LanHostNetworkControlModel{}.AttrTypes()),
	}
}

func (o LanHostModel) FromClientType(host freeboxTypes.LanInterfaceHost) basetypes.ObjectValue {
	l3connectivities := make([]attr.Value, len(host.L3Connectivities))
	for i, connectivity := range host.L3Connectivities {
		l3connectivities[i] = LanHostL3ConnectivityModel{}.FromClientType(connectivity)
	}

	names := make([]attr.Value, len(host.Names))
	for i, name := range host.Names {
		names[i] = LanHostNameModel{}.FromClientType(name)
	}

	var firstActivityValue basetypes.Float64Value

	if host.FirstActivity.IsZero() {
		firstActivityValue = basetypes.NewFloat64Null()
	} else {
		firstActivityValue = basetypes.NewFloat64Value(float64(host.FirstActivity.UnixMicro()) / 1_000_000.0)
	}

	var networkControlValue basetypes.ObjectValue
	if host.NetworkControl != nil {
		networkControlValue = LanHostNetworkControlModel{}.FromLanHostNetworkControl(*host.NetworkControl)
	} else {
		networkControlValue = basetypes.NewObjectNull(LanHostNetworkControlModel{}.AttrTypes())
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
		"l2ident":                LanHostL2IdentModel{}.FromClientType(host.L2Ident),
		"l3connectivities":       basetypes.NewListValueMust(types.ObjectType{}.WithAttributeTypes(LanHostL3ConnectivityModel{}.AttrTypes()), l3connectivities),
		"names":                  basetypes.NewListValueMust(types.ObjectType{}.WithAttributeTypes(LanHostNameModel{}.AttrTypes()), names),
		"network_control":        networkControlValue,
	})
}

type LanHostL3ConnectivityModel struct {
	Address   types.String `tfsdk:"address"`
	Active    types.Bool   `tfsdk:"active"`
	Reachable types.Bool   `tfsdk:"reachable"`
	Type      types.String `tfsdk:"type"`
}

func (o LanHostL3ConnectivityModel) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"address": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Address of the L3 connectivity",
		},
		"active": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether the L3 connectivity is active",
		},
		"reachable": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether the L3 connectivity is reachable",
		},
		"type": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Type of the L3 connectivity",
		},
	}
}

func (o LanHostL3ConnectivityModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"address":   types.StringType,
		"active":    types.BoolType,
		"reachable": types.BoolType,
		"type":      types.StringType,
	}
}

func (o LanHostL3ConnectivityModel) FromClientType(connectivity freeboxTypes.LanHostL3Connectivity) basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(o.AttrTypes(), map[string]attr.Value{
		"address":   basetypes.NewStringValue(connectivity.Address),
		"active":    basetypes.NewBoolValue(connectivity.Active),
		"reachable": basetypes.NewBoolValue(connectivity.Reachable),
		"type":      basetypes.NewStringValue(string(connectivity.Type)),
	})
}

type LanHostNetworkControlModel struct {
	ProfileID   types.Int64  `tfsdk:"profile_id"`
	Name        types.String `tfsdk:"name"`
	CurrentMode types.String `tfsdk:"current_mode"`
}

func (o LanHostNetworkControlModel) ResourceAttributes() map[string]schema.Attribute {
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

func (o LanHostNetworkControlModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"profile_id":   types.Int64Type,
		"name":         types.StringType,
		"current_mode": types.StringType,
	}
}

func (o LanHostNetworkControlModel) FromLanHostNetworkControl(networkControl freeboxTypes.LanHostNetworkControl) basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(o.AttrTypes(), map[string]attr.Value{
		"profile_id":   basetypes.NewInt64Value(int64(networkControl.ProfileID)),
		"name":         basetypes.NewStringValue(networkControl.Name),
		"current_mode": basetypes.NewStringValue(networkControl.CurrentMode),
	})
}

type LanHostNameModel struct {
	Name   types.String `tfsdk:"name"`
	Source types.String `tfsdk:"source"`
}

func (o LanHostNameModel) ResourceAttributes() map[string]schema.Attribute {
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

func (o LanHostNameModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":   types.StringType,
		"source": types.StringType,
	}
}

func (o LanHostNameModel) FromClientType(name freeboxTypes.HostName) basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(o.AttrTypes(), map[string]attr.Value{
		"name":   basetypes.NewStringValue(name.Name),
		"source": basetypes.NewStringValue(name.Source),
	})
}
