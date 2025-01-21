package models

import (
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type Polling struct {
	Interval timetypes.GoDuration `tfsdk:"interval"`
	Timeout  timetypes.GoDuration `tfsdk:"timeout"`
}

func NewPollingSpecModel(interval, timeout time.Duration) basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(Polling{}.AttrTypes(), map[string]attr.Value{
		"interval": timetypes.NewGoDurationValue(interval),
		"timeout":  timetypes.NewGoDurationValue(timeout),
	})
}

func PollingSpecModelResourceAttributes(interval, timeout time.Duration) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"interval": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			CustomType:          timetypes.GoDurationType{},
			MarkdownDescription: "The interval at which to poll.",
			Default:             stringdefault.StaticString(timetypes.NewGoDurationValue(interval).String()),
		},
		"timeout": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			CustomType:          timetypes.GoDurationType{},
			MarkdownDescription: "The timeout for the operation.",
			Default:             stringdefault.StaticString(timetypes.NewGoDurationValue(timeout).String()),
		},
	}
}

func (o Polling) ResourceAttributes() map[string]schema.Attribute {
	return PollingSpecModelResourceAttributes(5*time.Second, 5*time.Minute)
}

func (o Polling) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"interval": timetypes.GoDurationType{},
		"timeout":  timetypes.GoDurationType{},
	}
}
