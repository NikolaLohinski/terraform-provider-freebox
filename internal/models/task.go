package models

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Task struct {
	ID   types.Int64  `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

func (o Task) ResourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "The task ID.",
			Validators: []validator.Int64{
				int64validator.AtLeast(0),
			},
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"type": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "The task type.",
			Validators: []validator.String{
				TaskTypeValidator(),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func (o Task) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   types.Int64Type,
		"type": types.StringType,
	}
}

func TaskTypeValidator() validator.String {
	return stringvalidator.OneOf(
		string(TaskTypeDownload),
		string(TaskTypeFileSystem),
		string(TaskTypeVirtualDisk),
	)
}

type TaskType string

const (
	TaskTypeDownload    TaskType = "download"
	TaskTypeFileSystem  TaskType = "file_system"
	TaskTypeVirtualDisk TaskType = "virtual_disk"
)
