package internal_test

import (
	"context"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/nikolalohinski/terraform-provider-freebox/internal"
	"github.com/nikolalohinski/terraform-provider-freebox/internal/models"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
)

var _ = Context(`resource "freebox_virtual_disk" { ... }`, func() {
	type diskSpec struct {
		filepath string
		diskType string
		size     int
	}

	var (
		exampleDisk         diskSpec
		resourceName        string
		initialConfig       string
		originalvirtualSize int
	)

	BeforeEach(func(ctx SpecContext) {
		resourceName = "test-" + uuid.NewString() // prefix with test- so the name start with a letter

		originalvirtualSize = roundVirtualSize(randGenerator.Intn(2_000_000) + 1_000_000) // 1MB to 3MB
		exampleDisk = diskSpec{
			filepath: path.Join(root, existingDisk.directory, resourceName+".qcow2"),
			diskType: freeboxTypes.QCow2Disk,
			size:     originalvirtualSize,
		}
	})

	JustBeforeEach(func(ctx SpecContext) {
		initialConfig = providerBlock + `
			resource "freebox_virtual_disk" "` + resourceName + `" {
				path = "` + exampleDisk.filepath + `"
				type = "` + exampleDisk.diskType + `"
				virtual_size = ` + strconv.Itoa(originalvirtualSize) + `
				resize_from = null
			}
		`
	})

	Context("create and delete", func() {
		It("should create and delete the file with the defaults", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: initialConfig,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", exampleDisk.filepath),
							resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", exampleDisk.diskType),
							resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(originalvirtualSize)),
							func(s *terraform.State) error {
								path := s.RootModule().Resources["freebox_virtual_disk."+resourceName].Primary.Attributes["path"]
								diskInfo, err := freeboxClient.GetVirtualDiskInfo(ctx, path)
								Expect(err).To(BeNil())
								Expect(diskInfo.Type).To(Equal(exampleDisk.diskType))
								Expect(diskInfo.VirtualSize).To(Equal(int64(originalvirtualSize)))
								Expect(diskInfo.ActualSize).To(BeNumerically(">", 0))

								sizeOnDisk, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_disk."+resourceName].Primary.Attributes["size_on_disk"])
								Expect(err).To(BeNil())
								Expect(sizeOnDisk).To(BeEquivalentTo(diskInfo.ActualSize))

								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					_, err := freeboxClient.GetFileInfo(ctx, exampleDisk.filepath)
					Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleDisk.filepath)

					return nil
				},
			})
		})

		Context("when the file already exists", func() {
			BeforeEach(func(ctx SpecContext) {
				task, err := freeboxClient.CopyFiles(ctx, []string{existingDisk.filepath}, exampleDisk.filepath, freeboxTypes.FileCopyModeSkip)
				Expect(err).To(BeNil())
				Expect(task.ID).ToNot(BeZero())

				DeferCleanup(func(ctx SpecContext) {
					Expect(freeboxClient.DeleteFileSystemTask(ctx, task.ID)).To(Succeed())
				})

				Eventually(func() freeboxTypes.FileSystemTask {
					task, err := freeboxClient.GetFileSystemTask(ctx, task.ID)
					Expect(err).To(BeNil())
					return task
				}, "1m").Should(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"State": BeEquivalentTo(freeboxTypes.FileTaskStateDone),
				}))
			})

			It("should fail", func(ctx SpecContext) {
				errStr := (&client.APIError{
					Code: string(freeboxTypes.DiskTaskErrorExists),
				}).Error()

				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
							// Replace spaces with \s+ to match fixed-size error messages
							ExpectError: regexp.MustCompile(strings.ReplaceAll(regexp.QuoteMeta(errStr), " ", `\s+`)),
						},
					},
				})
			})
		})

		Context("when the resize_from is specified", func() {
			JustBeforeEach(func(ctx SpecContext) {
				initialConfig = terraformConfigWithAttribute("resize_from", existingDisk.filepath)(initialConfig)
				initialConfig = terraformConfigWithoutAttribute("type")(initialConfig)
			})

			It("should create the disk from the existing disk", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", exampleDisk.filepath),
								func(s *terraform.State) error {
									path := s.RootModule().Resources["freebox_virtual_disk."+resourceName].Primary.Attributes["path"]
									diskInfo, err := freeboxClient.GetVirtualDiskInfo(ctx, path)
									Expect(err).To(BeNil())
									Expect(diskInfo.Type).To(Equal(freeboxTypes.QCow2Disk))
									Expect(diskInfo.VirtualSize).To(Equal(int64(originalvirtualSize)))
									Expect(diskInfo.ActualSize).To(BeNumerically(">", 0))

									sizeOnDisk, err := strconv.Atoi(s.RootModule().Resources["freebox_virtual_disk."+resourceName].Primary.Attributes["size_on_disk"])
									Expect(err).To(BeNil())
									Expect(sizeOnDisk).To(BeEquivalentTo(diskInfo.ActualSize))

									return nil
								},
							),
						},
					},
				})
			})
		})
	})

	Context("create, update and delete", func() {
		var newDisk diskSpec
		var newConfig string

		BeforeEach(func(ctx SpecContext) {
			newDisk = exampleDisk
		})

		JustBeforeEach(func(ctx SpecContext) {
			newConfig = initialConfig

			if newDisk.filepath != exampleDisk.filepath {
				newConfig = terraformConfigWithAttribute("path", newDisk.filepath)(newConfig)
			}
			if newDisk.diskType != exampleDisk.diskType {
				newConfig = terraformConfigWithAttribute("type", newDisk.diskType)(newConfig)
			}
			if newDisk.size != exampleDisk.size {
				newConfig = terraformConfigWithAttribute("virtual_size", strconv.Itoa(newDisk.size))(newConfig)
			}
		})

		Context("the type changes", func() {
			BeforeEach(func(ctx SpecContext) {
				newDisk.diskType = freeboxTypes.RawDisk

				Expect(newDisk.diskType).ToNot(Equal(exampleDisk.diskType))
			})

			Context("the size changes", func() {
				BeforeEach(func(ctx SpecContext) {
					newDisk.size = roundVirtualSize(newDisk.size + randGenerator.Intn(1_000_000) + 1_000_000) // +1MB to 2MB
				})
				Context("when the path changes", func() {
					BeforeEach(func(ctx SpecContext) {
						newDisk.filepath = exampleDisk.filepath + ".new"
					})
					It("recreates the disk", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: initialConfig,
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", exampleDisk.filepath),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", exampleDisk.diskType),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(exampleDisk.size)),
										checkDiskStateFunc(ctx, resourceName, exampleDisk.filepath),
									),
								},
								{
									Config: newConfig,
									ConfigPlanChecks: resource.ConfigPlanChecks{
										PreApply: []plancheck.PlanCheck{
											plancheck.ExpectResourceAction("freebox_virtual_disk."+resourceName, plancheck.ResourceActionReplace),
										},
									},
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", newDisk.filepath),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", newDisk.diskType),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(newDisk.size)),
										checkDiskStateFunc(ctx, resourceName, newDisk.filepath),
									),
								},
							},
							CheckDestroy: func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleDisk.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleDisk.filepath)

								_, err = freeboxClient.GetFileInfo(ctx, newDisk.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", newDisk.filepath)

								return nil
							},
						})
					})
				})
				Context("the path remains unchanged", func() {
					It("recreates the disk", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: initialConfig,
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", exampleDisk.filepath),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", exampleDisk.diskType),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(exampleDisk.size)),
										checkDiskStateFunc(ctx, resourceName, exampleDisk.filepath),
									),
								},
								{
									Config:             newConfig,
									ExpectNonEmptyPlan: false,
									ConfigPlanChecks: resource.ConfigPlanChecks{
										PreApply: []plancheck.PlanCheck{
											plancheck.ExpectResourceAction("freebox_virtual_disk."+resourceName, plancheck.ResourceActionReplace),
										},
									},
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", newDisk.filepath),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", newDisk.diskType),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(newDisk.size)),
										checkDiskStateFunc(ctx, resourceName, newDisk.filepath),
									),
								},
							},
							CheckDestroy: func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleDisk.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleDisk.filepath)

								_, err = freeboxClient.GetFileInfo(ctx, newDisk.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", newDisk.filepath)

								return nil
							},
						})
					})
				})
			})

			Context("the size remains unchanged", func() {
				Context("the path changes", func() {
					BeforeEach(func(ctx SpecContext) {
						newDisk.filepath = exampleDisk.filepath + ".new"
					})
					It("recreates the disk", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: initialConfig,
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", exampleDisk.filepath),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", exampleDisk.diskType),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(exampleDisk.size)),
										checkDiskStateFunc(ctx, resourceName, exampleDisk.filepath),
									),
								},
								{
									Config: newConfig,
									ConfigPlanChecks: resource.ConfigPlanChecks{
										PreApply: []plancheck.PlanCheck{
											plancheck.ExpectResourceAction("freebox_virtual_disk."+resourceName, plancheck.ResourceActionReplace),
										},
									},
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", newDisk.filepath),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", newDisk.diskType),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(newDisk.size)),
										checkDiskStateFunc(ctx, resourceName, newDisk.filepath),
									),
								},
							},
							CheckDestroy: func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleDisk.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleDisk.filepath)

								_, err = freeboxClient.GetFileInfo(ctx, newDisk.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", newDisk.filepath)

								return nil
							},
						})
					})
				})
				Context("the path remains unchanged", func() {
					It("recreates the disk", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: initialConfig,
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", exampleDisk.filepath),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", exampleDisk.diskType),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(exampleDisk.size)),
										checkDiskStateFunc(ctx, resourceName, exampleDisk.filepath),
									),
								},
								{
									Config: newConfig,
									ConfigPlanChecks: resource.ConfigPlanChecks{
										PreApply: []plancheck.PlanCheck{
											plancheck.ExpectResourceAction("freebox_virtual_disk."+resourceName, plancheck.ResourceActionReplace),
										},
									},
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", newDisk.filepath),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", newDisk.diskType),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(newDisk.size)),
										checkDiskStateFunc(ctx, resourceName, newDisk.filepath),
									),
								},
							},
							CheckDestroy: func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleDisk.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleDisk.filepath)

								_, err = freeboxClient.GetFileInfo(ctx, newDisk.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", newDisk.filepath)

								return nil
							},
						})
					})
				})
			})
		})

		Context("the type remains unchanged", func() {
			Context("the size changes", func() {
				BeforeEach(func(ctx SpecContext) {
					newDisk.size += 1024_000_000 // +1MB
				})
				Context("when the path changes", func() {
					BeforeEach(func(ctx SpecContext) {
						newDisk.filepath = exampleDisk.filepath + ".new"
					})

					It("move and resize the disk", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: initialConfig,
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", exampleDisk.filepath),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", exampleDisk.diskType),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(exampleDisk.size)),
										checkDiskStateFunc(ctx, resourceName, exampleDisk.filepath),
									),
								},
								{
									Config: newConfig,
									ConfigPlanChecks: resource.ConfigPlanChecks{
										PreApply: []plancheck.PlanCheck{
											plancheck.ExpectResourceAction("freebox_virtual_disk."+resourceName, plancheck.ResourceActionUpdate),
										},
									},
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", newDisk.filepath),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", newDisk.diskType),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(newDisk.size)),
										checkDiskStateFunc(ctx, resourceName, newDisk.filepath),
									),
								},
							},
							CheckDestroy: func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleDisk.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleDisk.filepath)

								_, err = freeboxClient.GetFileInfo(ctx, newDisk.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", newDisk.filepath)

								return nil
							},
						})
					})
				})
				Context("the path remains unchanged", func() {
					It("resize the disk", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: initialConfig,
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", exampleDisk.filepath),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", exampleDisk.diskType),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(exampleDisk.size)),
										checkDiskStateFunc(ctx, resourceName, exampleDisk.filepath),
									),
								},
								{
									Config:             newConfig,
									ExpectNonEmptyPlan: false,
									ConfigPlanChecks: resource.ConfigPlanChecks{
										PreApply: []plancheck.PlanCheck{
											plancheck.ExpectResourceAction("freebox_virtual_disk."+resourceName, plancheck.ResourceActionUpdate),
										},
									},
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", newDisk.filepath),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", newDisk.diskType),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(newDisk.size)),
										checkDiskStateFunc(ctx, resourceName, newDisk.filepath),
									),
								},
							},
							CheckDestroy: func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleDisk.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleDisk.filepath)

								_, err = freeboxClient.GetFileInfo(ctx, newDisk.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", newDisk.filepath)

								return nil
							},
						})
					})
				})
			})

			Context("the size remains unchanged", func() {
				Context("the path changes", func() {
					BeforeEach(func(ctx SpecContext) {
						newDisk.filepath = exampleDisk.filepath + ".new"
					})
					It("move the disk", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: initialConfig,
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", exampleDisk.filepath),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", exampleDisk.diskType),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(exampleDisk.size)),
										checkDiskStateFunc(ctx, resourceName, exampleDisk.filepath),
									),
								},
								{
									Config: newConfig,
									ConfigPlanChecks: resource.ConfigPlanChecks{
										PreApply: []plancheck.PlanCheck{
											plancheck.ExpectResourceAction("freebox_virtual_disk."+resourceName, plancheck.ResourceActionUpdate),
										},
									},
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", newDisk.filepath),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", newDisk.diskType),
										resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(newDisk.size)),
										checkDiskStateFunc(ctx, resourceName, newDisk.filepath),
									),
								},
							},
							CheckDestroy: func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleDisk.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleDisk.filepath)

								_, err = freeboxClient.GetFileInfo(ctx, newDisk.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", newDisk.filepath)

								return nil
							},
						})
					})
				})
				Context("the path remains unchanged", func() {
					// It should not do anything
				})
			})
		})
	})

	Context("import and delete", func() {
		Context("the file exists", func() {
			BeforeEach(func(ctx SpecContext) {
				taskID, err := freeboxClient.CreateVirtualDisk(ctx, freeboxTypes.VirtualDisksCreatePayload{
					DiskPath: freeboxTypes.Base64Path(exampleDisk.filepath),
					Size:     int64(exampleDisk.size),
					DiskType: exampleDisk.diskType,
				})
				Expect(err).To(BeNil())
				second := time.Second
				minute := time.Minute
				Expect(internal.WaitForTask(ctx, freeboxClient, models.TaskTypeVirtualDisk, taskID, &models.Polling{
					Interval: timetypes.NewGoDurationPointerValue(&second),
					Timeout:  timetypes.NewGoDurationPointerValue(&minute),
				})).To(BeEmpty())
			})

			It("should work", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config:             initialConfig,
							ResourceName:       "freebox_virtual_disk." + resourceName,
							ImportState:        true,
							ImportStateId:      exampleDisk.filepath,
							ImportStatePersist: true,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", exampleDisk.filepath),
								resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", freeboxTypes.QCow2Disk),
								resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(originalvirtualSize)),
							),
							Destroy: true,
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						_, err := freeboxClient.GetFileInfo(ctx, exampleDisk.filepath)
						Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleDisk.filepath)

						return nil
					},
				})
			})
		})
	})
})

func checkDiskStateFunc(ctx context.Context, resourceName, path string) func(s *terraform.State) error {
	return func(s *terraform.State) error {
		diskInfo, err := freeboxClient.GetVirtualDiskInfo(ctx, path)
		Expect(err).To(BeNil())

		state := s.RootModule().Resources["freebox_virtual_disk."+resourceName].Primary.Attributes

		diskType := state["type"]
		Expect(diskInfo.Type).To(BeEquivalentTo(diskType))

		virtualSize, err := strconv.Atoi(state["virtual_size"])
		Expect(err).To(BeNil())
		Expect(diskInfo.VirtualSize).To(BeEquivalentTo(virtualSize))

		sizeOnDisk, err := strconv.Atoi(state["size_on_disk"])
		Expect(err).To(BeNil())
		Expect(diskInfo.ActualSize).To(BeEquivalentTo(sizeOnDisk))

		return nil
	}
}

func roundVirtualSize(size int) int {
	mod := size % 8_192
	if mod == 0 {
		return size
	}
	return size + 8_192 - mod
}
