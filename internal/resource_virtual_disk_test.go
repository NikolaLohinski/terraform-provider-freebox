package internal_test

import (
	"context"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/nikolalohinski/terraform-provider-freebox/internal"
	"github.com/nikolalohinski/terraform-provider-freebox/internal/models"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context(`resource "freebox_virtual_disk" { ... }`, func() {
	type diskSpec struct {
		filepath string
		diskType string
		size     int
	}

	var (
		exampleDisk  diskSpec
		resourceName string
		config       string
	)

	const (
		originalvirtualSize = 2048_000_000 // 2MB
	)

	BeforeEach(func(ctx SpecContext) {
		resourceName = "test-" + uuid.NewString() // prefix with test- so the name start with a letter
		exampleDisk = diskSpec{
			filepath: path.Join(root, existingDisk.directory, resourceName + ".raw"),
			diskType: freeboxTypes.RawDisk,
			size:    originalvirtualSize,
		}
	})

	JustBeforeEach(func(ctx SpecContext) {
		config = providerBlock + `
			resource "freebox_virtual_disk" "` + resourceName + `" {
				path = "` + exampleDisk.filepath + `"
				type = "` + exampleDisk.diskType + `"
				virtual_size = ` + strconv.Itoa(originalvirtualSize) + `
			}
		`
	})

	Context("create and delete (CD)", func() {
		It("should create and delete the file with the defaults", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerBlock + `
							resource "freebox_virtual_disk" "` + resourceName + `" {
								path = "` + exampleDisk.filepath + `"
								type = "qcow2"
								virtual_size = ` + strconv.Itoa(originalvirtualSize) + `
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "path", exampleDisk.filepath),
							resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "type", freeboxTypes.QCow2Disk),
							resource.TestCheckResourceAttr("freebox_virtual_disk."+resourceName, "virtual_size", strconv.Itoa(originalvirtualSize)),
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
				CheckDestroy: func(s *terraform.State) error {
					_, err := freeboxClient.GetFileInfo(ctx, exampleDisk.filepath)
					Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleDisk.filepath)

					return nil
				},
			})
		})

		Context("when the file already exists", func() {
			It("should fail", func(ctx SpecContext) {
				errStr := (&client.APIError{
					Code:    "exists",
				}).Error()

				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: providerBlock + `
								resource "freebox_virtual_disk" "` + resourceName + `" {
									path = "` + existingDisk.filepath + `"
									type = "qcow2"
									virtual_size = ` + strconv.Itoa(originalvirtualSize) + `
								}
							`,
							// Replace spaces with \s+ to match fixed-size error messages
							ExpectError: regexp.MustCompile(strings.ReplaceAll(regexp.QuoteMeta(errStr), " ", `\s+`)),
						},
					},
				})
			})
		})
	})

	Context("create, update and delete (CUD)", func() {
		var newDisk diskSpec
		var newConfig string

		BeforeEach(func(ctx SpecContext) {
			newDisk = exampleDisk
		})

		JustBeforeEach(func(ctx SpecContext) {
			newConfig = providerBlock + `
				resource "freebox_virtual_disk" "` + resourceName + `" {
					path = "` + newDisk.filepath + `"
					type = "` + newDisk.diskType + `"
					virtual_size = ` + strconv.Itoa(newDisk.size) + `
				}
			`
		})

		Context("the type changes", func() {
			BeforeEach(func(ctx SpecContext) {
				newDisk.diskType = freeboxTypes.QCow2Disk

				Expect(newDisk.diskType).ToNot(Equal(exampleDisk.diskType))
			})

			Context("the size changes", func() {
				BeforeEach(func(ctx SpecContext) {
					newDisk.size += 1024_000_000 // +1MB
				})
				Context("when the path changes", func() {
					BeforeEach(func(ctx SpecContext) {
						newDisk.filepath = exampleDisk.filepath + ".new"
					})
					It("creates, recreates and deletes a file", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: config,
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
					It("creates, recreates and deletes a file", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: config,
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
						It("creates, recreates and deletes a file", func(ctx SpecContext) {
							resource.UnitTest(GinkgoT(), resource.TestCase{
								ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
								Steps: []resource.TestStep{
									{
										Config: config,
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
						It("should creates, recreates and deletes a file", func(ctx SpecContext) {
							resource.UnitTest(GinkgoT(), resource.TestCase{
								ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
								Steps: []resource.TestStep{
									{
										Config: config,
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
					It("creates, recreates and deletes a file", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: config,
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
					It("creates, recreates and deletes a file", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: config,
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
					It("creates, recreates and deletes a file", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: config,
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

	Context("import and delete (ID)", func() {
		Context("the file exists", func() {
			BeforeEach(func(ctx SpecContext) {
				taskID, err := freeboxClient.CreateVirtualDisk(ctx, freeboxTypes.VirtualDisksCreatePayload{
					DiskPath: freeboxTypes.Base64Path(exampleDisk.filepath),
					Size:     int64(exampleDisk.size),
					DiskType: exampleDisk.diskType,
				})
				Expect(err).To(BeNil())

				Expect(internal.WaitForTask(ctx, freeboxClient, models.TaskTypeVirtualDisk, taskID, nil)).To(BeEmpty())
			})

			It("should import and then delete a remote file", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: providerBlock + `
								resource "freebox_virtual_disk" "` + resourceName + `" {
									path = "` + exampleDisk.filepath + `"
								}
							`,
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
