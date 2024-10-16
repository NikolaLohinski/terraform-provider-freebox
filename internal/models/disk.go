package models

import (
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
	return int64validator.AtLeast(0)
}
