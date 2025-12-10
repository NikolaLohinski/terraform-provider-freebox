package internal_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
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

func terraformConfigWithAttribute(attribute string, value interface{}) func(config string) string {
	r, err := regexp.Compile(`(?m)^\s*` + regexp.QuoteMeta(attribute) + `\s*=\s*.*$`)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	var encodedValue []byte

	switch v := value.(type) {
	case []byte:
		encodedValue = v
	default:
		encodedValue, err = json.Marshal(value)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}

	encodedValueString := string(encodedValue)

	return func(config string) string {
		gomega.Expect(config).To(gomega.ContainSubstring(attribute))

		count := strings.Count(config, encodedValueString)
		config = r.ReplaceAllString(config, attribute+" = "+encodedValueString)
		gomega.Expect(strings.Count(config, encodedValueString)).To(gomega.Equal(count+1), "expected %d occurrences of %s, got %d", count+1, encodedValueString, strings.Count(config, encodedValueString))

		return config
	}
}

func terraformConfigWithoutAttribute(attribute string) func(config string) string {
	ginkgo.GinkgoHelper()

	r, err := regexp.Compile(`(?m)^\s*` + regexp.QuoteMeta(attribute) + `\s*=\s*.*$`)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	return func(config string) string {
		ginkgo.GinkgoHelper()

		gomega.Expect(config).To(gomega.ContainSubstring(attribute))

		count := strings.Count(config, attribute)
		config = r.ReplaceAllString(config, "")
		gomega.Expect(strings.Count(config, attribute)).To(gomega.Equal(count-1), "expected %d occurrences of %s, got %d", count-1, attribute, strings.Count(config, attribute))

		return config
	}
}
