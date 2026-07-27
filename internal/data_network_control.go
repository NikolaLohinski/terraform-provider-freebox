package internal

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/nikolalohinski/terraform-provider-freebox/internal/models"
)

var (
	_ datasource.DataSource = &networkControlDataSource{}
)

func NewNetworkControlDataSource() datasource.DataSource {
	return &networkControlDataSource{}
}

// networkControlDataSource defines the resource implementation.
type networkControlDataSource struct {
	client client.Client
}

type NetworkControlModel struct {
	ProfileID types.Int64 `tfsdk:"profile_id"`
	NextChange timetypes.GoDuration `tfsdk:"next_change"`
	OverrideMode types.String `tfsdk:"override_mode"`
	CurrentMode types.String `tfsdk:"current_mode"`
	RuleMode types.String `tfsdk:"rule_mode"`
	OverrideUntil types.Int64 `tfsdk:"override_until"`
	Override types.Bool `tfsdk:"override"`
	Macs types.List `tfsdk:"macs"`
	Hosts types.List `tfsdk:"hosts"`
	Resolution types.Int64 `tfsdk:"resolution"`
	CustomDayRanges types.List `tfsdk:"custom_day_ranges"`
}

func (o *NetworkControlModel) FromNetworkControlInfo(networkControl freeboxTypes.NetworkControlInfo) (diagnostics diag.Diagnostics) {
	o.ProfileID = basetypes.NewInt64Value(networkControl.ProfileID)
	if networkControl.NextChange > 0 {
		o.NextChange = timetypes.NewGoDurationValue(time.Duration(networkControl.NextChange) * time.Second)
	} else {
		o.NextChange = timetypes.NewGoDurationNull()
	}
	o.OverrideMode = basetypes.NewStringValue(string(networkControl.OverrideMode))
	o.CurrentMode = basetypes.NewStringValue(string(networkControl.CurrentMode))
	o.RuleMode = basetypes.NewStringValue(string(networkControl.RuleMode))
	o.OverrideUntil = basetypes.NewInt64Value(int64(networkControl.OverrideUntil))
	o.Override = basetypes.NewBoolValue(networkControl.Override)
	macs := make([]attr.Value, len(networkControl.Macs))
	for i, mac := range networkControl.Macs {
		macs[i] = basetypes.NewStringValue(mac)
	}
	o.Macs = basetypes.NewListValueMust(types.StringType, macs)
	hosts := make([]attr.Value, len(networkControl.Hosts))
	for i, host := range networkControl.Hosts {
		hosts[i] = models.LanHostModel{}.FromClientType(host)
	}
	o.Hosts = basetypes.NewListValueMust(types.ObjectType{
		AttrTypes: models.LanHostModel{}.AttrTypes(),
	}, hosts)
	o.Resolution = basetypes.NewInt64Value(int64(networkControl.Resolution))
	customDayRanges := make([]attr.Value, len(networkControl.CustomDayRanges))
	for i, dayRange := range networkControl.CustomDayRanges {
		customDayRanges[i] = basetypes.NewStringValue(dayRange)
	}
	o.CustomDayRanges = basetypes.NewListValueMust(types.StringType, customDayRanges)
	return nil
}

func (v *networkControlDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_control"
}

func (v *networkControlDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a virtual machine instance within a Freebox. See the [Freebox blog](https://dev.freebox.fr/blog/?p=5450) for additional details",
		Attributes: map[string]schema.Attribute{
			"profile_id": schema.Int64Attribute{
				Required: true,
				MarkdownDescription: "Id of the profile this network control is associated with.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"next_change": schema.StringAttribute{
				Computed:            true,
				CustomType:          timetypes.GoDurationType{},
				MarkdownDescription: "UNIX timestamp of next rule change in seconds.",
			},
			"override_mode": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Mode of current override.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(freeboxTypes.RuleModeAllowed),
						string(freeboxTypes.RuleModeDenied),
						string(freeboxTypes.RuleModeWebOnly),
					),
				},
			},
			"current_mode": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Mode in use. If override is true, it will be override_mode, otherwise it’s the mode from the rules attached to this NetworkControl.",
				Validators: []validator.String{
					models.NetworkControlValidator(),
				},
			},
			"rule_mode": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Mode that would be in use if there was no override. Depends only on rules, and is useful to determine what will happen when override is lifted.",
				Validators: []validator.String{
					models.NetworkControlValidator(),
				},
			},
			"override_until": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Unix timestamp in seconds when override ends. Relevant when override is true. Set at 0 for unlimited.",
			},
			"override": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether there’s an override at the moment.",
			},
			"macs": schema.ListAttribute{
				Computed:            true,
				MarkdownDescription: "List of mac adresses associated with this profile’s network control.",
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(regexp.MustCompile(`^[0-9A-Fa-f]{2}(:[0-9A-Fa-f]{2}){5}$`), "Must be a valid MAC address"),
					),
				},
			},
			"hosts": schema.ListAttribute{
				Computed:            true,
				MarkdownDescription: "List of Lan Host objects associated with this profile’s network control. Derived from the macs array.",
				ElementType: types.ObjectType{
					AttrTypes: models.LanHostModel{}.AttrTypes(),
				},
				Validators: []validator.List{
					listvalidator.UniqueValues(),
				},
			},
			"resolution": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Control resolution per day of this network control.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"custom_day_ranges": schema.ListAttribute{
				Computed:            true,
				MarkdownDescription: "List of custom day range, each custom day range represents a group of days for which you want to use a different planning than other week days.",
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						models.NetworkControlDayRangeValidator(),
					),
				},
			},
		},
	}
}

func (v *networkControlDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (v *networkControlDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model NetworkControlModel

	if diags := req.Config.Get(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	networkControl, err := v.client.GetNetworkControl(ctx, model.ProfileID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get network control",
			err.Error(),
		)
		return
	}

	if diags := model.FromNetworkControlInfo(networkControl); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if diags := resp.State.Set(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
}
