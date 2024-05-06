package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &virtualMachineResource{}

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
			"bind_usbports": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of ports that should be bound to this VM. Only one VM can use USB at given time, whether is uses only one or all USB ports. The list of system USB ports is available in VmSystemInfo. For example: `usb-external-type-a`, `usb-external-type-c`",
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
	var data virtualMachineModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (v *virtualMachineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data virtualMachineModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	virtualMachine, err := v.client.GetVirtualMachine(ctx, data.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get virtual machine",
			err.Error(),
		)
		return
	}
	data.ID = basetypes.NewInt64Value(virtualMachine.ID)
	data.Mac = basetypes.NewStringValue(virtualMachine.Mac)
	data.Status = basetypes.NewStringValue(virtualMachine.Status)
	data.Name = basetypes.NewStringValue(virtualMachine.Name)
	data.DiskPath = basetypes.NewStringValue(string(virtualMachine.DiskPath))
	data.DiskType = basetypes.NewStringValue(string(virtualMachine.DiskType))
	data.CDPath = basetypes.NewStringValue(string(virtualMachine.CDPath))
	data.Memory = basetypes.NewInt64Value(virtualMachine.Memory)
	data.VCPUs = basetypes.NewInt64Value(virtualMachine.VCPUs)
	data.OS = basetypes.NewStringValue(virtualMachine.OS)
	data.EnableScreen = basetypes.NewBoolValue(virtualMachine.EnableScreen)
	data.EnableCloudInit = basetypes.NewBoolValue(virtualMachine.EnableCloudInit)
	data.CloudInitUserData = basetypes.NewStringValue(string(virtualMachine.CloudInitUserData))
	data.CloudHostName = basetypes.NewStringValue(string(virtualMachine.CloudHostName))

	usbPorts := []attr.Value{}
	for _, port := range virtualMachine.BindUSBPorts {
		usbPorts = append(usbPorts, basetypes.NewStringValue(port))
	}
	data.BindUSBPorts = basetypes.NewListValueMust(types.StringType, usbPorts)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (v *virtualMachineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data virtualMachineModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (v *virtualMachineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data virtualMachineModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO
}
