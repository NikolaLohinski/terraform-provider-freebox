package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

var (
	_ resource.Resource = &virtualMachineResource{}

	defaultCreateTimeout = time.Minute * 5
	defaultUpdateTimeout = defaultCreateTimeout
	defaultDeleteTimeout = defaultCreateTimeout
)

func NewVirtualMachineResource() resource.Resource {
	return &virtualMachineResource{}
}

// virtualMachineResource defines the resource implementation.
type virtualMachineResource struct {
	client client.Client
}

// virtualMachineModel describes the resource data model.
type virtualMachineModel struct {
	ID                types.Int64    `tfsdk:"id"`
	Mac               types.String   `tfsdk:"mac"`
	Status            types.String   `tfsdk:"status"`
	Name              types.String   `tfsdk:"name"`
	DiskPath          types.String   `tfsdk:"disk_path"`
	DiskType          types.String   `tfsdk:"disk_type"`
	CDPath            types.String   `tfsdk:"cd_path"`
	Memory            types.Int64    `tfsdk:"memory"`
	OS                types.String   `tfsdk:"os"`
	VCPUs             types.Int64    `tfsdk:"vcpus"`
	EnableScreen      types.Bool     `tfsdk:"enable_screen"`
	BindUSBPorts      types.List     `tfsdk:"bind_usb_ports"`
	EnableCloudInit   types.Bool     `tfsdk:"enable_cloudinit"`
	CloudInitUserData types.String   `tfsdk:"cloudinit_userdata"`
	CloudHostName     types.String   `tfsdk:"cloudinit_hostname"`
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
}

func (v *virtualMachineModel) fromClientType(virtualMachine freeboxTypes.VirtualMachine) (diagnostics diag.Diagnostics) {
	v.ID = basetypes.NewInt64Value(virtualMachine.ID)
	v.Mac = basetypes.NewStringValue(virtualMachine.Mac)
	v.Status = basetypes.NewStringValue(virtualMachine.Status)
	v.Name = basetypes.NewStringValue(virtualMachine.Name)
	v.DiskPath = basetypes.NewStringValue(string(virtualMachine.DiskPath))
	v.DiskType = basetypes.NewStringValue(string(virtualMachine.DiskType))
	v.CDPath = basetypes.NewStringValue(string(virtualMachine.CDPath))
	v.Memory = basetypes.NewInt64Value(virtualMachine.Memory)
	v.VCPUs = basetypes.NewInt64Value(virtualMachine.VCPUs)
	v.OS = basetypes.NewStringValue(virtualMachine.OS)
	v.EnableScreen = basetypes.NewBoolValue(virtualMachine.EnableScreen)
	v.EnableCloudInit = basetypes.NewBoolValue(virtualMachine.EnableCloudInit)
	v.CloudInitUserData = basetypes.NewStringValue(string(virtualMachine.CloudInitUserData))
	v.CloudHostName = basetypes.NewStringValue(string(virtualMachine.CloudHostName))

	usbPorts := []attr.Value{}
	for _, port := range virtualMachine.BindUSBPorts {
		usbPorts = append(usbPorts, basetypes.NewStringValue(port))
	}
	v.BindUSBPorts, diagnostics = basetypes.NewListValue(types.StringType, usbPorts)

	return diagnostics
}

func (v *virtualMachineModel) toClientPayload(ctx context.Context) (payload freeboxTypes.VirtualMachinePayload, diagnostics diag.Diagnostics) {
	payload.Name = v.Name.ValueString()
	payload.DiskPath = freeboxTypes.Base64Path(v.DiskPath.ValueString())
	payload.DiskType = v.DiskType.ValueString()
	payload.CDPath = freeboxTypes.Base64Path(v.CDPath.ValueString())
	payload.Memory = v.Memory.ValueInt64()
	payload.VCPUs = v.VCPUs.ValueInt64()
	payload.OS = v.OS.ValueString()
	payload.EnableScreen = v.EnableScreen.ValueBool()
	payload.EnableCloudInit = v.EnableCloudInit.ValueBool()
	payload.CloudInitUserData = v.CloudInitUserData.ValueString()
	payload.CloudHostName = v.CloudHostName.ValueString()

	return payload, v.BindUSBPorts.ElementsAs(ctx, &payload.BindUSBPorts, false)
}

func (v *virtualMachineResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine"
}

