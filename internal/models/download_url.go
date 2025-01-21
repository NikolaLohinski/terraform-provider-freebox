package models

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func DownloadURLValidator() validator.String {
	return &downloadURLValidator{
		schemeValidator: stringvalidator.OneOf("http", "https", "ftp", "magnet"),
	}
}

type downloadURLValidator struct {
	schemeValidator validator.String
}

func (s *downloadURLValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	u, err := url.Parse(req.ConfigValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid URL", err.Error())
		return
	}

	s.schemeValidator.ValidateString(ctx, validator.StringRequest{
		Path:           req.Path.AtName("scheme"),
		PathExpression: req.PathExpression.AtName("scheme"),
		ConfigValue:    basetypes.NewStringValue(u.Scheme),
		Config:         req.Config,
	}, resp)
}

func (s *downloadURLValidator) Description(ctx context.Context) string {
	return "Download URL validator"
}

func (s *downloadURLValidator) MarkdownDescription(ctx context.Context) string {
	return "Download URL validator"
}
