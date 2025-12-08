package models

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

func DiskTypeValidator() validator.String {
	return stringvalidator.OneOf(
		string(freeboxTypes.QCow2Disk),
		string(freeboxTypes.RawDisk),
	)
}

func DiskSizeValidator() validator.Int64 {
	return int64validator.AtLeast(0)
}

func VirtualDiskSizeValidator() validator.Int64 {
	return &virtualDiskSizeValidator{}
}

type virtualDiskSizeValidator struct {}

func (v virtualDiskSizeValidator) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	int64validator.AtLeast(1).ValidateInt64(ctx, req, resp)

	value := req.ConfigValue.ValueInt64()
	if value % 8_192 != 0 {
		resp.Diagnostics.AddError("Invalid virtual disk size", "Virtual disk size must be a multiple of 8_192")
	}
}

func (v virtualDiskSizeValidator) Description(ctx context.Context) string {
	return "Virtual disk size must be greater than 0 and a multiple of 1024"
}

func (v virtualDiskSizeValidator) MarkdownDescription(ctx context.Context) string {
	return "Virtual disk size must be greater than 0 and a multiple of 1024"
}

var _ validator.Int64 = (*virtualDiskSizeValidator)(nil)
