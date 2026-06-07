package internal_test

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/nikolalohinski/free-go/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe(`resource "freebox_dhcp_lease" { ... }`, func() {
	var (
		resourceName  string
		mac           string
		ip            string
		hostname      string
		initialConfig string
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
	})

	JustBeforeEach(func(ctx SpecContext) {
		initialConfig = providerBlock + `
			resource "freebox_dhcp_lease" "` + resourceName + `" {
				mac      = "` + mac + `"
				ip       = "` + ip + `"
				hostname = "` + hostname + `"
			}
		`
	})

	Context("create and delete", func() {
		It("should create and delete the lease", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: initialConfig,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_dhcp_lease."+resourceName, "mac", strings.ToUpper(mac)),
							resource.TestCheckResourceAttr("freebox_dhcp_lease."+resourceName, "ip", ip),
							resource.TestCheckResourceAttr("freebox_dhcp_lease."+resourceName, "hostname", hostname),
							resource.TestCheckResourceAttr("freebox_dhcp_lease."+resourceName, "comment", ""),
							resource.TestCheckResourceAttrSet("freebox_dhcp_lease."+resourceName, "id"),
							func(s *terraform.State) error {
								id := s.RootModule().Resources["freebox_dhcp_lease."+resourceName].Primary.Attributes["id"]
								lease, err := freeboxClient.GetDHCPStaticLease(ctx, id)
								Expect(err).To(BeNil())
								Expect(lease.Mac).To(Equal(strings.ToUpper(mac)))
								Expect(lease.IP).To(Equal(ip))
								Expect(lease.Hostname).To(Equal(hostname))
								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					id := s.RootModule().Resources["freebox_dhcp_lease."+resourceName].Primary.Attributes["id"]
					_, err := freeboxClient.GetDHCPStaticLease(ctx, id)
					var apiErr *client.APIError
					Expect(errors.As(err, &apiErr)).To(BeTrue(), "expected noent error after destroy, got: %v", err)
					Expect(apiErr.Code).To(Equal("noent"))
					return nil
				},
			})
		})
	})

	Context("create, update and delete", func() {
		Context("updating comment", func() {
			var newComment string

			BeforeEach(func(ctx SpecContext) {
				newComment = resourceName + "-updated"
			})

			It("should update without replacing the resource", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_dhcp_lease."+resourceName, "comment", ""),
							),
						},
						{
							Config: terraformConfigWithAttribute("comment", newComment)(initialConfig),
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply: []plancheck.PlanCheck{
									plancheck.ExpectResourceAction("freebox_dhcp_lease."+resourceName, plancheck.ResourceActionUpdate),
								},
							},
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_dhcp_lease."+resourceName, "comment", newComment),
								resource.TestCheckResourceAttr("freebox_dhcp_lease."+resourceName, "mac", strings.ToUpper(mac)),
								resource.TestCheckResourceAttr("freebox_dhcp_lease."+resourceName, "hostname", hostname),
							),
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						id := s.RootModule().Resources["freebox_dhcp_lease."+resourceName].Primary.Attributes["id"]
						_, err := freeboxClient.GetDHCPStaticLease(ctx, id)
						var apiErr *client.APIError
						Expect(errors.As(err, &apiErr)).To(BeTrue())
						Expect(apiErr.Code).To(Equal("noent"))
						return nil
					},
				})
			})
		})

		Context("changing the MAC address", func() {
			It("should replace the resource", func(ctx SpecContext) {
				newMac := fmt.Sprintf("02:00:%02X:%02X:%02X:%02X",
					randGenerator.Intn(256), randGenerator.Intn(256),
					randGenerator.Intn(256), randGenerator.Intn(256),
				)

				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
						},
						{
							Config: terraformConfigWithAttribute("mac", newMac)(initialConfig),
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply: []plancheck.PlanCheck{
									plancheck.ExpectResourceAction("freebox_dhcp_lease."+resourceName, plancheck.ResourceActionReplace),
								},
							},
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_dhcp_lease."+resourceName, "mac", strings.ToUpper(newMac)),
							),
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						id := s.RootModule().Resources["freebox_dhcp_lease."+resourceName].Primary.Attributes["id"]
						_, err := freeboxClient.GetDHCPStaticLease(ctx, id)
						var apiErr *client.APIError
						Expect(errors.As(err, &apiErr)).To(BeTrue())
						Expect(apiErr.Code).To(Equal("noent"))
						return nil
					},
				})
			})
		})
	})

	Context("import and delete", func() {
		It("should import by MAC address", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: initialConfig,
					},
					{
						Config:             initialConfig,
						ResourceName:       "freebox_dhcp_lease." + resourceName,
						ImportState:        true,
						ImportStateId:      strings.ToUpper(mac),
						ImportStatePersist: true,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_dhcp_lease."+resourceName, "mac", strings.ToUpper(mac)),
							resource.TestCheckResourceAttr("freebox_dhcp_lease."+resourceName, "ip", ip),
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					id := s.RootModule().Resources["freebox_dhcp_lease."+resourceName].Primary.Attributes["id"]
					_, err := freeboxClient.GetDHCPStaticLease(ctx, id)
					var apiErr *client.APIError
					Expect(errors.As(err, &apiErr)).To(BeTrue())
					Expect(apiErr.Code).To(Equal("noent"))
					return nil
				},
			})
		})
	})
})
