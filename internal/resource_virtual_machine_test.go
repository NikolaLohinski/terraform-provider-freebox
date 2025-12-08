package internal_test

import (
	"strconv"
	"strings"

	"github.com/nikolalohinski/free-go/client"
	"github.com/nikolalohinski/free-go/types"
	"gopkg.in/yaml.v3"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("resource \"freebox_virtual_machine\" { ... }", Ordered, func() {
	const diskType = "qcow2"

	var (
		resourceName string

		initialConfig string
		status        string
	)

	BeforeEach(func(ctx SpecContext) {
		splitName := strings.Split(("test-" + uuid.New().String())[:30], "-")
		resourceName = strings.Join(splitName[:len(splitName)-1], "-")

		status = "running"
	})

	JustBeforeEach(func(ctx SpecContext) {
		initialConfig = providerBlock + `
			resource "freebox_virtual_machine" "` + resourceName + `" {
				vcpus     = 1
				memory    = 300
				name      = "` + resourceName + `"
				disk_type = "` + diskType + `"
				disk_path = "` + existingDisk.filepath + `"
				status    = "` + status + `"

				enable_cloudinit = false
				cloudinit_hostname = null
				cloudinit_userdata = null

				timeouts = {
					kill       = "500ms" // The image used for tests hangs on SIGTERM and needs a SIGKILL to be killed
					networking = "0s" // The image used for tests does not register to the network
				}
			}
		`
	})

	BeforeEach(func(ctx SpecContext) {
		Expect(existingDisk.filepath).To(HaveSuffix(".qcow2"))
	})

	Context("create and delete (CD)", func() {
		It("should create, start, stop and delete a virtual machine", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: initialConfig,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "name", resourceName),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "vcpus", "1"),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "memory", "300"),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "disk_type", types.QCow2Disk),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "disk_path", existingDisk.filepath),
							resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "status", "running"),
							func(s *terraform.State) error {
								identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+resourceName].Primary.Attributes["id"])
								Expect(err).To(BeNil())
								vm, err := freeboxClient.GetVirtualMachine(ctx, int64(identifier))
								Expect(err).To(BeNil())
								Expect(vm.VCPUs).To(Equal(int64(1)))
								Expect(vm.Memory).To(Equal(int64(300)))
								Expect(vm.Name).To(Equal(resourceName))
								Expect(vm.DiskType).To(Equal(types.QCow2Disk))
								Expect(vm.DiskPath).To(Equal(types.Base64Path(existingDisk.filepath)))
								Expect(vm.Status).To(BeEquivalentTo(types.RunningStatus))
								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					id, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+resourceName].Primary.Attributes["id"])
					Expect(err).To(BeNil())

					_, err = freeboxClient.GetVirtualMachine(ctx, int64(id))
					Expect(err).To(MatchError(client.ErrVirtualMachineNotFound), "virtual machine %d should not exist", id)

					return nil
				},
			})
		})

		Context("when the status is unspecified", func() {
			JustBeforeEach(func(ctx SpecContext) {
				initialConfig = terraformConfigWithoutAttribute("status")(initialConfig)
			})
			It("should start the virtual machine", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "status", "running"),
							),
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						id, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+resourceName].Primary.Attributes["id"])
						Expect(err).To(BeNil())

						_, err = freeboxClient.GetVirtualMachine(ctx, int64(id))
						Expect(err).To(MatchError(client.ErrVirtualMachineNotFound), "virtual machine %d should not exist", id)

						return nil
					},
				})
			})
		})

		Context("when the status is stopped", func() {
			BeforeEach(func(ctx SpecContext) {
				status = "stopped"
			})
			It("should create a stopped virtual machine", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "status", "stopped"),
							),
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						id, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+resourceName].Primary.Attributes["id"])
						Expect(err).To(BeNil())

						_, err = freeboxClient.GetVirtualMachine(ctx, int64(id))
						Expect(err).To(MatchError(client.ErrVirtualMachineNotFound), "virtual machine %d should not exist", id)

						return nil
					},
				})
			})
		})
	})
	Context("create, update and delete", func() {
		var (
			newConfig string
		)

		JustBeforeEach(func(ctx SpecContext) {
			newConfig = initialConfig
		})

		Context("when the cloudinit is enabled", func() {
			var cloudInitConfig string

			BeforeEach(func(ctx SpecContext) {
				cloudInitConfigBytes, err := yaml.Marshal(map[string]any{
					"system_info": map[string]any{
						"default_user": map[string]any{
							"name": "freebox",
						},
					},
					"password": "freebox",
					"chpasswd": map[string]any{
						"expire": false,
					},
					"ssh_pwauth": true,
				})
				Expect(err).ToNot(HaveOccurred())

				cloudInitConfig = string(cloudInitConfigBytes)
			})

			JustBeforeEach(func(ctx SpecContext) {
				newConfig = terraformConfigWithAttribute("enable_cloudinit", true)(newConfig)
				newConfig = terraformConfigWithAttribute("cloudinit_hostname", resourceName)(newConfig)
				newConfig = terraformConfigWithAttribute("cloudinit_userdata", cloudInitConfig)(newConfig)
			})

			It("should add cloudinit to a virtual machine", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "name", resourceName),
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "enable_cloudinit", "false"),
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "status", "running"),
								func(s *terraform.State) error {
									identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+resourceName].Primary.Attributes["id"])
									Expect(err).To(BeNil())
									vm, err := freeboxClient.GetVirtualMachine(ctx, int64(identifier))
									Expect(err).To(BeNil())
									Expect(vm.Status).To(BeEquivalentTo(types.RunningStatus))
									return nil
								},
							),
						},
						{
							Config: newConfig,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "name", resourceName),
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "enable_cloudinit", "true"),
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "cloudinit_hostname", resourceName),
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "status", "running"),
								func(s *terraform.State) error {
									identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+resourceName].Primary.Attributes["id"])
									Expect(err).To(BeNil())
									vm, err := freeboxClient.GetVirtualMachine(ctx, int64(identifier))
									Expect(err).To(BeNil())
									Expect(vm.EnableCloudInit).To(BeTrue())
									Expect(vm.CloudHostName).To(Equal(resourceName))
									Expect(vm.CloudInitUserData).To(MatchYAML(cloudInitConfig))
									Expect(vm.Status).To(BeEquivalentTo(types.RunningStatus))
									return nil
								},
							),
						},
					},
				})
			})
		})

		Context("when the state changes", func() {
			JustBeforeEach(func(ctx SpecContext) {
				newConfig = terraformConfigWithAttribute("status", "stopped")(newConfig)
			})

			It("should add cloudinit to a virtual machine", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "name", resourceName),
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "enable_cloudinit", "false"),
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "status", "running"),
								func(s *terraform.State) error {
									identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+resourceName].Primary.Attributes["id"])
									Expect(err).To(BeNil())
									vm, err := freeboxClient.GetVirtualMachine(ctx, int64(identifier))
									Expect(err).To(BeNil())
									Expect(vm.Status).To(BeEquivalentTo(types.RunningStatus))
									return nil
								},
							),
						},
						{
							Config: newConfig,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "name", resourceName),
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "status", "stopped"),
								func(s *terraform.State) error {
									identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+resourceName].Primary.Attributes["id"])
									Expect(err).To(BeNil())
									vm, err := freeboxClient.GetVirtualMachine(ctx, int64(identifier))
									Expect(err).To(BeNil())
									Expect(vm.Status).To(BeEquivalentTo(types.StoppedStatus))
									return nil
								},
							),
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						id, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+resourceName].Primary.Attributes["id"])
						Expect(err).To(BeNil())

						_, err = freeboxClient.GetVirtualMachine(ctx, int64(id))
						Expect(err).To(MatchError(client.ErrVirtualMachineNotFound), "virtual machine %d should not exist", id)

						return nil
					},
				})
			})
		})
	})

	Context("import and delete", func() {
		Context("from virtual machine ID", func() {
			var (
				virtualMachineID = new(int64)
			)
			BeforeEach(func(ctx SpecContext) {
				vm := Must(freeboxClient.CreateVirtualMachine(ctx, types.VirtualMachinePayload{
					Name:     resourceName,
					VCPUs:    1,
					Memory:   2000,
					DiskType: types.QCow2Disk,
					DiskPath: types.Base64Path(existingDisk.filepath),
				}))
				*virtualMachineID = vm.ID
			})

			It("should work", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config:             initialConfig,
							ResourceName:       "freebox_virtual_machine." + resourceName,
							ImportState:        true,
							ImportStateId:      strconv.Itoa(int(*virtualMachineID)),
							ImportStatePersist: true,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "name", resourceName),
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "vcpus", "1"),
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "memory", "300"),
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "disk_type", types.QCow2Disk),
								resource.TestCheckResourceAttr("freebox_virtual_machine."+resourceName, "disk_path", existingDisk.filepath),
							),
							Destroy: true,
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						id, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_machine."+resourceName].Primary.Attributes["id"])
						Expect(err).To(BeNil())

						_, err = freeboxClient.GetVirtualMachine(ctx, int64(id))
						Expect(err).To(MatchError(client.ErrVirtualMachineNotFound), "virtual machine %d should not exist", id)

						return nil
					},
				})
			})
		})
	})
})
