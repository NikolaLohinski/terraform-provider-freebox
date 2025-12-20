package internal_test

import (
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DataVirtualDisk", func() {
	var (
		config       string
		resourceName string
		lanConfig    freeboxTypes.LanConfig
	)

	BeforeEach(func(ctx SpecContext) {
		splitName := strings.Split(("test-" + uuid.New().String())[:30], "-")
		resourceName = strings.Join(splitName[:len(splitName)-1], "-")

		var err error
		lanConfig, err = freeboxClient.GetLanConfig(ctx)
		Expect(err).To(BeNil())
		Expect(lanConfig).ToNot(BeNil())
	})

	JustBeforeEach(func(ctx SpecContext) {
		config = providerBlock + `
			data "freebox_lan_config" "` + resourceName + `" {
			}
		`
	})

	It("should fetch lan config information", func(ctx SpecContext) {
		resource.UnitTest(GinkgoT(), resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						func(s *terraform.State) error {
							state := s.RootModule().Resources["data.freebox_lan_config."+resourceName].Primary.Attributes

							Expect(state["ip"]).To(Equal(lanConfig.IP))
							Expect(state["name"]).To(Equal(lanConfig.Name))
							Expect(state["name_dns"]).To(Equal(lanConfig.NameDNS))
							Expect(state["name_mdns"]).To(Equal(lanConfig.NameMDNS))
							Expect(state["name_netbios"]).To(Equal(lanConfig.NameNetBIOS))
							Expect(state["mode"]).To(Equal(string(lanConfig.Mode)))

							return nil
						},
					),
				},
			},
		})
	})
})
