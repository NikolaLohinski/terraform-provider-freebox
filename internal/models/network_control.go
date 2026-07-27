package models

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

func NetworkControlValidator() validator.String {
	return stringvalidator.OneOf(
		string(freeboxTypes.RuleModeAllowed),
		string(freeboxTypes.RuleModeDenied),
		string(freeboxTypes.RuleModeWebOnly),
	)
}

func NetworkControlDayRangeValidator() validator.String {
	return stringvalidator.OneOf(
		string(freeboxTypes.DayRangeFrenchBankHolidays),
		string(freeboxTypes.DayRangeFrenchSchoolHolidaysA),
		string(freeboxTypes.DayRangeFrenchSchoolHolidaysB),
		string(freeboxTypes.DayRangeFrenchSchoolHolidaysC),
		string(freeboxTypes.DayRangeFrenchSchoolHolidaysCorse),
	)
}
