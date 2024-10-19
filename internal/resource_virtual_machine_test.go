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
)

var _ = Context("resource \"freebox_virtual_machine\" { ... }", Ordered, func() {
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
								disk_path = "` + existingDisk.filepath + `"
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
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "disk_path", existingDisk.filepath),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "status", "running"),
							func(s *terraform.State) error {
								identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+name].Primary.Attributes["id"])
								Expect(err).To(BeNil())
								vm, err := freeboxClient.GetVirtualMachine(ctx, int64(identifier))
								Expect(err).To(BeNil())
								Expect(vm.VCPUs).To(Equal(int64(1)))
								Expect(vm.Memory).To(Equal(int64(300)))
								Expect(vm.Name).To(Equal(name))
								Expect(vm.DiskType).To(Equal(types.QCow2Disk))
								Expect(vm.DiskPath).To(Equal(types.Base64Path(existingDisk.filepath)))
								Expect(vm.Status).To(BeEquivalentTo(types.RunningStatus))
								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					id, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+name].Primary.Attributes["id"])
					Expect(err).To(BeNil())

					_, err = freeboxClient.GetVirtualMachine(ctx, int64(id))
					Expect(err).To(MatchError(client.ErrVirtualMachineNotFound), "virtual machine %d should not exist", id)

					return nil
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
								status    = "stopped"
								disk_path = "` + existingDisk.filepath + `"
								timeouts = {
									kill       = "500ms" // The image used for tests hangs on SIGTERM and needs a SIGKILL to terminate
									networking = "0s" // The image used for tests does not register to the network
								}
							}

						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "name", name),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "enable_cloudinit", "false"),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "status", "stopped"),
							func(s *terraform.State) error {
								identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+name].Primary.Attributes["id"])
								Expect(err).To(BeNil())
								vm, err := freeboxClient.GetVirtualMachine(ctx, int64(identifier))
								Expect(err).To(BeNil())
								Expect(vm.Status).To(BeEquivalentTo(types.StoppedStatus))
								return nil
							},
						),
					},
					{
						Config: providerBlock + `
							resource "freebox_virtual_machine" "` + name + `" {
								vcpus     = 1
								memory    = 300
								name      = "` + name + `"
								disk_type = "qcow2"
								status    = "running"
								disk_path = "` + existingDisk.filepath + `"
								enable_cloudinit   = true
								cloudinit_hostname = "` + name + `"
								cloudinit_userdata = yamlencode(jsondecode(<<EOF
								` + cloudInitConfig + `
								EOF
								))
								timeouts = {
									kill       = "500ms" // The image used for tests hangs on SIGTERM and needs a SIGKILL to terminate
									networking = "0s" // The image used for tests does not register to the network
								}
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "name", name),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "enable_cloudinit", "true"),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "cloudinit_hostname", name),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "status", "running"),
							func(s *terraform.State) error {
								identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+name].Primary.Attributes["id"])
								Expect(err).To(BeNil())
								vm, err := freeboxClient.GetVirtualMachine(ctx, int64(identifier))
								Expect(err).To(BeNil())
								Expect(vm.EnableCloudInit).To(BeTrue())
								Expect(vm.CloudHostName).To(Equal(name))
								Expect(vm.CloudInitUserData).To(MatchYAML(cloudInitConfig))
								Expect(vm.Status).To(BeEquivalentTo(types.RunningStatus))
								return nil
							},
						),
					},
					{
						Config: providerBlock + `
							resource "freebox_virtual_machine" "` + name + `" {
								vcpus     = 1
								memory    = 300
								name      = "` + name + `"
								disk_type = "qcow2"
								status    = "stopped"
								disk_path = "` + existingDisk.filepath + `"
								enable_cloudinit   = true
								cloudinit_hostname = "` + name + `"
								cloudinit_userdata = yamlencode(jsondecode(<<EOF
								` + cloudInitConfig + `
								EOF
								))
								timeouts = {
									kill       = "500ms" // The image used for tests hangs on SIGTERM and needs a SIGKILL to terminate
									networking = "0s" // The image used for tests does not register to the network
								}
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "name", name),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "enable_cloudinit", "true"),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "cloudinit_hostname", name),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+name, "status", "stopped"),
							func(s *terraform.State) error {
								identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+name].Primary.Attributes["id"])
								Expect(err).To(BeNil())
								vm, err := freeboxClient.GetVirtualMachine(ctx, int64(identifier))
								Expect(err).To(BeNil())
								Expect(vm.EnableCloudInit).To(BeTrue())
								Expect(vm.CloudHostName).To(Equal(name))
								Expect(vm.CloudInitUserData).To(MatchYAML(cloudInitConfig))
								Expect(vm.Status).To(BeEquivalentTo(types.StoppedStatus))
								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					id, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+name].Primary.Attributes["id"])
					Expect(err).To(BeNil())

					_, err = freeboxClient.GetVirtualMachine(ctx, int64(id))
					Expect(err).To(MatchError(client.ErrVirtualMachineNotFound), "virtual machine %d should not exist", id)

					return nil
				},
			})
		})
	})
	// Flaky test needs some investigation
	XContext("import and delete (ID)", func() {
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
				DiskPath: types.Base64Path(existingDisk.filepath),
			}))
			*virtualMachineID = vm.ID
		})
		It("should import and then delete a virtual machine", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerBlock + `
							resource "freebox_virtual_machine" "` + *name + `" {
								vcpus     = 1
								memory    = 300
								name      = "` + *name + `"
								disk_type = "qcow2"
								disk_path = "` + existingDisk.filepath + `"
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
							resource.TestCheckResourceAttr("freebox_virtual_machine."+*name, "disk_path", existingDisk.filepath),
						),
						Destroy: true,
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					id, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+*name].Primary.Attributes["id"])
					Expect(err).To(BeNil())

					_, err = freeboxClient.GetVirtualMachine(ctx, int64(id))
					Expect(err).To(MatchError(client.ErrVirtualMachineNotFound), "virtual machine %d should not exist", id)

					return nil
				},
			})
		})
	})
})
