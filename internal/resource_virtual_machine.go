package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

var (
	_ resource.Resource                = &virtualMachineResource{}
	_ resource.ResourceWithImportState = &virtualMachineResource{}

	defaultTimeoutRead       = "5m"
	defaultTimeoutCreate     = "5m"
	defaultTimeoutUpdate     = "5m"
	defaultTimeoutDelete     = "5m"
	defaultTimeoutKill       = "30s"
	defaultTimeoutNetworking = "1m"

	errNetworkingTimeout = errors.New("NetworkingTimeoutError")
)

type virtualMachineStateChangeEvent struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

type networkBind struct {
	Interface string
	IPv6      []string
	IPv4      string
}

func NewVirtualMachineResource() resource.Resource {
	return &virtualMachineResource{}
}

// virtualMachineResource defines the resource implementation.
type virtualMachineResource struct {
	client client.Client
}

// virtualMachineModel describes the resource data model.
type virtualMachineModel struct {
	ID                types.Int64  `tfsdk:"id"`
	Mac               types.String `tfsdk:"mac"`
	Status            types.String `tfsdk:"status"`
	Name              types.String `tfsdk:"name"`
	DiskPath          types.String `tfsdk:"disk_path"`
	DiskType          types.String `tfsdk:"disk_type"`
	CDPath            types.String `tfsdk:"cd_path"`
	Memory            types.Int64  `tfsdk:"memory"`
	OS                types.String `tfsdk:"os"`
	VCPUs             types.Int64  `tfsdk:"vcpus"`
	EnableScreen      types.Bool   `tfsdk:"enable_screen"`
	BindUSBPorts      types.List   `tfsdk:"bind_usb_ports"`
	EnableCloudInit   types.Bool   `tfsdk:"enable_cloudinit"`
	CloudInitUserData types.String `tfsdk:"cloudinit_userdata"`
	CloudHostName     types.String `tfsdk:"cloudinit_hostname"`
	Timeouts          types.Object `tfsdk:"timeouts"`
	Networking        types.Set    `tfsdk:"networking"`
}

func (v *virtualMachineModel) fromClientType(virtualMachine freeboxTypes.VirtualMachine) (diagnostics diag.Diagnostics) {
	v.ID = basetypes.NewInt64Value(virtualMachine.ID)
	v.Mac = basetypes.NewStringValue(virtualMachine.Mac)
	v.Status = basetypes.NewStringValue(virtualMachine.Status)
	v.Name = basetypes.NewStringValue(virtualMachine.Name)
	v.DiskPath = basetypes.NewStringValue(string(virtualMachine.DiskPath))
	v.DiskType = basetypes.NewStringValue(string(virtualMachine.DiskType))
	v.Memory = basetypes.NewInt64Value(virtualMachine.Memory)
	v.VCPUs = basetypes.NewInt64Value(virtualMachine.VCPUs)
	v.CDPath = basetypes.NewStringValue(string(virtualMachine.CDPath))
	if virtualMachine.OS != "" {
		v.OS = basetypes.NewStringValue(virtualMachine.OS)
	}
	v.EnableScreen = basetypes.NewBoolValue(virtualMachine.EnableScreen)
	v.EnableCloudInit = basetypes.NewBoolValue(virtualMachine.EnableCloudInit)
	v.CloudInitUserData = basetypes.NewStringValue(string(virtualMachine.CloudInitUserData))
	v.CloudHostName = basetypes.NewStringValue(string(virtualMachine.CloudHostName))
	if len(virtualMachine.BindUSBPorts) > 0 {
		usbPorts := []attr.Value{}
		for _, port := range virtualMachine.BindUSBPorts {
			usbPorts = append(usbPorts, basetypes.NewStringValue(port))
		}
		v.BindUSBPorts, diagnostics = basetypes.NewListValue(types.StringType, usbPorts)
	}
	return diagnostics
}
func (v *virtualMachineModel) setNetworking(binds []networkBind) (diagnostics diag.Diagnostics) {
	networking := []attr.Value{}
	bindType := map[string]attr.Type{
		"ipv6": basetypes.SetType{
			ElemType: basetypes.StringType{},
		},
		"ipv4":      basetypes.StringType{},
		"interface": basetypes.StringType{},
	}
	for _, bind := range binds {
		if bind.Interface == "" {
			diagnostics.AddError("Incomplete networking information", "Missing interface name in network bind object")
			return
		}
		if bind.IPv4 == "" {
			diagnostics.AddError("Incomplete networking information", "Missing IPv4 in network bind object")
			return
		}
		ipv6Values := []attr.Value{}
		for _, address := range bind.IPv6 {
			ipv6Values = append(ipv6Values, basetypes.NewStringValue(address))
		}
		var ipv6 basetypes.SetValue
		ipv6, diagnostics = basetypes.NewSetValue(basetypes.StringType{}, ipv6Values)
		if diagnostics.HasError() {
			return
		}

		var bindObjectValue attr.Value
		bindObjectValue, diagnostics = basetypes.NewObjectValue(bindType, map[string]attr.Value{
			"interface": basetypes.NewStringValue(bind.Interface),
			"ipv4":      basetypes.NewStringValue(bind.IPv4),
			"ipv6":      ipv6,
		})
		if diagnostics.HasError() {
			return
		}
		networking = append(networking, bindObjectValue)
	}

	v.Networking, diagnostics = basetypes.NewSetValue(basetypes.ObjectType{
		AttrTypes: bindType,
	}, networking)
	return diagnostics
}

