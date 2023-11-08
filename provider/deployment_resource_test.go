package provider_test

import (
	"github.com/MakeNowJust/heredoc"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	. "github.com/onsi/ginkgo/v2"
	// . "github.com/onsi/gomega"
	// . "github.com/onsi/gomega/gstruct"
)

var _ = Context("resource freebox_some", func() {
	Context("...", func() {
		It("...", func() {
			resource.Test(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: heredoc.Doc(`
							// TODO
							`,
						),
						Check: resource.ComposeAggregateTestCheckFunc(
							func(s *terraform.State) error {
								// TODO
								return nil
							},
						),
					},
				},
			})
		})
	})
})
