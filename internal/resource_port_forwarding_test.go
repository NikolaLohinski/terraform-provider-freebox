package internal_test

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/nikolalohinski/free-go/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
)

var _ = Describe("resource \"freebox_port_forwarding\" { ... }", func() {
	var (
		enabled        bool
		ipProtocol     string
		resourceName   string
		initialConfig  string
		portRangeStart int64
		portRangeEnd   int64
		sourceIP       string
		targetIP       string
		targetPort     int64
	)

	BeforeEach(func(ctx SpecContext) {
		splitName := strings.Split(("test-" + uuid.New().String())[:30], "-")
		resourceName = strings.Join(splitName[:len(splitName)-1], "-")

		enabled = true
		ipProtocol = "tcp"

		sourceSubnet := randGenerator.Int63n(255)
		sourceIP = fmt.Sprintf("192.168.%d.%d", sourceSubnet, randGenerator.Int63n(255)+1)
		targetIP = fmt.Sprintf("192.168.%d.%d", sourceSubnet+1%255, randGenerator.Int63n(255)+1)
		targetPort = randGenerator.Int63n(65353) + 1

		portRangeStart = randGenerator.Int63n(65353) + 1
		portRangeEnd = randGenerator.Int63n(65353-portRangeStart) + portRangeStart + 1
	})

	JustBeforeEach(func(ctx SpecContext) {
		initialConfig = providerBlock + `
			resource "freebox_port_forwarding" "` + resourceName + `" {
				enabled 		  = ` + strconv.FormatBool(enabled) + `
				ip_protocol       = "` + ipProtocol + `"
				comment 	      = "` + resourceName + `"
				source_ip         = "` + sourceIP + `"
				port_range_start  = ` + strconv.FormatInt(portRangeStart, 10) + `
				port_range_end    = ` + strconv.FormatInt(portRangeEnd, 10) + `
				target_ip         = "` + targetIP + `"
				target_port       = ` + strconv.FormatInt(targetPort, 10) + `
			}
		`
	})

	Context("create and delete", func() {
		Describe("Using a range of ports", func() {
			JustBeforeEach(func(ctx SpecContext) {
				initialConfig = terraformConfigWithoutAttribute(`target_port`)(initialConfig)
			})

			It("should create and delete the rule", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "enabled", strconv.FormatBool(enabled)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "ip_protocol", ipProtocol),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "comment", resourceName),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "source_ip", sourceIP),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "port_range_start", strconv.FormatInt(portRangeStart, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "port_range_end", strconv.FormatInt(portRangeEnd, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "target_port", strconv.FormatInt(portRangeStart, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "target_ip", targetIP),
								resource.TestCheckResourceAttrWith("freebox_port_forwarding."+resourceName, "id", func(resourceID string) error {
									portForwardingRuleID, err := strconv.Atoi(resourceID)
									Expect(err).ToNot(HaveOccurred())

									portForwardingRule, err := freeboxClient.GetPortForwardingRule(ctx, int64(portForwardingRuleID))
									Expect(err).ToNot(HaveOccurred())

									Expect(portForwardingRule).ToNot(BeNil())
									Expect(portForwardingRule.Enabled).To(gstruct.PointTo(Equal(enabled)))
									Expect(portForwardingRule.IPProtocol).To(Equal(ipProtocol))
									Expect(portForwardingRule.Comment).To(Equal(resourceName))
									Expect(portForwardingRule.SourceIP).To(Equal(sourceIP))
									Expect(portForwardingRule.WanPortStart).To(Equal(portRangeStart))
									Expect(portForwardingRule.WanPortEnd).To(Equal(portRangeEnd))
									Expect(portForwardingRule.LanPort).To(Equal(portRangeStart))
									Expect(portForwardingRule.LanIP).To(Equal(targetIP))

									return nil
								}),
							),
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						id, err := strconv.Atoi(s.RootModule().Resources["freebox_port_forwarding."+resourceName].Primary.Attributes["id"])
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

		Describe("Using a single port", func() {
			JustBeforeEach(func(ctx SpecContext) {
				initialConfig = terraformConfigWithoutAttribute(`port_range_end`)(initialConfig)
			})

			It("should create and delete the rule", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "enabled", strconv.FormatBool(enabled)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "ip_protocol", ipProtocol),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "comment", resourceName),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "source_ip", sourceIP),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "port_range_start", strconv.FormatInt(portRangeStart, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "port_range_end", strconv.FormatInt(portRangeStart, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "target_port", strconv.FormatInt(targetPort, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "target_ip", targetIP),
								resource.TestCheckResourceAttrWith("freebox_port_forwarding."+resourceName, "id", func(resourceID string) error {
									portForwardingRuleID, err := strconv.Atoi(resourceID)
									Expect(err).ToNot(HaveOccurred())

									portForwardingRule, err := freeboxClient.GetPortForwardingRule(ctx, int64(portForwardingRuleID))
									Expect(err).ToNot(HaveOccurred())

									Expect(portForwardingRule).ToNot(BeNil())
									Expect(portForwardingRule.Enabled).To(gstruct.PointTo(Equal(enabled)))
									Expect(portForwardingRule.IPProtocol).To(Equal(ipProtocol))
									Expect(portForwardingRule.Comment).To(Equal(resourceName))
									Expect(portForwardingRule.SourceIP).To(Equal(sourceIP))
									Expect(portForwardingRule.WanPortStart).To(Equal(portRangeStart))
									Expect(portForwardingRule.WanPortEnd).To(Equal(portRangeStart))
									Expect(portForwardingRule.LanPort).To(Equal(targetPort))
									Expect(portForwardingRule.LanIP).To(Equal(targetIP))

									return nil
								}),
							),
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						id, err := strconv.Atoi(s.RootModule().Resources["freebox_port_forwarding."+resourceName].Primary.Attributes["id"])
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

	Context("create, update and delete", func() {
		Describe("Using a single port", func() {
			var (
				newPortRangeStart int64
				newTargetPort     int64
			)

			BeforeEach(func(ctx SpecContext) {
				newPortRangeStart = (portRangeStart % 65353) + 1
				newTargetPort = (targetPort % 65353) + 1
			})

			JustBeforeEach(func(ctx SpecContext) {
				initialConfig = terraformConfigWithoutAttribute("port_range_end")(initialConfig)
			})

			It("should update the rule", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "port_range_start", strconv.FormatInt(portRangeStart, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "port_range_end", strconv.FormatInt(portRangeStart, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "target_port", strconv.FormatInt(targetPort, 10)),
								resource.TestCheckResourceAttrWith("freebox_port_forwarding."+resourceName, "id", func(value string) error {
									id, err := strconv.Atoi(value)
									Expect(err).ToNot(HaveOccurred())

									portForwardingRule, err := freeboxClient.GetPortForwardingRule(ctx, int64(id))
									Expect(err).ToNot(HaveOccurred())

									Expect(portForwardingRule).ToNot(BeNil())
									Expect(portForwardingRule.WanPortStart).To(Equal(portRangeStart))
									Expect(portForwardingRule.WanPortEnd).To(Equal(portRangeStart))

									return nil
								}),
							),
						},
						{
							Config: terraformConfigWithAttribute(`port_range_start`, newPortRangeStart)(initialConfig),
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply: []plancheck.PlanCheck{
									plancheck.ExpectResourceAction("freebox_port_forwarding."+resourceName, plancheck.ResourceActionUpdate),
								},
							},
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "port_range_start", strconv.FormatInt(newPortRangeStart, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "port_range_end", strconv.FormatInt(newPortRangeStart, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "target_port", strconv.FormatInt(targetPort, 10)),
								resource.TestCheckResourceAttrWith("freebox_port_forwarding."+resourceName, "id", func(value string) error {
									id, err := strconv.Atoi(value)
									Expect(err).ToNot(HaveOccurred())

									portForwardingRule, err := freeboxClient.GetPortForwardingRule(ctx, int64(id))
									Expect(err).ToNot(HaveOccurred())

									Expect(portForwardingRule).ToNot(BeNil())
									Expect(portForwardingRule.WanPortStart).To(Equal(newPortRangeStart))
									Expect(portForwardingRule.WanPortEnd).To(Equal(newPortRangeStart))

									return nil
								}),
							),
						},
						{
							Config: terraformConfigWithAttribute(`target_port`, newTargetPort)(initialConfig),
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply: []plancheck.PlanCheck{
									plancheck.ExpectResourceAction("freebox_port_forwarding."+resourceName, plancheck.ResourceActionUpdate),
								},
							},
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "port_range_start", strconv.FormatInt(portRangeStart, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "port_range_end", strconv.FormatInt(portRangeStart, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "target_port", strconv.FormatInt(newTargetPort, 10)),
								resource.TestCheckResourceAttrWith("freebox_port_forwarding."+resourceName, "id", func(value string) error {
									id, err := strconv.Atoi(value)
									Expect(err).ToNot(HaveOccurred())

									portForwardingRule, err := freeboxClient.GetPortForwardingRule(ctx, int64(id))
									Expect(err).ToNot(HaveOccurred())

									Expect(portForwardingRule).ToNot(BeNil())
									Expect(portForwardingRule.WanPortStart).To(Equal(portRangeStart))
									Expect(portForwardingRule.WanPortEnd).To(Equal(portRangeStart))
									Expect(portForwardingRule.LanPort).To(Equal(newTargetPort))

									return nil
								}),
							),
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						id, err := strconv.Atoi(s.RootModule().Resources["freebox_port_forwarding."+resourceName].Primary.Attributes["id"])
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

		Describe("Using a range of ports", func() {
			var (
				newPortRangeStart int64
				newPortRangeEnd   int64
			)

			BeforeEach(func(ctx SpecContext) {
				newPortRangeStart = (portRangeStart % 65353) + 1
				newPortRangeEnd = randGenerator.Int63n(65353-newPortRangeStart) + newPortRangeStart + 1
			})

			JustBeforeEach(func(ctx SpecContext) {
				initialConfig = terraformConfigWithoutAttribute(`target_port`)(initialConfig)
			})

			It("should update the rule", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "port_range_start", strconv.FormatInt(portRangeStart, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "port_range_end", strconv.FormatInt(portRangeEnd, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "target_port", strconv.FormatInt(portRangeStart, 10)),
								resource.TestCheckResourceAttrWith("freebox_port_forwarding."+resourceName, "id", func(value string) error {
									id, err := strconv.Atoi(value)
									Expect(err).ToNot(HaveOccurred())

									portForwardingRule, err := freeboxClient.GetPortForwardingRule(ctx, int64(id))
									Expect(err).ToNot(HaveOccurred())

									Expect(portForwardingRule).ToNot(BeNil())
									Expect(portForwardingRule.WanPortStart).To(Equal(portRangeStart))
									Expect(portForwardingRule.WanPortEnd).To(Equal(portRangeEnd))
									Expect(portForwardingRule.LanPort).To(Equal(portRangeStart))

									return nil
								}),
							),
						},
						{
							Config: terraformConfigWithAttribute(`port_range_start`, newPortRangeStart)(terraformConfigWithAttribute(`port_range_end`, newPortRangeEnd)(initialConfig)),
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply: []plancheck.PlanCheck{
									plancheck.ExpectResourceAction("freebox_port_forwarding."+resourceName, plancheck.ResourceActionUpdate),
								},
							},
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "port_range_start", strconv.FormatInt(newPortRangeStart, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "port_range_end", strconv.FormatInt(newPortRangeEnd, 10)),
								resource.TestCheckResourceAttr("freebox_port_forwarding."+resourceName, "target_port", strconv.FormatInt(portRangeStart, 10)),
								resource.TestCheckResourceAttrWith("freebox_port_forwarding."+resourceName, "id", func(value string) error {
									id, err := strconv.Atoi(value)
									Expect(err).ToNot(HaveOccurred())

									portForwardingRule, err := freeboxClient.GetPortForwardingRule(ctx, int64(id))
									Expect(err).ToNot(HaveOccurred())

									Expect(portForwardingRule).ToNot(BeNil())
									Expect(portForwardingRule.WanPortStart).To(Equal(newPortRangeStart))
									Expect(portForwardingRule.WanPortEnd).To(Equal(newPortRangeEnd))
									Expect(portForwardingRule.LanPort).To(Equal(newPortRangeStart))

									return nil
								}),
							),
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						id, err := strconv.Atoi(s.RootModule().Resources["freebox_port_forwarding."+resourceName].Primary.Attributes["id"])
						Expect(err).ToNot(HaveOccurred())

						_, err = freeboxClient.GetPortForwardingRule(ctx, int64(id))
						Expect(err).To(Equal(client.ErrPortForwardingRuleNotFound))

						return nil
					},
				})
			})
		})
	})
})
