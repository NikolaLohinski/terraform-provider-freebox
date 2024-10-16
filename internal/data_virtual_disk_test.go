package internal_test

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/nikolalohinski/free-go/client"
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("DataVirtualDisk", func() {
	Context("with an existing disk", func() {
		It("should fetch disk information", func() {
			splitName := strings.Split(("test-CD-" + uuid.New().String())[:30], "-")
			name := strings.Join(splitName[:len(splitName)-1], "-")
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerBlock + `
							data "freebox_virtual_disk" "` + name + `" {
								path = "` + existingDisk.filepath + `"
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.freebox_virtual_disk."+name, "path", existingDisk.filepath),
							resource.TestCheckResourceAttr("data.freebox_virtual_disk."+name, "type", "qcow2"),
							resource.TestCheckResourceAttr("data.freebox_virtual_disk."+name, "actual_size", "72220672"),
							resource.TestCheckResourceAttr("data.freebox_virtual_disk."+name, "virtual_size", "72800256"),
						),
					},
				},
			})
		})
	})

	Context("with an non existing disk", func() {
		It("should fail", func() {
			splitName := strings.Split(("test-CD-" + uuid.New().String())[:30], "-")
			name := strings.Join(splitName[:len(splitName)-1], "-")
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerBlock + `
							data "freebox_virtual_disk" "` + name + `" {
								path = "/terraform-provider/non/existing/path"
							}
						`,
						ExpectError: regexp.MustCompile(regexp.QuoteMeta(client.ErrPathNotFound.Error())),
					},
				},
			})
		})
	})
})
