package internal_test

import (
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("resource \"freebox_port_forwarding\" { ... }", func() {
	Context("create, update and delete (CUD)", func() {
		It("should create, update and delete the rule", func() {
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
  							    port_range_start = 8000
  							    port_range_end   = 8000
  							    target_ip        = "192.168.1.1"
								comment 	     = "` + name + `"
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "enabled", "false"),
							resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "ip_protocol", "tcp"),
							resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "port_range_start", "8000"),
							resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "port_range_end", "8000"),
							resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "target_ip", "192.168.1.1"),
							resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "comment", name),
						),
					},
					{
						Config: providerBlock + `
							resource "freebox_port_forwarding" "` + name + `" {
								enabled 		 = false
  							    ip_protocol      = "tcp"
  							    port_range_start = 8000
  							    port_range_end   = 8000
  							    target_ip        = "192.168.1.2"
								comment 	     = "` + name + `"
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "enabled", "false"),
							resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "ip_protocol", "tcp"),
							resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "port_range_start", "8000"),
							resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "port_range_end", "8000"),
							resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "target_ip", "192.168.1.2"),
							resource.TestCheckResourceAttr("freebox_port_forwarding."+name, "comment", name),
						),
					},
				},
			})
		})
	})
})