func (v *virtualMachineModel) toClientPayload(ctx context.Context) (payload freeboxTypes.VirtualMachinePayload, diagnostics diag.Diagnostics) {
	payload.Name = v.Name.ValueString()
	payload.DiskPath = freeboxTypes.Base64Path(v.DiskPath.ValueString())
	payload.DiskType = v.DiskType.ValueString()
	payload.CDPath = freeboxTypes.Base64Path(v.CDPath.ValueString())
	payload.Memory = v.Memory.ValueInt64()
	payload.VCPUs = v.VCPUs.ValueInt64()
	if !v.OS.IsNull() && !v.OS.IsUnknown() {
		payload.OS = v.OS.ValueString()
	}
	payload.CloudInitUserData = v.CloudInitUserData.ValueString()
	payload.CloudHostName = v.CloudHostName.ValueString()
	payload.EnableCloudInit = v.EnableCloudInit.ValueBool()
	payload.EnableScreen = v.EnableScreen.ValueBool()
	if !v.BindUSBPorts.IsNull() && !v.BindUSBPorts.IsUnknown() {
		return payload, v.BindUSBPorts.ElementsAs(ctx, &payload.BindUSBPorts, false)
	}
	return payload, nil
}

type timeoutsModel struct {
	Create     timetypes.GoDuration `tfsdk:"create"`
	Update     timetypes.GoDuration `tfsdk:"update"`
	Read       timetypes.GoDuration `tfsdk:"read"`
	Delete     timetypes.GoDuration `tfsdk:"delete"`
	Kill       timetypes.GoDuration `tfsdk:"kill"`
	Networking timetypes.GoDuration `tfsdk:"networking"`
}

func (v *virtualMachineResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine"
}

type virtualMachineStaticStatusModifier struct {
	status string
}

func (m virtualMachineStaticStatusModifier) Description(_ context.Context) string {
	return "The value of the virtual machine must be \"" + m.status + "\"."
}

func (m virtualMachineStaticStatusModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m virtualMachineStaticStatusModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	resp.PlanValue = basetypes.NewStringValue(m.status)
}

