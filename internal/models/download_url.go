package models

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func DownloadURLValidator(pathAttribute path.Path) validator.String {
	return &downloadURLValidator{
		schemeValidator: stringvalidator.OneOf("http", "https", "ftp", "magnet"),
		pathAttribute:   pathAttribute,
	}
}

type downloadURLValidator struct {
	schemeValidator validator.String
	pathAttribute   path.Path
}

func (s *downloadURLValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	var sourceURL basetypes.StringValue
	if diags := req.Config.GetAttribute(ctx, s.pathAttribute, &sourceURL); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueString() == "" {
		return
	}

	// The null case should be handled by required attribute or conflicts with other validators.
	if sourceURL.IsNull() || sourceURL.IsUnknown() {
		return
	}

	u, err := url.Parse(sourceURL.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid URL", err.Error())
		return
	}

	s.schemeValidator.ValidateString(ctx, validator.StringRequest{
		Path:           s.pathAttribute.AtName("scheme"),
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
