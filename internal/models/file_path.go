package models

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func FilePathValidator(pathAttribute path.Path) validator.String {
	return &filePathValidator{
		pathAttribute: pathAttribute,
	}
}

type filePathValidator struct {
	pathAttribute path.Path
}

func (s *filePathValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	var filePath basetypes.StringValue
	if diags := req.Config.GetAttribute(ctx, s.pathAttribute, &filePath); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	// The null case should be handled by required attribute or conflicts with other validators.
	if filePath.IsNull() || filePath.IsUnknown() {
		return
	}

	stringvalidator.LengthAtLeast(1).ValidateString(ctx, validator.StringRequest{
		Path:           s.pathAttribute,
		PathExpression: req.PathExpression,
		ConfigValue:    basetypes.NewStringValue(filePath.ValueString()),
		Config:         req.Config,
	}, resp)
	stringvalidator.RegexMatches(regexp.MustCompile("^(/?[^/]+)+$"), "File path must be a valid file path").ValidateString(ctx, validator.StringRequest{
		Path:           s.pathAttribute,
		PathExpression: req.PathExpression,
		ConfigValue:    basetypes.NewStringValue(filePath.ValueString()),
		Config:         req.Config,
	}, resp)
}

func (s *filePathValidator) Description(ctx context.Context) string {
	return "File path validator"
}

func (s *filePathValidator) MarkdownDescription(ctx context.Context) string {
	return "File path validator"
}
