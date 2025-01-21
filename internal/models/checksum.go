package models

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

func ChecksumValidator() validator.String {
	return &checksumValidator{
		methodValidator: stringvalidator.OneOf(
			string(freeboxTypes.HashTypeMD5),
			string(freeboxTypes.HashTypeSHA1),
			string(freeboxTypes.HashTypeSHA256),
			string(freeboxTypes.HashTypeSHA512),
		),
		valueValidator: stringvalidator.LengthAtLeast(1),
	}
}

type checksumValidator struct{
	methodValidator validator.String
	valueValidator  validator.String
}

func (s *checksumValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// The null case should be handled by required attribute or conflicts with other validators.
	if req.ConfigValue.IsNull() {
		return
	}

	parts := strings.SplitN(req.ConfigValue.ValueString(), ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid checksum", "Checksum must be in the format <method>:<value>")
		return
	}

	s.methodValidator.ValidateString(ctx, validator.StringRequest{
		Path: req.Path.AtTupleIndex(0),
		PathExpression: req.PathExpression.AtSetValue(basetypes.NewStringValue(parts[0])),
		ConfigValue: basetypes.NewStringValue(parts[0]),
		Config: req.Config,
	}, resp)
	s.valueValidator.ValidateString(ctx, validator.StringRequest{
		Path: req.Path.AtTupleIndex(1),
		PathExpression: req.PathExpression.AtSetValue(basetypes.NewStringValue(parts[1])),
		ConfigValue: basetypes.NewStringValue(parts[1]),
		Config: req.Config,
	}, resp)

	return
}

func (s *checksumValidator) Description(ctx context.Context) string {
	return "Checksum validator"
}

func (s *checksumValidator) MarkdownDescription(ctx context.Context) string {
	return "Checksum validator"
}
