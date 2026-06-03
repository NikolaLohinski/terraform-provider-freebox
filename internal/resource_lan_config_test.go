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

var _ = Describe(`resource "freebox_lan_config" { ... }`, func() {
	var (
		resName       string
		config        string
		originalConfig freeboxTypes.LanConfig
	)

	BeforeEach(func(ctx SpecContext) {
		splitName := strings.Split(("test-" + uuid.New().String())[:30], "-")
		resName = strings.Join(splitName[:len(splitName)-1], "-")

		var err error
		originalConfig, err = freeboxClient.GetLanConfig(ctx)
		Expect(err).To(BeNil())

		DeferCleanup(func(ctx SpecContext) {
			_, err := freeboxClient.UpdateLanConfig(ctx, originalConfig)
			Expect(err).To(BeNil(), "failed to restore original LAN config")
		})
	})

	Context("when managing names only", func() {
		JustBeforeEach(func() {
			config = providerBlock + `
				resource "freebox_lan_config" "` + resName + `" {
					name         = "` + originalConfig.Name + `"
					name_dns     = "` + originalConfig.NameDNS + `"
					name_mdns    = "` + originalConfig.NameMDNS + `"
					name_netbios = "` + originalConfig.NameNetBIOS + `"
				}
			`
		})

		It("should manage the LAN configuration", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_lan_config."+resName, "id", "lan_config"),
							resource.TestCheckResourceAttr("freebox_lan_config."+resName, "name", originalConfig.Name),
							resource.TestCheckResourceAttr("freebox_lan_config."+resName, "name_dns", originalConfig.NameDNS),
							resource.TestCheckResourceAttr("freebox_lan_config."+resName, "name_mdns", originalConfig.NameMDNS),
							resource.TestCheckResourceAttr("freebox_lan_config."+resName, "name_netbios", originalConfig.NameNetBIOS),
							resource.TestCheckResourceAttrSet("freebox_lan_config."+resName, "ip"),
							resource.TestCheckResourceAttrSet("freebox_lan_config."+resName, "mode"),
						),
					},
				},
			})
		})
	})

	Context("when importing", func() {
		JustBeforeEach(func() {
			config = providerBlock + `
				resource "freebox_lan_config" "` + resName + `" {}
			`
		})

		It("should import the existing LAN configuration", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:        config,
						ResourceName:  "freebox_lan_config." + resName,
						ImportState:   true,
						ImportStateId: "lan_config",
						ImportStateCheck: func(states []*terraform.InstanceState) error {
							Expect(states).To(HaveLen(1))
							Expect(states[0].ID).To(Equal("lan_config"))
							Expect(states[0].Attributes["ip"]).To(Equal(originalConfig.IP))
							Expect(states[0].Attributes["name"]).To(Equal(originalConfig.Name))
							return nil
						},
					},
				},
			})
		})
	})
})
