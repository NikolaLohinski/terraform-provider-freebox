package models

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func FilePathValidator() validator.String {
	return &filePathValidator{}
}

type filePathValidator struct{}

func (s *filePathValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	stringvalidator.LengthAtLeast(1).ValidateString(ctx, req, resp)

	return
}

func (s *filePathValidator) Description(ctx context.Context) string {
	return "File path validator"
}

func (s *filePathValidator) MarkdownDescription(ctx context.Context) string {
	return "File path validator"
}
