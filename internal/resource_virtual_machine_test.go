package internal_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/nikolalohinski/free-go/client"
	"github.com/nikolalohinski/free-go/types"
	freeboxTypes "github.com/nikolalohinski/free-go/types"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Context("resource \"freebox_virtual_machine\" { ... }", Ordered, func() {
	var (
		diskFolder    = "Logiciels"
		diskImageName = "terraform-provider-freebox-alpine-3.20.0-aarch64.qcow2"
		diskImagePath = root + "/" + diskFolder + "/" + diskImageName

		ctx           context.Context
		freeboxClient client.Client
	)
	BeforeAll(func() {
		freeboxClient = Must(client.New(endpoint, version)).
			WithAppID(appID).
			WithPrivateToken(token)
		ctx = context.Background()
		// Login
		permissions := Must(freeboxClient.Login(ctx))
		Expect(permissions.Settings).To(BeTrue(), fmt.Sprintf("the token for the '%s' app does not appear to have the permissions to modify freebox settings", os.Getenv("FREEBOX_APP_ID")))
		// Create directory
		_, err := freeboxClient.CreateDirectory(ctx, root, diskFolder)
		Expect(err).To(Or(BeNil(), Equal(client.ErrDestinationConflict)))
		// Check that the image exists and if so, do an early return
		_, err = freeboxClient.GetFileInfo(ctx, diskImagePath)
		if err == nil {
			return
		}
		if err != nil && err == client.ErrPathNotFound {
			// If not, then pre-download the image
			taskID, err := freeboxClient.AddDownloadTask(ctx, types.DownloadRequest{
				DownloadURLs: []string{
					"https://raw.githubusercontent.com/NikolaLohinski/terraform-provider-freebox/main/examples/alpine-virt-3.20.0-aarch64.qcow2",
				},
				Hash:              "sha256:c7adb3d1fa28cd2abc208e83358a7d065116c6fce1c631ff1d03ace8a992bb69",
				DownloadDirectory: root + "/" + diskFolder,
				Filename:          diskImageName,
			})
			Expect(err).To(BeNil())
			// Wait for download task to be done
			Eventually(func() types.DownloadTask {
				downloadTask, err := freeboxClient.GetDownloadTask(ctx, taskID)
				Expect(err).To(BeNil())
				return downloadTask
			}, "5m").Should(MatchFields(IgnoreExtras, Fields{
				"Status": BeEquivalentTo(types.DownloadTaskStatusDone),
			}))
		} else {
			Expect(err).To(BeNil())
		}
	})
	Context("create and delete (CD)", func() {
		It("should create, start, stop and delete a virtual machine", func() {
			splitName := strings.Split(("test-CD-" + uuid.New().String())[:30], "-")
			name := strings.Join(splitName[:len(splitName)-1], "-")
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerBlock + `
							resource "freebox_virtual_machine" "` + name + `" {
								vcpus     = 1
								memory    = 300
								name      = "` + name + `"
								disk_type = "qcow2"
								disk_path = "` + diskImagePath + `"
								timeouts = {
									kill = "5s" // Alpine image used hangs on SIGTERM and needs a SIGKILL to terminate
								}
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "name", name),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "vcpus", "1"),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "memory", "300"),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "disk_type", freeboxTypes.QCow2Disk),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "disk_path", diskImagePath),
							func(s *terraform.State) error {
								identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+name].Primary.Attributes["id"])
								Expect(err).To(BeNil())
								vm, err := freeboxClient.GetVirtualMachine(ctx, int64(identifier))
								Expect(err).To(BeNil())
								Expect(vm.VCPUs).To(Equal(int64(1)))
								Expect(vm.Memory).To(Equal(int64(300)))
								Expect(vm.Name).To(Equal(name))
								Expect(vm.DiskType).To(Equal(freeboxTypes.QCow2Disk))
								Expect(vm.DiskPath).To(Equal(types.Base64Path(diskImagePath)))
								return nil
							},
						),
					},
					// ImportState Testing
					{
						ResourceName:      "freebox_virtual_machine." + name,
						ImportState:       true,
						ImportStateVerify: true,
					},
				},
			})
		})
	})
	Context("create, update and delete (CUD)", func() {
		var cloudInitConfig = strings.ReplaceAll(`{
			"system_info": {
				"default_user": {
					"name":"freebox"
				}
			},
			"password": "freebox",
			"chpasswd": {
				"expire": false
			},
			"ssh_pwauth":true
		}`, "\n", "")
		It("should create, start, stop, update, start again, stop again and finally delete a virtual machine", func() {
			splitName := strings.Split(("test-CUD-" + uuid.New().String())[:30], "-")
			name := strings.Join(splitName[:len(splitName)-1], "-")
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerBlock + `
							resource "freebox_virtual_machine" "` + name + `" {
								vcpus     = 1
								memory    = 300
								name      = "` + name + `"
								disk_type = "qcow2"
								disk_path = "` + diskImagePath + `"
								timeouts = {
									kill = "5s" // Alpine image used hangs on SIGTERM and needs a SIGKILL to terminate
								}
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "name", name),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "enable_cloudinit", "false"),
						),
					},
					{
						Config: providerBlock + `
							resource "freebox_virtual_machine" "` + name + `" {
								vcpus     = 1
								memory    = 300
								name      = "` + name + `"
								disk_type = "qcow2"
								disk_path = "` + diskImagePath + `"
								timeouts = {
									kill = "5s" // Alpine image used hangs on SIGTERM and needs a SIGKILL to terminate
								}
								enable_cloudinit   = true
								cloudinit_hostname = "` + name + `"
								cloudinit_userdata = yamlencode(jsondecode(<<EOF
								` + cloudInitConfig + `
								EOF
								))
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "name", name),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "enable_cloudinit", "true"),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "cloudinit_hostname", name),
							func(s *terraform.State) error {
								identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+name].Primary.Attributes["id"])
								Expect(err).To(BeNil())
								vm, err := freeboxClient.GetVirtualMachine(ctx, int64(identifier))
								Expect(err).To(BeNil())
								Expect(vm.EnableCloudInit).To(BeTrue())
								Expect(vm.CloudHostName).To(Equal(name))
								Expect(vm.CloudInitUserData).To(MatchYAML(cloudInitConfig))
								return nil
							},
						),
					},
				},
			})
		})
	})
})
