package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

var (
	_ resource.Resource = &virtualMachineResource{}
)

type virtualMachineStateChangeEvent struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
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
}

type timeoutsModel struct {
	Create timetypes.Duration `tfsdk:"create"`
	Update timetypes.Duration `tfsdk:"update"`
	Read   timetypes.Duration `tfsdk:"read"`
	Delete timetypes.Duration `tfsdk:"delete"`
	Kill   timetypes.Duration `tfsdk:"kill"`
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
	if !v.BindUSBPorts.IsNull() && !v.BindUSBPorts.IsUnknown() {
		return payload, v.BindUSBPorts.ElementsAs(ctx, &payload.BindUSBPorts, false)
	}
	return payload, nil
}

func (v *virtualMachineResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine"
}

func (v *virtualMachineResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a virtual machine instance within a Freebox box. See the [Freebox blog](https://dev.freebox.fr/blog/?p=5450) for additional details",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the VM",
			},
			"mac": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "VM ethernet interface MAC address",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "VM status",
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
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Timeouts for various operations expressed as strings such as `30s` or `2h45m` where valid time units are `s` (seconds), `m` (minutes) and `h` (hours)",
				Default: objectdefault.StaticValue(basetypes.NewObjectValueMust(map[string]attr.Type{
					"create": timetypes.DurationType{},
					"delete": timetypes.DurationType{},
					"update": timetypes.DurationType{},
					"read":   timetypes.DurationType{},
					"kill":   timetypes.DurationType{},
				},
					map[string]attr.Value{
						"read":   timetypes.NewDurationValueFromStringMust("5m"),
						"create": timetypes.NewDurationValueFromStringMust("5m"),
						"update": timetypes.NewDurationValueFromStringMust("5m"),
						"delete": timetypes.NewDurationValueFromStringMust("5m"),
						"kill":   timetypes.NewDurationValueFromStringMust("30s"),
					},
				)),
				Attributes: map[string]schema.Attribute{
					"create": schema.StringAttribute{
						Optional:   true,
						Computed:   true,
						CustomType: timetypes.DurationType{},
						Default:    stringdefault.StaticString("5m"),
						Validators: []validator.String{
							stringvalidator.Duration(),
						},
						MarkdownDescription: "Timeout for resource creation (default: `\"5m\"`)",
					},
					"update": schema.StringAttribute{
						Optional:   true,
						Computed:   true,
						CustomType: timetypes.DurationType{},
						Default:    stringdefault.StaticString("5m"),
						Validators: []validator.String{
							stringvalidator.Duration(),
						},
						MarkdownDescription: "Timeout for resource updating (default: `\"5m\"`)",
					},
					"read": schema.StringAttribute{
						Optional:   true,
						Computed:   true,
						CustomType: timetypes.DurationType{},
						Default:    stringdefault.StaticString("5m"),
						Validators: []validator.String{
							stringvalidator.Duration(),
						},
						MarkdownDescription: "Timeout for resource refreshing (default: `\"5m\"`)",
					},
					"delete": schema.StringAttribute{
						Optional:   true,
						Computed:   true,
						CustomType: timetypes.DurationType{},
						Default:    stringdefault.StaticString("5m"),
						Validators: []validator.String{
							stringvalidator.Duration(),
						},
						MarkdownDescription: "Timeout for resource deletion (default: `\"5m\"`)",
					},
					"kill": schema.StringAttribute{
						Optional:   true,
						Computed:   true,
						CustomType: timetypes.DurationType{},
						Default:    stringdefault.StaticString("30s"),
						Validators: []validator.String{
							stringvalidator.Duration(),
						},
						MarkdownDescription: "Duration to wait for a graceful shutdown before force killing the virtual machine (default: `\"30s\"`)",
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
	defer func() {
		if d := resp.State.Set(ctx, &model); d.HasError() {
			resp.Diagnostics.Append(d...)
		}
	}()
	var timeouts timeoutsModel
	resp.Diagnostics.Append(model.Timeouts.As(ctx, &timeouts, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}
	createTimeout, diag := timeouts.Create.ValueDuration()
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
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
	readTimeout, diag := timeouts.Read.ValueDuration()
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

	select {
	case <-ctx.Done():
		resp.Diagnostics.AddError(
			"Context cancelled",
			fmt.Sprintf("reading virtual machine `%d` was cancelled or reached timeout", model.ID.ValueInt64()),
		)
		return
	case vm := <-resultChannel:
		if d := model.fromClientType(vm); d.HasError() {
			resp.Diagnostics.Append(d...)
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	case err := <-errChannel:
		resp.Diagnostics.AddError(
			"Failed to get virtual machine",
			err.Error(),
		)
		return
	}
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
	defer func() {
		if d := resp.State.Set(ctx, &model); d.HasError() {
			resp.Diagnostics.Append(d...)
		}
	}()
	var timeouts timeoutsModel
	resp.Diagnostics.Append(model.Timeouts.As(ctx, &timeouts, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateTimeout, diag := timeouts.Update.ValueDuration()
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	if model.Status.ValueString() != freeboxTypes.StoppedStatus {
		killTimeout, diag := timeouts.Kill.ValueDuration()
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

	if d := resp.State.Set(ctx, &model); d.HasError() {
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
	deleteTimeout, diag := timeouts.Delete.ValueDuration()
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	if model.Status.ValueString() != freeboxTypes.StoppedStatus {
		killTimeout, diag := timeouts.Kill.ValueDuration()
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
