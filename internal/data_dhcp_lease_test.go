package internal_test

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe(`data "freebox_dhcp_lease" { ... }`, func() {
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
			data "freebox_dhcp_lease" "` + resourceName + `" {
				mac = "` + mac + `"
			}
		`
	})

	It("should read the DHCP lease by MAC address", func(ctx SpecContext) {
		resource.UnitTest(GinkgoT(), resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("data.freebox_dhcp_lease."+resourceName, "id"),
						resource.TestCheckResourceAttr("data.freebox_dhcp_lease."+resourceName, "mac", strings.ToUpper(mac)),
						resource.TestCheckResourceAttr("data.freebox_dhcp_lease."+resourceName, "ip", ip),
						resource.TestCheckResourceAttr("data.freebox_dhcp_lease."+resourceName, "hostname", hostname),
						resource.TestCheckResourceAttr("data.freebox_dhcp_lease."+resourceName, "comment", ""),
					),
				},
			},
		})
	})

	Context("when the MAC address is not found", func() {
		BeforeEach(func(ctx SpecContext) {
			mac = fmt.Sprintf("02:FF:%02X:%02X:%02X:%02X",
				randGenerator.Intn(256), randGenerator.Intn(256),
				randGenerator.Intn(256), randGenerator.Intn(256),
			)
		})

		It("should return an error", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile(`DHCP lease not found`),
					},
				},
			})
		})
	})
})
