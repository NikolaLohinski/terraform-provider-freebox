package internal_test

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe(`data "freebox_dhcp_leases" { ... }`, func() {
	var (
		resourceName string
		mac          string
		ip           string
		hostname     string
		config       string
	)

	BeforeEach(func(ctx SpecContext) {
		splitName := strings.Split(("test-" + uuid.New().String())[:30], "-")
		resourceName = strings.Join(splitName[:len(splitName)-1], "-")

		mac = fmt.Sprintf("02:00:%02X:%02X:%02X:%02X",
			randGenerator.Intn(256), randGenerator.Intn(256),
			randGenerator.Intn(256), randGenerator.Intn(256),
		)
		ip = fmt.Sprintf("192.168.1.%d", randGenerator.Intn(54)+200)
		hostname = resourceName

		lanHost, err := freeboxClient.CreateDHCPStaticLease(ctx, freeboxTypes.DHCPStaticLeasePayload{
			Mac:      mac,
			IP:       ip,
			Hostname: hostname,
		})
		Expect(err).To(BeNil())

		DeferCleanup(func(ctx SpecContext) {
			err := freeboxClient.DeleteDHCPStaticLease(ctx, lanHost.ID)
			if err != nil {
				var apiErr *client.APIError
				if !errors.As(err, &apiErr) || apiErr.Code != "noent" {
					Expect(err).To(BeNil(), "failed to clean up DHCP lease %s", lanHost.ID)
				}
			}
		})
	})

	JustBeforeEach(func(ctx SpecContext) {
		config = providerBlock + `
			data "freebox_dhcp_leases" "` + resourceName + `" {}
		`
	})

	It("should list all DHCP leases including the created one", func(ctx SpecContext) {
		resource.UnitTest(GinkgoT(), resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						func(s *terraform.State) error {
							attrs := s.RootModule().Resources["data.freebox_dhcp_leases."+resourceName].Primary.Attributes

							count, err := strconv.Atoi(attrs["leases.#"])
							Expect(err).To(BeNil())
							Expect(count).To(BeNumerically(">", 0), "expected at least one DHCP lease in the list")

							upperMac := strings.ToUpper(mac)
							found := false
							for i := 0; i < count; i++ {
								if attrs[fmt.Sprintf("leases.%d.mac", i)] == upperMac {
									found = true
									Expect(attrs[fmt.Sprintf("leases.%d.ip", i)]).To(Equal(ip))
									Expect(attrs[fmt.Sprintf("leases.%d.hostname", i)]).To(Equal(hostname))
									break
								}
							}
							Expect(found).To(BeTrue(), "lease with MAC %s not found in list", upperMac)

							return nil
						},
					),
				},
			},
		})
	})
})
