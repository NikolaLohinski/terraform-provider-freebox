package internal_test

import (
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DataNetshareSambaConfiguration", func() {
	var (
		config             string
		resourceName       string
		sambaConfiguration freeboxTypes.SambaConfiguration
	)

	BeforeEach(func(ctx SpecContext) {
		splitName := strings.Split(("test-" + uuid.New().String())[:30], "-")
		resourceName = strings.Join(splitName[:len(splitName)-1], "-")

		var err error
		sambaConfiguration, err = freeboxClient.GetSambaConfiguration(ctx)
		Expect(err).To(BeNil())
		Expect(sambaConfiguration).ToNot(BeNil())
	})

	JustBeforeEach(func(ctx SpecContext) {
		config = providerBlock + `
			data "freebox_netshare_samba_configuration" "` + resourceName + `" {
			}
		`
	})

	It("should fetch samba configuration information", func(ctx SpecContext) {
		resource.UnitTest(GinkgoT(), resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						func(s *terraform.State) error {
							state := s.RootModule().Resources["data.freebox_netshare_samba_configuration."+resourceName].Primary.Attributes

							Expect(state["file_share_enabled"]).To(Equal(strconv.FormatBool(sambaConfiguration.FileShareEnabled)))
							Expect(state["print_share_enabled"]).To(Equal(strconv.FormatBool(sambaConfiguration.PrintShareEnabled)))
							Expect(state["logon_enabled"]).To(Equal(strconv.FormatBool(sambaConfiguration.LogonEnabled)))
							Expect(state["logon_user"]).To(Equal(sambaConfiguration.LogonUser))
							Expect(state["workgroup"]).To(Equal(sambaConfiguration.Workgroup))
							Expect(state["v2_enabled"]).To(Equal(strconv.FormatBool(sambaConfiguration.V2Enabled)))

							return nil
						},
					),
				},
			},
		})
	})
})