func (v *virtualMachineResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a virtual machine instance within a Freebox box. See the [Freebox blog](https://dev.freebox.fr/blog/?p=5450) for additional details",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the VM",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"mac": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "VM ethernet interface MAC address",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "VM status",
				PlanModifiers: []planmodifier.String{virtualMachineStaticStatusModifier{
					status: freeboxTypes.RunningStatus,
				}},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of this VM. Max 31 characters",
			},
			"disk_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Path to the hard disk image of this VM",
			},
			"disk_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Type of disk image",
				Validators: []validator.String{
					stringvalidator.OneOf([]string{freeboxTypes.QCow2Disk, freeboxTypes.RawDisk}...),
				},
			},
			"cd_path": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Path to CDROM device ISO image",
			},
			"memory": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Memory allocated to this VM in megabytes",
			},
			"os": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Type of OS used for this VM. Only used to set an icon for now",
				Validators: []validator.String{
					stringvalidator.OneOf([]string{
						freeboxTypes.CentosOS,
						freeboxTypes.DebianOS,
						freeboxTypes.FedoraOS,
						freeboxTypes.FreebsdOS,
						freeboxTypes.HomebridgeOS,
						freeboxTypes.JeedomOS,
						freeboxTypes.OpensuseOS,
						freeboxTypes.UbuntuOS,
						freeboxTypes.UnknownOS,
					}...),
				},
			},
			"vcpus": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Number of virtual CPUs to allocate to this VM",
			},
			"enable_screen": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether or not this VM should have a virtual screen, to use with the VNC websocket protocol",
			},
			"enable_cloudinit": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether or not to enable passing data through `cloudinit`. This uses the NoCloud iso image method; it will add a virtual CDROM drive (distinct from the one passed by `cd_path`) with the data in `cloudinit_userdata` and `cloudinit_hostname` when enabled",
			},
			"cloudinit_userdata": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "When cloudinit is enabled, raw YAML to be passed in the user-data file. Maximum 32767 characters",
			},
			"cloudinit_hostname": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "When cloudinit is enabled, hostname desired for this VM. Max 59 characters",
			},
			"bind_usb_ports": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of ports that should be bound to this VM. Only one VM can use USB at given time, whether is uses only one or all USB ports. The list of system USB ports is available in VmSystemInfo. For example: `usb-external-type-a`, `usb-external-type-c`",
			},
			"timeouts": schema.SingleNestedAttribute{
				MarkdownDescription: "Timeouts for various operations expressed as strings such as `30s` or `2h45m` where valid time units are `s` (seconds), `m` (minutes) and `h` (hours)",
				Computed:            true,
				Optional:            true,
				Default: objectdefault.StaticValue(
					types.ObjectValueMust(
						map[string]attr.Type{
							"create":     timetypes.GoDurationType{},
							"update":     timetypes.GoDurationType{},
							"read":       timetypes.GoDurationType{},
							"delete":     timetypes.GoDurationType{},
							"kill":       timetypes.GoDurationType{},
							"networking": timetypes.GoDurationType{},
						},
						map[string]attr.Value{
							"create":     timetypes.NewGoDurationValueFromStringMust(defaultTimeoutCreate),
							"update":     timetypes.NewGoDurationValueFromStringMust(defaultTimeoutUpdate),
							"read":       timetypes.NewGoDurationValueFromStringMust(defaultTimeoutRead),
							"delete":     timetypes.NewGoDurationValueFromStringMust(defaultTimeoutDelete),
							"kill":       timetypes.NewGoDurationValueFromStringMust(defaultTimeoutKill),
							"networking": timetypes.NewGoDurationValueFromStringMust(defaultTimeoutNetworking),
						},
					),
				),
				Attributes: map[string]schema.Attribute{
					"create": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						CustomType:          timetypes.GoDurationType{},
						Default:             stringdefault.StaticString(defaultTimeoutCreate),
						MarkdownDescription: "Timeout for resource creation (default: `\"" + defaultTimeoutCreate + "\"`)",
					},
					"update": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						CustomType:          timetypes.GoDurationType{},
						Default:             stringdefault.StaticString(defaultTimeoutUpdate),
						MarkdownDescription: "Timeout for resource updating (default: `\"" + defaultTimeoutUpdate + "\"`)",
					},
					"read": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						CustomType:          timetypes.GoDurationType{},
						Default:             stringdefault.StaticString(defaultTimeoutRead),
						MarkdownDescription: "Timeout for resource refreshing (default: `\"" + defaultTimeoutRead + "\"`)",
					},
					"delete": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						CustomType:          timetypes.GoDurationType{},
						Default:             stringdefault.StaticString(defaultTimeoutDelete),
						MarkdownDescription: "Timeout for resource deletion (default: `\"" + defaultTimeoutDelete + "\"`)",
					},
					"kill": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						CustomType:          timetypes.GoDurationType{},
						Default:             stringdefault.StaticString(defaultTimeoutKill),
						MarkdownDescription: "Duration to wait for a graceful shutdown before force killing the virtual machine (default: `\"" + defaultTimeoutKill + "\"`)",
					},
					"networking": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						CustomType:          timetypes.GoDurationType{},
						Default:             stringdefault.StaticString(defaultTimeoutNetworking),
						MarkdownDescription: "Duration to wait for the virtual machine to appear on the network (default: `\"" + defaultTimeoutNetworking + "\"`)",
					},
				},
			},
			"networking": schema.SetNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Network binds of the virtual machine",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ipv6": schema.SetAttribute{
							Computed:            true,
							MarkdownDescription: "List of IPV6 addresses on the network interface",
							ElementType:         types.StringType,
						},
						"ipv4": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Unique IPV4 address on the network interface",
						},
						"interface": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Name of the network interface the virtual machine is bound to",
						},
					},
				},
			},
		},
	}
}

