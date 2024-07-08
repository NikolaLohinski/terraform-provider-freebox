package internal

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.ResourceWithImportState = &portForwardingResource{}
)

func NewPortForwardingResource() resource.Resource {
	return &portForwardingResource{}
}

// virtualMachineResource defines the resource implementation.
type portForwardingResource struct {
	client client.Client
}

// virtualMachineModel describes the resource data model.
type portForwardingModel struct {
	ID             types.Int64  `tfsdk:"id"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	IPProtocol     types.String `tfsdk:"ip_protocol"`
	PortRangeStart types.Int64  `tfsdk:"port_range_start"`
	PortRangeEnd   types.Int64  `tfsdk:"port_range_end"`
	SourceIP       types.String `tfsdk:"source_ip"`
	TargetIP       types.String `tfsdk:"target_ip"`
	Comment        types.String `tfsdk:"comment"`
	Hostname       types.String `tfsdk:"hostname"`
}

func (p *portForwardingModel) toPayload() freeboxTypes.PortForwardingRulePayload {
	enabled := p.Enabled.ValueBool()
	payload := freeboxTypes.PortForwardingRulePayload{
		Enabled:      &enabled,
		IPProtocol:   p.IPProtocol.ValueString(),
		LanIP:        p.TargetIP.ValueString(),
		LanPort:      p.PortRangeStart.ValueInt64(),
		WanPortStart: p.PortRangeStart.ValueInt64(),
		WanPortEnd:   p.PortRangeEnd.ValueInt64(),
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
	if rule.Comment != "" {
		p.Comment = basetypes.NewStringValue(rule.Comment)
	}
	if rule.SourceIP != "" {
		p.SourceIP = basetypes.NewStringValue(rule.SourceIP)
	}
	p.Hostname = basetypes.NewStringValue(rule.Hostname)
}

func (v *portForwardingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_forwarding"
}

func (v *portForwardingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a port forwarding between a local network host and the Freebox Internet Gateway",
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
				MarkdownDescription: "Start boundary of the port range to forward",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
					int64validator.AtMost(65353),
				},
			},
			"port_range_end": schema.Int64Attribute{
				MarkdownDescription: "End boundary of the port range to forward",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
					int64validator.AtMost(65353),
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
		},
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
