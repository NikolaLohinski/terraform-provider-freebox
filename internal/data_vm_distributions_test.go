package internal_test

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe(`data "freebox_virtual_machine_distributions" { ... }`, func() {
	var (
		config        string
		resName       string
		distributions []freeboxTypes.VirtualMachineDistribution
	)

	BeforeEach(func(ctx SpecContext) {
		splitName := strings.Split(("test-" + uuid.New().String())[:30], "-")
		resName = strings.Join(splitName[:len(splitName)-1], "-")

		var err error
		distributions, err = freeboxClient.GetVirtualMachineDistributions(ctx)
		Expect(err).To(BeNil())
		Expect(distributions).ToNot(BeEmpty())
	})

	JustBeforeEach(func() {
		config = providerBlock + `
			data "freebox_virtual_machine_distributions" "` + resName + `" {
			}
		`
	})

	It("should list available VM distributions", func(ctx SpecContext) {
		resource.UnitTest(GinkgoT(), resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr(
							"data.freebox_virtual_machine_distributions."+resName,
							"distributions.#",
							fmt.Sprintf("%d", len(distributions)),
						),
						func(s *terraform.State) error {
							state := s.RootModule().Resources["data.freebox_virtual_machine_distributions."+resName].Primary.Attributes

							for i, dist := range distributions {
								Expect(state[fmt.Sprintf("distributions.%d.name", i)]).To(Equal(dist.Name))
								Expect(state[fmt.Sprintf("distributions.%d.os", i)]).To(Equal(dist.OS))
								Expect(state[fmt.Sprintf("distributions.%d.url", i)]).To(Equal(dist.URL))
								Expect(state[fmt.Sprintf("distributions.%d.hash", i)]).To(Equal(dist.Hash))
							}

							return nil
						},
					),
				},
			},
		})
	})
})
