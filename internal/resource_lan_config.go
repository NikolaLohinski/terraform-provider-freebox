package internal

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

var (
	_ resource.Resource                = &lanConfigResource{}
	_ resource.ResourceWithImportState = &lanConfigResource{}
)

func NewLanConfigResource() resource.Resource {
	return &lanConfigResource{}
}

type lanConfigResource struct {
	client client.Client
}

type lanConfigResourceModel struct {
	ID          types.String `tfsdk:"id"`
	IP          types.String `tfsdk:"ip"`
	Name        types.String `tfsdk:"name"`
	NameDNS     types.String `tfsdk:"name_dns"`
	NameMDNS    types.String `tfsdk:"name_mdns"`
	NameNetBIOS types.String `tfsdk:"name_netbios"`
	Mode        types.String `tfsdk:"mode"`
}

func (m *lanConfigResourceModel) fromClientType(config freeboxTypes.LanConfig) {
	m.ID = basetypes.NewStringValue("lan_config")
	m.IP = basetypes.NewStringValue(config.IP)
	m.Name = basetypes.NewStringValue(config.Name)
	m.NameDNS = basetypes.NewStringValue(config.NameDNS)
	m.NameMDNS = basetypes.NewStringValue(config.NameMDNS)
	m.NameNetBIOS = basetypes.NewStringValue(config.NameNetBIOS)
	m.Mode = basetypes.NewStringValue(config.Mode)
}

// applyToConfig overlays the non-null, non-unknown fields from the model onto base,
// which preserves current device values for any field not explicitly set in config.
func (m *lanConfigResourceModel) applyToConfig(base freeboxTypes.LanConfig) freeboxTypes.LanConfig {
	if !m.IP.IsNull() && !m.IP.IsUnknown() {
		base.IP = m.IP.ValueString()
	}
	if !m.Name.IsNull() && !m.Name.IsUnknown() {
		base.Name = m.Name.ValueString()
	}
	if !m.NameDNS.IsNull() && !m.NameDNS.IsUnknown() {
		base.NameDNS = m.NameDNS.ValueString()
	}
	if !m.NameMDNS.IsNull() && !m.NameMDNS.IsUnknown() {
		base.NameMDNS = m.NameMDNS.ValueString()
	}
	if !m.NameNetBIOS.IsNull() && !m.NameNetBIOS.IsUnknown() {
		base.NameNetBIOS = m.NameNetBIOS.ValueString()
	}
	if !m.Mode.IsNull() && !m.Mode.IsUnknown() {
		base.Mode = m.Mode.ValueString()
	}
	return base
}

func (v *lanConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lan_config"
}

func (v *lanConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the LAN configuration of the Freebox. This is a singleton resource: the LAN configuration always exists and cannot be deleted. Destroying this resource is a no-op.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Fixed identifier for the singleton LAN configuration resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Freebox Server IPv4 address",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$`),
						"Must be a valid IPv4 address",
					),
				},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Freebox Server name",
			},
			"name_dns": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Freebox Server DNS name",
			},
			"name_mdns": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Freebox Server mDNS name",
			},
			"name_netbios": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Freebox Server NetBIOS name",
				Validators: []validator.String{
					stringvalidator.All(
						stringvalidator.LengthAtMost(16),
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[^\/:*?"<>|]+$`),
							`Cannot contain forbidden characters: \ / : * ? < > |`,
						),
					),
				},
			},
			"mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "LAN mode (`router` or `bridge`)",
				Validators: []validator.String{
					stringvalidator.OneOf(
						freeboxTypes.LanModeRouter,
						freeboxTypes.LanModeBridge,
					),
				},
			},
		},
	}
}

func (v *lanConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (v *lanConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model lanConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := v.client.GetLanConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read LAN configuration", fmt.Sprintf("Failed to read LAN configuration: %s", err))
		return
	}

	result, err := v.client.UpdateLanConfig(ctx, model.applyToConfig(current))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update LAN configuration", fmt.Sprintf("Failed to update LAN configuration: %s", err))
		return
	}

	model.fromClientType(result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *lanConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model lanConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := v.client.GetLanConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get LAN configuration", fmt.Sprintf("Failed to get LAN configuration: %s", err))
		return
	}

	model.fromClientType(result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *lanConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model lanConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := v.client.GetLanConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read LAN configuration", fmt.Sprintf("Failed to read LAN configuration: %s", err))
		return
	}

	result, err := v.client.UpdateLanConfig(ctx, model.applyToConfig(current))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update LAN configuration", fmt.Sprintf("Failed to update LAN configuration: %s", err))
		return
	}

	model.fromClientType(result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *lanConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// LAN configuration cannot be deleted; this is a no-op.
}

func (v *lanConfigResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), "lan_config")...)
}
