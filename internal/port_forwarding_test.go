package internal_test

import (
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/nikolalohinski/free-go/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("resource \"freebox_port_forwarding\" { ... }", func() {
	Context("create, update and delete (CUD)", func() {
		Describe("No changes", func() {
			It("should create, update and delete the rule", func(ctx SpecContext) {
				splitName := strings.Split(("test-CUD-" + uuid.New().String())[:30], "-")
				name := strings.Join(splitName[:len(splitName)-1], "-")
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: providerBlock + `
								resource "freebox_port_forwarding" "` + name + `" {
									enabled 		 = true
									ip_protocol      = "tcp"
									target_ip        = "192.168.1.1"
									comment 	     = "` + name + `"
									source_ip        = "0.0.0.0"
									source_port      = 32768
									target_port      = 32768
								}
							`,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "enabled", "true"),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "ip_protocol", "tcp"),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "port_range_start", "32768"),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "port_range_end", "32768"),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "target_ip", "192.168.1.1"),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "comment", name),
								resource.TestCheckResourceAttrWith("freebox_port_forwarding."+name, "id", func(value string) error {
									id, err := strconv.Atoi(value)
									if err != nil {
										return err
									}

									portForwardingRule, err := freeboxClient.GetPortForwardingRule(ctx, int64(id))
									if err != nil {
										return err
									}

									Expect(portForwardingRule).ToNot(BeNil())
									Expect(portForwardingRule.WanPortStart).To(Equal(int64(32768)))
									Expect(portForwardingRule.WanPortEnd).To(Equal(int64(32768)))

									return nil
								}),
							),
						},
						{
							Config: providerBlock + `
								resource "freebox_port_forwarding" "` + name + `" {
									enabled 		 = true
									ip_protocol      = "tcp"
									target_ip        = "192.168.1.1"
									comment 	     = "` + name + `"
									source_ip        = "0.0.0.0"
									source_port      = 32768
									target_port      = 32768
								}
							`,
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply: []plancheck.PlanCheck{
									plancheck.ExpectResourceAction("freebox_port_forwarding."+name, plancheck.ResourceActionNoop),
								},
							},
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						id, err := strconv.Atoi(s.RootModule().Resources["freebox_port_forwarding."+name].Primary.Attributes["id"])
						if err != nil {
							return err
						}

						_, err = freeboxClient.GetPortForwardingRule(ctx, int64(id))
						Expect(err).To(Equal(client.ErrPortForwardingRuleNotFound))

						return nil
					},
				})
			})
		})
		Describe("The target_ip changes", func() {
			It("should create, update and delete the rule", func(ctx SpecContext) {
				splitName := strings.Split(("test-CUD-" + uuid.New().String())[:30], "-")
				name := strings.Join(splitName[:len(splitName)-1], "-")
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: providerBlock + `
								resource "freebox_port_forwarding" "` + name + `" {
									enabled 		 = false
									ip_protocol      = "tcp"
									port_range_start = 32768
									port_range_end   = 32768
									target_ip        = "192.168.1.1"
									comment 	     = "` + name + `"
								}
							`,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "enabled", "false"),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "ip_protocol", "tcp"),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "port_range_start", "32768"),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "port_range_end", "32768"),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "target_ip", "192.168.1.1"),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "comment", name),
								resource.TestCheckResourceAttrWith("freebox_port_forwarding."+name, "id", func(value string) error {
									id, err := strconv.Atoi(value)
									if err != nil {
										return err
									}

									portForwardingRule, err := freeboxClient.GetPortForwardingRule(ctx, int64(id))
									if err != nil {
										return err
									}

									Expect(portForwardingRule).ToNot(BeNil())
									Expect(portForwardingRule.WanPortStart).To(Equal(int64(32768)))
									Expect(portForwardingRule.WanPortEnd).To(Equal(int64(32768)))

									return nil
								}),
							),
						},
						{
							Config: providerBlock + `
								resource "freebox_port_forwarding" "` + name + `" {
									enabled 		 = false
									ip_protocol      = "tcp"
									port_range_start = 32768
									port_range_end   = 32768
									target_ip        = "192.168.1.2"
									comment 	     = "` + name + `"
								}
							`,
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply: []plancheck.PlanCheck{
									plancheck.ExpectResourceAction("freebox_port_forwarding."+name, plancheck.ResourceActionUpdate),
								},
							},
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "enabled", "false"),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "ip_protocol", "tcp"),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "port_range_start", "32768"),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "port_range_end", "32768"),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "target_ip", "192.168.1.2"),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "comment", name),
								resource.TestCheckResourceAttrWith("freebox_port_forwarding."+name, "id", func(value string) error {
									id, err := strconv.Atoi(value)
									if err != nil {
										return err
									}

									portForwardingRule, err := freeboxClient.GetPortForwardingRule(ctx, int64(id))
									if err != nil {
										return err
									}

									Expect(portForwardingRule).ToNot(BeNil())
									Expect(portForwardingRule.WanPortStart).To(Equal(int64(32768)))
									Expect(portForwardingRule.WanPortEnd).To(Equal(int64(32768)))

									return nil
								}),
							),
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						id, err := strconv.Atoi(s.RootModule().Resources["freebox_port_forwarding."+name].Primary.Attributes["id"])
						if err != nil {
							return err
						}

						_, err = freeboxClient.GetPortForwardingRule(ctx, int64(id))
						Expect(err).To(Equal(client.ErrPortForwardingRuleNotFound))

						return nil
					},
				})
			})
		})
	})
})
