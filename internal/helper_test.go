package internal_test

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func durationEqualFunc(expected time.Duration) func(value string) error {
	return func(strValue string) error {
		value, err := time.ParseDuration(strValue)
		if err != nil {
			return fmt.Errorf("failed to parse duration: %w", err)
		}
		if value != expected {
			return fmt.Errorf("expected %s, got %s", expected, value)
		}

		return nil
	}
}

var _ resource.CheckResourceAttrWithFunc = durationEqualFunc(0)
