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

var _ = Describe(`data "freebox_lan_interfaces" { ... }`, func() {
	var (
		config     string
		resName    string
		lanInfos   []freeboxTypes.LanInfo
	)

	BeforeEach(func(ctx SpecContext) {
		splitName := strings.Split(("test-" + uuid.New().String())[:30], "-")
		resName = strings.Join(splitName[:len(splitName)-1], "-")

		var err error
		lanInfos, err = freeboxClient.ListLanInterfaceInfo(ctx)
		Expect(err).To(BeNil())
		Expect(lanInfos).ToNot(BeEmpty())
	})

	JustBeforeEach(func() {
		config = providerBlock + `
			data "freebox_lan_interfaces" "` + resName + `" {
			}
		`
	})

	It("should list LAN interfaces", func(ctx SpecContext) {
		resource.UnitTest(GinkgoT(), resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr(
							"data.freebox_lan_interfaces."+resName,
							"interfaces.#",
							fmt.Sprintf("%d", len(lanInfos)),
						),
						func(s *terraform.State) error {
							state := s.RootModule().Resources["data.freebox_lan_interfaces."+resName].Primary.Attributes

							for i, info := range lanInfos {
								Expect(state[fmt.Sprintf("interfaces.%d.name", i)]).To(Equal(info.Name))
								Expect(state[fmt.Sprintf("interfaces.%d.host_count", i)]).To(Equal(fmt.Sprintf("%d", info.HostCount)))
							}

							return nil
						},
					),
				},
			},
		})
	})
})