func (v *virtualMachineResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (v *virtualMachineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model virtualMachineModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}

	payload, diagnostics := model.toClientPayload(ctx)
	if diagnostics.HasError() {
		resp.Diagnostics.Append(diagnostics...)
		return
	}

	virtualMachine, err := v.client.CreateVirtualMachine(ctx, payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create virtual machine",
			err.Error(),
		)
		return
	}

	if d := model.fromClientType(virtualMachine); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}

	var timeouts timeoutsModel
	resp.Diagnostics.Append(model.Timeouts.As(ctx, &timeouts, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}
	createTimeout, diags := timeouts.Create.ValueGoDuration()
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	status, err := v.start(ctx, virtualMachine.ID)
	model.Status = basetypes.NewStringValue(status)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to start virtual machine",
			err.Error(),
		)
		return
	}

	networkingTimeout, diags := timeouts.Networking.ValueGoDuration()
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	binds, err := v.getNetworkBinds(ctx, virtualMachine, networkingTimeout)
	if err == errNetworkingTimeout {
		resp.Diagnostics.AddWarning("Networking Timeout", "Reached timeout of \""+networkingTimeout.String()+"\" while waiting for networking to be available")
	} else if err != nil {
		resp.Diagnostics.AddError(
			"Failed to determine networking information",
			err.Error(),
		)
		return
	}
	if diags := model.setNetworking(binds); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *virtualMachineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model virtualMachineModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}
	var timeouts timeoutsModel
	resp.Diagnostics.Append(model.Timeouts.As(ctx, &timeouts, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}
	readTimeout, diag := timeouts.Read.ValueGoDuration()
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	resultChannel := make(chan freeboxTypes.VirtualMachine)
	errChannel := make(chan error)
	go func() {
		virtualMachine, err := v.client.GetVirtualMachine(ctx, model.ID.ValueInt64())
		if err != nil {
			errChannel <- err
			return
		}
		resultChannel <- virtualMachine
	}()

	var virtualMachine freeboxTypes.VirtualMachine
	select {
	case <-ctx.Done():
		resp.Diagnostics.AddError(
			"Context cancelled",
			fmt.Sprintf("reading virtual machine `%d` was cancelled or reached timeout", model.ID.ValueInt64()),
		)
		return
	case virtualMachine = <-resultChannel:
		if d := model.fromClientType(virtualMachine); d.HasError() {
			resp.Diagnostics.Append(d...)
			return
		}
	case err := <-errChannel:
		resp.Diagnostics.AddError(
			"Failed to get virtual machine",
			err.Error(),
		)
		return
	}

	networkingTimeout, diag := timeouts.Networking.ValueGoDuration()
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	binds, err := v.getNetworkBinds(ctx, virtualMachine, networkingTimeout)
	if err == errNetworkingTimeout {
		resp.Diagnostics.AddWarning("Networking Timeout", "Reached timeout of \""+networkingTimeout.String()+"\" while waiting for networking to be available")
	} else if err != nil {
		resp.Diagnostics.AddError(
			"Failed to determine networking information",
			err.Error(),
		)
		return
	}
	if diags := model.setNetworking(binds); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *virtualMachineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model virtualMachineModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, diagnostics := model.toClientPayload(ctx)
	if diagnostics.HasError() {
		resp.Diagnostics.Append(diagnostics...)
		return
	}

	var timeouts timeoutsModel
	resp.Diagnostics.Append(model.Timeouts.As(ctx, &timeouts, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diag := timeouts.Update.ValueGoDuration()
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	var state virtualMachineModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Status.ValueString() != freeboxTypes.StoppedStatus {
		killTimeout, diag := timeouts.Kill.ValueGoDuration()
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
		status, err := v.stop(ctx, model.ID.ValueInt64(), killTimeout)
		model.Status = basetypes.NewStringValue(status)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to stop virtual machine",
				err.Error(),
			)
			return
		}
	}

	virtualMachine, err := v.client.UpdateVirtualMachine(ctx, model.ID.ValueInt64(), payload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update virtual machine",
			err.Error(),
		)
		return
	}

	if d := model.fromClientType(virtualMachine); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}

	status, err := v.start(ctx, virtualMachine.ID)
	model.Status = basetypes.NewStringValue(status)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to start virtual machine",
			err.Error(),
		)
		return
	}

	networkingTimeout, diag := timeouts.Networking.ValueGoDuration()
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	binds, err := v.getNetworkBinds(ctx, virtualMachine, networkingTimeout)
	if err == errNetworkingTimeout {
		resp.Diagnostics.AddWarning("Networking Timeout", "Reached timeout of \""+networkingTimeout.String()+"\" while waiting for networking to be available")
	} else if err != nil {
		resp.Diagnostics.AddError(
			"Failed to determine networking information",
			err.Error(),
		)
		return
	}
	if diags := model.setNetworking(binds); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *virtualMachineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model virtualMachineModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}
	var timeouts timeoutsModel
	resp.Diagnostics.Append(model.Timeouts.As(ctx, &timeouts, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteTimeout, diag := timeouts.Delete.ValueGoDuration()
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	if model.Status.ValueString() != freeboxTypes.StoppedStatus {
		killTimeout, diag := timeouts.Kill.ValueGoDuration()
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
		status, err := v.stop(ctx, model.ID.ValueInt64(), killTimeout)
		model.Status = basetypes.NewStringValue(status)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to stop virtual machine",
				err.Error(),
			)
			return
		}
	}

	if err := v.client.DeleteVirtualMachine(ctx, model.ID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete virtual machine",
			err.Error(),
		)
		return
	}
}

