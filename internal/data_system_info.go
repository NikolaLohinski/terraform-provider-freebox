package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nikolalohinski/free-go/client"
)

var _ datasource.DataSource = &systemInfoDataSource{}

func NewSystemInfoDataSource() datasource.DataSource {
	return &systemInfoDataSource{}
}

type systemInfoDataSource struct {
	client client.Client
}

type systemInfoModel struct {
	FirmwareVersion  types.String `tfsdk:"firmware_version"`
	Mac              types.String `tfsdk:"mac"`
	Serial           types.String `tfsdk:"serial"`
	Uptime           types.String `tfsdk:"uptime"`
	UptimeVal        types.Int64  `tfsdk:"uptime_val"`
	BoardName        types.String `tfsdk:"board_name"`
	TempCPUM         types.Int64  `tfsdk:"temp_cpum"`
	TempSW           types.Int64  `tfsdk:"temp_sw"`
	TempCPUB         types.Int64  `tfsdk:"temp_cpub"`
	FanRPM           types.Int64  `tfsdk:"fan_rpm"`
	BoxAuthenticated types.Bool   `tfsdk:"box_authenticated"`
	UserMainStorage  types.String `tfsdk:"user_main_storage"`
}

func (d *systemInfoDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_info"
}

func (d *systemInfoDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get system information about the Freebox (firmware version, hardware, temperatures, uptime).",
		Attributes: map[string]schema.Attribute{
			"firmware_version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Freebox firmware version",
			},
			"mac": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Freebox MAC address",
			},
			"serial": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Freebox serial number",
			},
			"uptime": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Human-readable uptime string",
			},
			"uptime_val": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Uptime in seconds",
			},
			"board_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Hardware revision identifier",
			},
			"temp_cpum": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "CPU M temperature in °C",
			},
			"temp_sw": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Switch temperature in °C",
			},
			"temp_cpub": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "CPU B temperature in °C",
			},
			"fan_rpm": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Fan speed in RPM",
			},
			"box_authenticated": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the Freebox is currently authenticated",
			},
			"user_main_storage": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Label of the storage partition used for user data",
			},
		},
	}
}

func (d *systemInfoDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *systemInfoDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	info, err := d.client.GetSystemInfo(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get system info", fmt.Sprintf("Failed to get system info: %s", err))
		return
	}

	data := systemInfoModel{
		FirmwareVersion:  types.StringValue(info.FirmwareVersion),
		Mac:              types.StringValue(info.Mac),
		Serial:           types.StringValue(info.Serial),
		Uptime:           types.StringValue(info.Uptime),
		UptimeVal:        types.Int64Value(info.UptimeVal),
		BoardName:        types.StringValue(info.BoardName),
		TempCPUM:         types.Int64Value(int64(info.TempCPUM)),
		TempSW:           types.Int64Value(int64(info.TempSW)),
		TempCPUB:         types.Int64Value(int64(info.TempCPUB)),
		FanRPM:           types.Int64Value(int64(info.FanRPM)),
		BoxAuthenticated: types.BoolValue(info.BoxAuthenticated),
		UserMainStorage:  types.StringValue(info.UserMainStorage),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
