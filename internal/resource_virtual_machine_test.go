package internal_test

import (
	"strings"

	"github.com/MakeNowJust/heredoc"
	freeboxTypes "github.com/nikolalohinski/free-go/types"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	. "github.com/onsi/ginkgo/v2"
	// . "github.com/onsi/gomega"
	// . "github.com/onsi/gomega/gstruct"
)

var _ = Context("resource freebox_virtual_machine", Ordered, func() {
	Context("simplest create and delete", func() {
		It("should create, start, stop and delete a virtual machine", func() {
			splitName := strings.Split(("test-" + uuid.New().String())[:30], "-")
			name := strings.Join(splitName[:len(splitName)-1], "-")
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: heredoc.Doc(`
							resource "freebox_virtual_machine" "` + name + `" {
								vcpus     = 1
								memory    = 300
								name      = "` + name + `"
								disk_type = "qcow2"
								disk_path = "Freebox/free-go/free-go.integration.tests.qcow2" // TODO: download the image instead of expecting it to exist
							}`,
						),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "name", name),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "vcpus", "1"),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "memory", "300"),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "disk_type", freeboxTypes.QCow2Disk),
							func(s *terraform.State) error {
								return nil
							},
						),
					},
				},
			})
		})
	})
})