func (v *virtualMachineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("timeouts"), &timeoutsModel{
		Read:       timetypes.NewGoDurationValueFromStringMust(defaultTimeoutRead),
		Create:     timetypes.NewGoDurationValueFromStringMust(defaultTimeoutCreate),
		Update:     timetypes.NewGoDurationValueFromStringMust(defaultTimeoutUpdate),
		Delete:     timetypes.NewGoDurationValueFromStringMust(defaultTimeoutDelete),
		Kill:       timetypes.NewGoDurationValueFromStringMust(defaultTimeoutKill),
		Networking: timetypes.NewGoDurationValueFromStringMust(defaultTimeoutNetworking),
	})...)
}

func (v *virtualMachineResource) start(ctx context.Context, identifier int64) (status string, err error) {
	var channel chan freeboxTypes.Event
	channel, err = v.client.ListenEvents(ctx, []freeboxTypes.EventDescription{{
		Source: "vm",
		Name:   "state_changed",
	}})
	if err != nil {
		return status, fmt.Errorf("failed to subscribe to virtual machine state change events: %s", err)
	}

	if err = v.client.StartVirtualMachine(ctx, identifier); err != nil {
		return status, fmt.Errorf("failed to start virtual machine: %s", err)
	}

	for {
		select {
		case event := <-channel:
			if event.Error != nil {
				return status, fmt.Errorf("received an error while monitoring virtual machine state: %s", event.Error)
			}
			var stateChangeEvent virtualMachineStateChangeEvent
			if err = json.Unmarshal(event.Notification.Result, &stateChangeEvent); err != nil {
				return status, fmt.Errorf("failed to parse the received virtual machine state change event: %s", err)
			}
			if stateChangeEvent.ID != identifier {
				// Ignore state change event that are unrelated to the VM that was just created
				continue
			}

			status = stateChangeEvent.Status
			switch stateChangeEvent.Status {
			case freeboxTypes.RunningStatus:
				return status, err
			case freeboxTypes.StartingStatus:
				continue
			default:
				return status, fmt.Errorf("virtual machine `%d` is in state `%s` which is unexpected", identifier, stateChangeEvent.Status)
			}
		case <-ctx.Done():
			return status, fmt.Errorf("starting virtual machine `%d` was cancelled or reached timeout", identifier)
		}
	}
}

