package internal_test

import (
	"path"
	"regexp"
	"time"

	"github.com/nikolalohinski/free-go/client"
	"github.com/nikolalohinski/free-go/types"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Context(`resource "freebox_remote_file" { ... }`, func() {
	var (
		exampleFile  file
		resourceName string
	)

	BeforeEach(func(ctx SpecContext) {
		resourceName = "test-" + uuid.NewString() // prefix with test- so the name start with a letter
		filename := resourceName + ".txt"
		exampleFile = file{
			filename:  filename,
			directory: existingDisk.directory,
			filepath:  path.Join(root, existingDisk.directory, filename),
			digest:    "sha256:184725f66890632c7e67ec1713c50aa181c1bc60ee166c9ae13a48f1d60684b0",
			source:    "https://raw.githubusercontent.com/NikolaLohinski/terraform-provider-freebox/refs/heads/main/examples/file-to-download.txt",
		}
	})

	Context("create and delete (CD)", func() {
		Context("without a checksum", func() {
			It("should download and delete the file", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: providerBlock + `
								resource "freebox_remote_file" "` + resourceName + `" {
									source_url = "` + exampleFile.source + `"
									destination_path = "` + exampleFile.filepath + `"

									polling = {
										create = {
											interval = "1s"
											timeout = "1m"
										}
										delete = {
											interval = "1s"
											timeout = "1m"
										}
										checksum_compute = {
											interval = "1s"
											timeout = "1m"
										}
									}
								}
							`,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
								func(s *terraform.State) error {
									fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
									Expect(err).To(BeNil())
									Expect(fileInfo.Name).To(Equal(exampleFile.filename))
									Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))
									return nil
								},
							),
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
						Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)
						return nil
					},
				})
			})
		})

		Context("without a polling", func() {
			It("should download and delete the file with the defaults", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: providerBlock + `
								resource "freebox_remote_file" "` + resourceName + `" {
									source_url = "` + exampleFile.source + `"
									destination_path = "` + exampleFile.filepath + `"
								}
							`,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
								resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.create.interval", durationEqualFunc(3*time.Second)),
								resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.create.timeout", durationEqualFunc(30*time.Minute)),
								resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.delete.interval", durationEqualFunc(time.Second)),
								resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.delete.timeout", durationEqualFunc(time.Minute)),
								func(s *terraform.State) error {
									fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
									Expect(err).To(BeNil())
									Expect(fileInfo.Name).To(Equal(exampleFile.filename))
									Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))

									return nil
								},
							),
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
						Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)

						return nil
					},
				})
			})

			Context("with some polling", func() {
				It("should download and delete the file with the defaults", func(ctx SpecContext) {
					resource.UnitTest(GinkgoT(), resource.TestCase{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Steps: []resource.TestStep{
							{
								Config: providerBlock + `
									resource "freebox_remote_file" "` + resourceName + `" {
										source_url = "` + exampleFile.source + `"
										destination_path = "` + exampleFile.filepath + `"

										polling = {
											create = {
												interval = "1s"
												timeout = "1m"
											}
											delete = null
										}
									}
								`,
								Check: resource.ComposeAggregateTestCheckFunc(
									resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
									resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
									resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.create.interval", durationEqualFunc(time.Second)),
									resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.create.timeout", durationEqualFunc(time.Minute)),
									resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.delete.interval", durationEqualFunc(time.Second)),
									resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.delete.timeout", durationEqualFunc(time.Minute)),
									resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.checksum_compute.interval", durationEqualFunc(time.Second)),
									resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.checksum_compute.timeout", durationEqualFunc(2*time.Minute)),
									func(s *terraform.State) error {
										fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
										Expect(err).To(BeNil())
										Expect(fileInfo.Name).To(Equal(exampleFile.filename))
										Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))
										return nil
									},
								),
							},
						},
						CheckDestroy: func(s *terraform.State) error {
							_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
							Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)
							return nil
						},
					})
				})
			})
		})

		Context("with a checksum", func() {
			It("should download, verify the checksum and delete the file", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: providerBlock + `
								resource "freebox_remote_file" "` + resourceName + `" {
									source_url = "` + exampleFile.source + `"
									destination_path = "` + exampleFile.filepath + `"
									checksum = "` + exampleFile.digest + `"

									polling = {
										create = {
											interval = "1s"
											timeout = "1m"
										}
										delete = {
											interval = "1s"
											timeout = "1m"
										}
										checksum_compute = {
											interval = "1s"
											timeout = "1m"
										}
									}
								}
							`,

							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
								func(s *terraform.State) error {
									fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
									Expect(err).To(BeNil())
									Expect(fileInfo.Name).To(Equal(exampleFile.filename))
									Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))
									return nil
								},
							),
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
						Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)
						return nil
					},
				})
			})
		})

		Context("when the file already exists", func() {
			It("should fail", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: providerBlock + `
								resource "freebox_remote_file" "` + resourceName + `" {
									source_url = "` + existingDisk.source + `"
									destination_path = "` + existingDisk.filepath + `"
									checksum = "` + existingDisk.digest + `"

									polling = {
										create = {
											interval = "1s"
											timeout = "1m"
										}
										delete = {
											interval = "1s"
											timeout = "1m"
										}
										checksum_compute = {
											interval = "1s"
											timeout = "1m"
										}
									}
								}
							`,
							ExpectError: regexp.MustCompile(`File already exists`),
						},
					},
				})
			})
		})
	})
	Context("create, update and delete (CUD)", func() {
		var newFile file
		var config, newConfig string

		BeforeEach(func(ctx SpecContext) {
			newFile = exampleFile
		})

		JustBeforeEach(func(ctx SpecContext) {
			config = providerBlock + `
				resource "freebox_remote_file" "` + resourceName + `" {
					source_url = "` + exampleFile.source + `"
					destination_path = "` + exampleFile.filepath + `"
					checksum = "` + exampleFile.digest + `"

					polling = {
						create = {
							interval = "1s"
							timeout = "1m"
						}
						delete = {
							interval = "1s"
							timeout = "1m"
						}
						checksum_compute = {
							interval = "1s"
							timeout = "1m"
						}
					}
				}
			`
			newConfig = providerBlock + `
				resource "freebox_remote_file" "` + resourceName + `" {
					source_url = "` + newFile.source + `"
					destination_path = "` + newFile.filepath + `"
					checksum = "` + newFile.digest + `"

					polling = {
						create = {
							interval = "1s"
							timeout = "5m"
						}
						delete = {
							interval = "1s"
							timeout = "1m"
						}
						checksum_compute = {
							interval = "1s"
							timeout = "1m"
						}
					}
				}
			`
		})

		Context("the checksum changes", func() {
			BeforeEach(func(ctx SpecContext) {
				newFile.digest = "sha512:77f0161a5481d84b6efe9e6be5bf48b4c5892d37a1db69a85704c3902489fd6cb6921b6d9bf252faab0a6ea13dafc0c0392e73c346a4578949f38ac3c04c43b7" // sha512 of the same file
			})

			Context("the source changes", func() {
				BeforeEach(func(ctx SpecContext) {
					newFile.source = exampleFile.source + "?new=true"
				})
				Context("when the destination changes", func() {
					BeforeEach(func(ctx SpecContext) {
						newFile.filepath = exampleFile.filepath + ".new"
						newFile.filename = exampleFile.filename + ".new"
					})
					It("creates, recreates and deletes a file", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: config,
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
										func(s *terraform.State) error {
											fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
											Expect(err).To(BeNil())
											Expect(fileInfo.Name).To(Equal(exampleFile.filename))
											Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))
											return nil
										},
									),
								},
								{
									Config: newConfig,
									ConfigPlanChecks: resource.ConfigPlanChecks{
										PreApply: []plancheck.PlanCheck{
											plancheck.ExpectResourceAction("freebox_remote_file."+resourceName, plancheck.ResourceActionReplace),
										},
									},
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", newFile.digest),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", newFile.filepath),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", newFile.source),
										func(s *terraform.State) error {
											_, err := freeboxClient.GetFileInfo(ctx, newFile.filepath)
											Expect(err).To(BeNil(), "file %s should exist", newFile.filepath)

											_, err = freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
											Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)

											return nil
										},
									),
								},
							},
							CheckDestroy: func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)

								_, err = freeboxClient.GetFileInfo(ctx, newFile.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", newFile.filepath)

								return nil
							},
						})
					})
				})
				Context("the destination remains unchanged", func() {
					It("creates, recreates and deletes a file", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: config,
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
										func(s *terraform.State) error {
											fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
											Expect(err).To(BeNil())
											Expect(fileInfo.Name).To(Equal(exampleFile.filename))
											Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))

											return nil
										},
									),
								},
								{
									Config:             newConfig,
									ExpectNonEmptyPlan: false,
									ConfigPlanChecks: resource.ConfigPlanChecks{
										PreApply: []plancheck.PlanCheck{
											plancheck.ExpectResourceAction("freebox_remote_file."+resourceName, plancheck.ResourceActionReplace),
										},
									},
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", newFile.digest),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", newFile.filepath),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", newFile.source),
									),
								},
							},
							CheckDestroy: func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)

								_, err = freeboxClient.GetFileInfo(ctx, newFile.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", newFile.filepath)

								return nil
							},
						})
					})
				})
			})

			Context("the source remains unchanged", func() {
				Context("the destination changes", func() {
					BeforeEach(func(ctx SpecContext) {
						newFile.filepath = exampleFile.filepath + ".new"
						newFile.filename = exampleFile.filename + ".new"
					})
					It("creates, recreates and deletes a file", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: config,
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
										func(s *terraform.State) error {
											fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
											Expect(err).To(BeNil())
											Expect(fileInfo.Name).To(Equal(exampleFile.filename))
											Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))

											return nil
										},
									),
								},
								{
									Config: newConfig,
									ConfigPlanChecks: resource.ConfigPlanChecks{
										PreApply: []plancheck.PlanCheck{
											plancheck.ExpectResourceAction("freebox_remote_file."+resourceName, plancheck.ResourceActionReplace),
										},
									},
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", newFile.digest),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", newFile.filepath),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", newFile.source),
										func(s *terraform.State) error {
											_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
											Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should no more exist", exampleFile.filepath)

											fileInfo, err := freeboxClient.GetFileInfo(ctx, newFile.filepath)
											Expect(err).To(BeNil())
											Expect(fileInfo.Name).To(Equal(newFile.filename))
											Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))

											return nil
										},
									),
								},
							},
							CheckDestroy: func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)

								_, err = freeboxClient.GetFileInfo(ctx, newFile.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", newFile.filepath)

								return nil
							},
						})
					})
				})
				Context("the destination remains unchanged", func() {
					It("should creates, recreates and deletes a file", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: config,
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
										func(s *terraform.State) error {
											fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
											Expect(err).To(BeNil())
											Expect(fileInfo.Name).To(Equal(exampleFile.filename))
											Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))
											return nil
										},
									),
								},
								{
									Config: newConfig,
									ConfigPlanChecks: resource.ConfigPlanChecks{
										PreApply: []plancheck.PlanCheck{
											plancheck.ExpectResourceAction("freebox_remote_file."+resourceName, plancheck.ResourceActionReplace),
										},
									},
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", newFile.digest),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", newFile.filepath),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", newFile.source),
										func(s *terraform.State) error {
											_, err := freeboxClient.GetFileInfo(ctx, newFile.filepath)
											Expect(err).To(BeNil(), "file %s should exist", newFile.filepath)

											return nil
										},
									),
								},
							},
							CheckDestroy: func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)

								_, err = freeboxClient.GetFileInfo(ctx, newFile.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", newFile.filepath)

								return nil
							},
						})
					})
				})
			})
		})
		Context("the checksum remains unchanged", func() {
			Context("the source changes", func() {
				BeforeEach(func(ctx SpecContext) {
					newFile.source = exampleFile.source + "?new=true"
				})
				Context("the destination changes", func() {
					BeforeEach(func(ctx SpecContext) {
						newFile.filepath = exampleFile.filepath + ".new"
						newFile.filename = exampleFile.filename + ".new"
					})
					It("creates, moves and deletes a file", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: config,
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
										func(s *terraform.State) error {
											fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
											Expect(err).To(BeNil())
											Expect(fileInfo.Name).To(Equal(exampleFile.filename))
											Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))
											return nil
										},
									),
								},
								{
									Config: newConfig,
									ConfigPlanChecks: resource.ConfigPlanChecks{
										PreApply: []plancheck.PlanCheck{
											plancheck.ExpectResourceAction("freebox_remote_file."+resourceName, plancheck.ResourceActionUpdate),
										},
									},
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", newFile.digest),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", newFile.filepath),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", newFile.source),
										func(s *terraform.State) error {
											_, err := freeboxClient.GetFileInfo(ctx, newFile.filepath)
											Expect(err).To(BeNil(), "file %s should exist", newFile.filepath)

											_, err = freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
											Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)

											return nil
										},
									),
								},
							},
							CheckDestroy: func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)

								_, err = freeboxClient.GetFileInfo(ctx, newFile.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", newFile.filepath)

								return nil
							},
						})
					})
				})
				Context("the destination remains unchanged", func() {
					It("creates, verify checksum and deletes a file", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: config,
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
										func(s *terraform.State) error {
											fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
											Expect(err).To(BeNil())
											Expect(fileInfo.Name).To(Equal(exampleFile.filename))
											Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))
											return nil
										},
									),
								},
								{
									Config:             newConfig,
									ExpectNonEmptyPlan: false,
									ConfigPlanChecks: resource.ConfigPlanChecks{
										PreApply: []plancheck.PlanCheck{
											plancheck.ExpectResourceAction("freebox_remote_file."+resourceName, plancheck.ResourceActionUpdate),
										},
									},
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", newFile.digest),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", newFile.filepath),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", newFile.source),
										func(s *terraform.State) error {
											_, err := freeboxClient.GetFileInfo(ctx, newFile.filepath)
											Expect(err).To(BeNil(), "file %s should exist", newFile.filepath)

											return nil
										},
									),
								},
							},
							CheckDestroy: func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)

								_, err = freeboxClient.GetFileInfo(ctx, newFile.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", newFile.filepath)

								return nil
							},
						})
					})
				})
			})

			Context("when the source remain unchanged", func() {
				Context("when the destination change", func() {
					BeforeEach(func(ctx SpecContext) {
						newFile.filepath = exampleFile.filepath + ".new"
						newFile.filename = exampleFile.filename + ".new"
					})
					It("creates, moves and delete a file", func(ctx SpecContext) {
						resource.UnitTest(GinkgoT(), resource.TestCase{
							ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
							Steps: []resource.TestStep{
								{
									Config: config,
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
										func(s *terraform.State) error {
											fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
											Expect(err).To(BeNil())
											Expect(fileInfo.Name).To(Equal(exampleFile.filename))
											Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))

											return nil
										},
									),
								},
								{
									Config: newConfig,
									ConfigPlanChecks: resource.ConfigPlanChecks{
										PreApply: []plancheck.PlanCheck{
											plancheck.ExpectResourceAction("freebox_remote_file."+resourceName, plancheck.ResourceActionUpdate),
										},
									},
									Check: resource.ComposeAggregateTestCheckFunc(
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", newFile.digest),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", newFile.filepath),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", newFile.source),
										func(s *terraform.State) error {
											_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
											Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should no more exist", exampleFile.filepath)

											fileInfo, err := freeboxClient.GetFileInfo(ctx, newFile.filepath)
											Expect(err).To(BeNil())
											Expect(fileInfo.Name).To(Equal(newFile.filename))
											Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))

											return nil
										},
									),
								},
							},
							CheckDestroy: func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)

								_, err = freeboxClient.GetFileInfo(ctx, newFile.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", newFile.filepath)

								return nil
							},
						})
					})
				})
			})
		})
	})
	Context("import and delete (ID)", func() {
		BeforeEach(func(ctx SpecContext) {
			taskID, err := freeboxClient.AddDownloadTask(ctx, types.DownloadRequest{
				DownloadDirectory: path.Join(root, exampleFile.directory),
				DownloadURLs:      []string{exampleFile.source},
				Filename:          exampleFile.filename,
				Hash:              exampleFile.digest,
			})
			Expect(err).To(BeNil())
			Expect(taskID).ToNot(BeZero())

			DeferCleanup(func(ctx SpecContext, taskID int64) {
				Expect(freeboxClient.DeleteDownloadTask(ctx, taskID)).To(Succeed())
			}, taskID)

			Eventually(func() types.DownloadTask {
				downloadTask, err := freeboxClient.GetDownloadTask(ctx, taskID)
				Expect(err).To(BeNil())
				return downloadTask
			}, "30s").Should(MatchFields(IgnoreExtras, Fields{
				"Status": BeEquivalentTo(types.DownloadTaskStatusDone),
			}))
		})

		Describe("import and delete with path", func() {
			It("should import and then delete a remote file", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: providerBlock + `
								resource "freebox_remote_file" "` + resourceName + `" {
									source_url = "` + exampleFile.source + `"
									destination_path = "` + exampleFile.filepath + `"
									checksum = "` + exampleFile.digest + `"

									polling = {
										create = {
											interval = "1s"
											timeout = "1m"
										}
										delete = {
											interval = "1s"
											timeout = "1m"
										}
										checksum_compute = {
											interval = "1s"
											timeout = "1m"
										}
									}
								}
							`,
							ResourceName:       "freebox_remote_file." + resourceName,
							ImportState:        true,
							ImportStateId:      exampleFile.filepath,
							ImportStatePersist: true,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
							),
							Destroy: true,
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
						Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)
						return nil
					},
				})
			})
		})
	})
})
