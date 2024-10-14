package internal_test

import (
	"strconv"
	"strings"

	"github.com/nikolalohinski/free-go/client"
	"github.com/nikolalohinski/free-go/types"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Context("resource \"freebox_virtual_machine\" { ... }", Ordered, func() {
	var (
		testDisks = map[string]struct {
			filename  string
			directory string
			filepath  string
			digest    string
			source    string
		}{
			"alpine": existingDisk,
			"ubuntu": {
				filename:  "terraform-provider-freebox-ubuntu-22.04-aarch64.qcow2",
				directory: "VMs",
				filepath:  root + "/VMs/terraform-provider-freebox-ubuntu-22.04-aarch64.qcow2",
				digest:    "http://ftp.free.fr/.private/ubuntu-cloud/releases/jammy/release/SHA256SUMS",
				source:    "http://ftp.free.fr/.private/ubuntu-cloud/releases/jammy/release/ubuntu-22.04-server-cloudimg-arm64.img",
			},
		}
	)
	BeforeAll(func(ctx SpecContext) {
		// Note: alpine already exists
		// Download the ubuntu image
		disk := testDisks["ubuntu"]
		// Create directory
		_, err := freeboxClient.CreateDirectory(ctx, root, disk.directory)
		Expect(err).To(Or(BeNil(), Equal(client.ErrDestinationConflict)))
		// Check that the image exists and if so, do an early return
		_, err = freeboxClient.GetFileInfo(ctx, disk.filepath)
		if err == nil {
			return
		}
		if err != nil && err == client.ErrPathNotFound {
			// If not, then pre-download the image
			taskID, err := freeboxClient.AddDownloadTask(ctx, types.DownloadRequest{
				DownloadURLs:      []string{disk.source},
				Hash:              disk.digest,
				DownloadDirectory: root + "/" + disk.directory,
				Filename:          disk.filename,
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
		It("should create, start, stop and delete a virtual machine", func(ctx SpecContext) {
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
								disk_path = "` + testDisks["alpine"].filepath + `"
								timeouts = {
									kill       = "500ms" // The image used for tests hangs on SIGTERM and needs a SIGKILL to terminate
									networking = "0s" // The image used for tests does not register to the network
								}
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "name", name),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "vcpus", "1"),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "memory", "300"),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "disk_type", types.QCow2Disk),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "disk_path", testDisks["alpine"].filepath),
							func(s *terraform.State) error {
								identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+name].Primary.Attributes["id"])
								Expect(err).To(BeNil())
								vm, err := freeboxClient.GetVirtualMachine(ctx, int64(identifier))
								Expect(err).To(BeNil())
								Expect(vm.VCPUs).To(Equal(int64(1)))
								Expect(vm.Memory).To(Equal(int64(300)))
								Expect(vm.Name).To(Equal(name))
								Expect(vm.DiskType).To(Equal(types.QCow2Disk))
								Expect(vm.DiskPath).To(Equal(types.Base64Path(testDisks["alpine"].filepath)))
								return nil
							},
						),
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
		It("should create, start, stop, update, start again, stop again and finally delete a virtual machine", func(ctx SpecContext) {
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
								disk_path = "` + testDisks["ubuntu"].filepath + `"
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
								disk_path = "` + testDisks["ubuntu"].filepath + `"
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
	Context("import and delete (ID)", func() {
		var (
			virtualMachineID = new(int64)
			name             = new(string)
		)
		BeforeEach(func(ctx SpecContext) {
			splitName := strings.Split(("test-ID-" + uuid.New().String())[:30], "-")
			*name = strings.Join(splitName[:len(splitName)-1], "-")
			vm := Must(freeboxClient.CreateVirtualMachine(ctx, types.VirtualMachinePayload{
				Name:     *name,
				VCPUs:    1,
				Memory:   2000,
				DiskType: types.QCow2Disk,
				DiskPath: types.Base64Path(testDisks["alpine"].filepath),
			}))
			*virtualMachineID = vm.ID
		})
		It("should import and then delete a virtual machine", func() {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerBlock + `
							resource "freebox_virtual_machine" "` + *name + `" {
								vcpus     = 1
								memory    = 2000
								name      = "` + *name + `"
								disk_type = "qcow2"
								disk_path = "` + testDisks["alpine"].filepath + `"
								timeouts = {
									networking = "0s" // The image used for tests does not register to the network
								}
							}
						`,
						ResourceName:       "freebox_virtual_machine." + *name,
						ImportState:        true,
						ImportStateId:      strconv.Itoa(int(*virtualMachineID)),
						ImportStatePersist: true,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_machine."+*name, "name", *name),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+*name, "vcpus", "1"),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+*name, "memory", "300"),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+*name, "disk_type", types.QCow2Disk),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+*name, "disk_path", testDisks["ubuntu"].filepath),
						),
						Destroy: true,
					},
				},
			})
		})
	})
})