func (v *virtualMachineResource) stop(ctx context.Context, identifier int64, killTimeout time.Duration) (status string, err error) {
	var channel chan freeboxTypes.Event
	channel, err = v.client.ListenEvents(ctx, []freeboxTypes.EventDescription{{
		Source: "vm",
		Name:   "state_changed",
	}})
	if err != nil {
		return status, fmt.Errorf("failed to subscribe to virtual machine state change events: %s", err)
	}

	timeToKill := time.After(killTimeout)

	if err = v.client.StopVirtualMachine(ctx, identifier); err != nil {
		return status, fmt.Errorf("failed to stop virtual machine: %s", err)
	}

	for {
		select {
		case <-time.After(time.Second * 5):
			switch status {
			case freeboxTypes.StoppingStatus, freeboxTypes.StoppedStatus:
				continue
			default:
				if err = v.client.StopVirtualMachine(ctx, identifier); err != nil {
					return status, fmt.Errorf("failed to stop virtual machine: %s", err)
				}
			}
		case event := <-channel:
			if event.Error != nil {
				return status, fmt.Errorf("received an error while monitoring virtual machine state: %s", event.Error)
			}
			var stateChangeEvent virtualMachineStateChangeEvent
			if err = json.Unmarshal(event.Notification.Result, &stateChangeEvent); err != nil {
				return status, fmt.Errorf("failed to parse the received virtual machine state change event: %s", err)
			}
			if stateChangeEvent.ID != identifier {
				// Ignore state change event that are unrelated to the VM that was just created
				continue
			}

			status = stateChangeEvent.Status
			switch stateChangeEvent.Status {
			case freeboxTypes.StoppedStatus:
				return status, nil
			case freeboxTypes.StoppingStatus:
				continue
			case freeboxTypes.RunningStatus:
				continue
			default:
				return status, fmt.Errorf("virtual machine `%d` is in state `%s` which is unexpected", identifier, stateChangeEvent.Status)
			}
		case <-timeToKill:
			if err = v.client.KillVirtualMachine(ctx, identifier); err != nil {
				return status, fmt.Errorf("failed to kill virtual machine: %s", err)
			}
		case <-ctx.Done():
			return status, fmt.Errorf("stopping virtual machine `%d` was cancelled or reached timeout", identifier)
		}
	}
}

func (v *virtualMachineResource) getNetworkBinds(ctx context.Context, virtualMachine freeboxTypes.VirtualMachine, networkingTimeout time.Duration) ([]networkBind, error) {
	timeoutDeadline := time.After(networkingTimeout)
	for {
		var binds []networkBind
		interfaces, err := v.client.ListLanInterfaceInfo(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list lan interface info: %s", err)
		}
		for _, interfaceInfo := range interfaces {
			interfaceName := interfaceInfo.Name
			if interfaceInfo.HostCount == 0 {
				continue
			}
			hosts, err := v.client.GetLanInterface(ctx, interfaceName)
			if err != nil {
				return nil, fmt.Errorf("failed to get lan interface \"%s\": %s", interfaceName, err)
			}
			for _, host := range hosts {
				if host.L2Ident.Type == "mac_address" && strings.EqualFold(host.L2Ident.ID, virtualMachine.Mac) {
					bind := networkBind{
						Interface: interfaceName,
					}
					for _, connectivity := range host.L3Connectivities {
						if connectivity.Type == freeboxTypes.IPV4 {
							bind.IPv4 = connectivity.Address
						}
						if connectivity.Type == freeboxTypes.IPV6 {
							bind.IPv6 = append(bind.IPv6, connectivity.Address)
						}
					}
					if bind.IPv4 != "" && bind.IPv6 != nil {
						binds = append(binds, bind)
					}
				}
			}
		}
		if len(binds) > 0 {
			return binds, nil
		}
		select {
		case <-timeoutDeadline:
			return nil, errNetworkingTimeout
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(slices.Min([]time.Duration{networkingTimeout / 10, time.Second * 5})):
			// Just wait for at most 5s before retrying
		}
	}
}