func (v *virtualMachineResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a virtual machine instance within a Freebox box. See the [Freebox blog](https://dev.freebox.fr/blog/?p=5450) for additional details",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
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
			"name": schema.BoolAttribute{
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
				MarkdownDescription: "Path to CDROM device ISO image",
			},
			"memory": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Memory allocated to this VM in megabytes",
			},
			"os": schema.StringAttribute{
				Optional:            true,
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
				MarkdownDescription: "Whether or not this VM should have a virtual screen, to use with the VNC websocket protocol",
			},
			"enable_cloudinit": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether or not to enable passing data through `cloudinit`. This uses the NoCloud iso image method; it will add a virtual CDROM drive (distinct from the one passed by `cd_path`) with the data in `cloudinit_userdata` and `cloudinit_hostname` when enabled",
			},
			"cloudinit_userdata": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "When cloudinit is enabled, raw YAML to be passed in the user-data file. Maximum 32767 characters",
			},
			"cloudinit_hostname": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "When cloudinit is enabled, hostname desired for this VM. Max 59 characters",
			},
			"bind_usb_ports": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of ports that should be bound to this VM. Only one VM can use USB at given time, whether is uses only one or all USB ports. The list of system USB ports is available in VmSystemInfo. For example: `usb-external-type-a`, `usb-external-type-c`",
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create:            true,
				CreateDescription: "A duration string such as `30s` or `2h45m` where valid time units are `s` (seconds), `m` (minutes) and `h` (hours) [default: `" + defaultCreateTimeout.String() + "`]",
				Delete:            true,
				DeleteDescription: "A duration string such as `30s` or `2h45m` where valid time units are `s` (seconds), `m` (minutes) and `h` (hours) [default: `" + defaultDeleteTimeout.String() + "`]",
				Update:            true,
				UpdateDescription: "A duration string such as `30s` or `2h45m` where valid time units are `s` (seconds), `m` (minutes) and `h` (hours) [default: `" + defaultUpdateTimeout.String() + "`]",
			}),
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

	createTimeout, diag := model.Timeouts.Create(ctx, defaultCreateTimeout)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	channel, err := v.client.ListenEvents(ctx, []freeboxTypes.EventDescription{{
		Source: "vm",
		Name:   "state_changed",
	}})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to subscribe to virtual machine state change events",
			err.Error(),
		)
		return
	}

	if err := v.client.StartVirtualMachine(ctx, virtualMachine.ID); err != nil {
		resp.Diagnostics.AddError(
			"Failed to start virtual machine",
			err.Error(),
		)
		return
	}

	for {
		select {
		case event := <-channel:
			if event.Error != nil {
				resp.Diagnostics.AddError(
					"Received an error while monitoring virtual machine state",
					event.Error.Error(),
				)
				return
			}
			var stateChangeEvent struct {
				ID     int64  `json:"id"`
				Status string `json:"status"`
			}
			if err := json.Unmarshal(event.Notification.Result, &stateChangeEvent); err != nil {
				resp.Diagnostics.AddError(
					"Failed to parse the received virtual machine state change event",
					err.Error(),
				)
				return
			}
			if stateChangeEvent.ID != virtualMachine.ID {
				// Ignore state change event that are unrelated to the VM that was just created
				continue
			}

			model.Status = basetypes.NewStringValue(stateChangeEvent.Status)
			switch stateChangeEvent.Status {
			case freeboxTypes.RunningStatus:
				return
			case freeboxTypes.StartingStatus:
				continue
			default:
				resp.Diagnostics.AddError(
					"Virtual machine is in a unexpected state",
					fmt.Sprintf("virtual machine `%s` (id: `%d`) is in state `%s` which is unexpected", virtualMachine.Name, virtualMachine.ID, stateChangeEvent.Status),
				)
				return
			}
		case <-ctx.Done():
			resp.Diagnostics.AddError(
				"Virtual machine state monitoring was stopped unexpectedly",
				fmt.Sprintf("execution context was cancelled or reached the defined timeout (%s)", createTimeout.String()),
			)
			return
		}
	}
}

func (v *virtualMachineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model virtualMachineModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}

	virtualMachine, err := v.client.GetVirtualMachine(ctx, model.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get virtual machine",
			err.Error(),
		)
		return
	}

	if d := model.fromClientType(virtualMachine); d.HasError() {
		resp.Diagnostics.Append(d...)
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

	// TODO: stop the VM and monitor its status or just restart it if only the status is different

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

	// TODO: start the VM and monitor its status

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (v *virtualMachineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model virtualMachineModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: stop the VM and monitor its status

	if err := v.client.DeleteVirtualMachine(ctx, model.ID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete virtual machine",
			err.Error(),
		)
		return
	}
}
