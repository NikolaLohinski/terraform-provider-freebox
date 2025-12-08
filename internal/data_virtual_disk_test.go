package internal_test

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DataVirtualDisk", func() {
	var (
		config       string
		diskPath     string
		resourceName string
	)

	BeforeEach(func(ctx SpecContext) {
		splitName := strings.Split(("test-CD-" + uuid.New().String())[:30], "-")
		resourceName = strings.Join(splitName[:len(splitName)-1], "-")
	})

	JustBeforeEach(func(ctx SpecContext) {
		config = providerBlock + `
			data "freebox_virtual_disk" "` + resourceName + `" {
				path = "` + diskPath + `"
			}
		`
	})

	Context("with an existing disk", func() {
		var diskType string

		BeforeEach(func(ctx SpecContext) {
			diskPath = existingDisk.filepath

			Expect(existingDisk.filepath).To(HaveSuffix(".qcow2"))
			diskType = freeboxTypes.QCow2Disk
		})

		It("should fetch disk information", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.freebox_virtual_disk."+resourceName, "path", existingDisk.filepath),
							resource.TestCheckResourceAttr("data.freebox_virtual_disk."+resourceName, "type", diskType),
							resource.TestCheckResourceAttr("data.freebox_virtual_disk."+resourceName, "actual_size", "72224768"),
							resource.TestCheckResourceAttr("data.freebox_virtual_disk."+resourceName, "virtual_size", "72800256"),
						),
					},
				},
			})
		})
	})

	Context("with an non existing disk", func() {
		BeforeEach(func(ctx SpecContext) {
			diskPath = "/terraform-provider/non/existing/path"
		})

		It("should fail", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile(regexp.QuoteMeta(client.ErrPathNotFound.Error())),
					},
				},
			})
		})
	})
})
