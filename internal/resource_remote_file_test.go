package internal_test

import (
	"path"
	"regexp"
	"strings"
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
			filename:              filename,
			directory:             existingDisk.directory,
			filepath:              path.Join(root, existingDisk.directory, filename),
			digest:                "sha256:184725f66890632c7e67ec1713c50aa181c1bc60ee166c9ae13a48f1d60684b0",
			source_url_or_content: "https://raw.githubusercontent.com/NikolaLohinski/terraform-provider-freebox/refs/heads/main/examples/file-to-download.txt",
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
									source_url = "` + exampleFile.source_url_or_content + `"
									destination_path = "` + exampleFile.filepath + `"

									polling = {
										download = {
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
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
								resource.TestCheckNoResourceAttr("freebox_remote_file."+resourceName, "source_remote_file"),
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
									source_url = "` + exampleFile.source_url_or_content + `"
									destination_path = "` + exampleFile.filepath + `"
								}
							`,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
								resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.download.interval", durationEqualFunc(3*time.Second)),
								resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.download.timeout", durationEqualFunc(30*time.Minute)),
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
										source_url = "` + exampleFile.source_url_or_content + `"
										destination_path = "` + exampleFile.filepath + `"

										polling = {
											download = {
												interval = "1s"
												timeout = "1m"
											}
											delete = null
										}
									}
								`,
								Check: resource.ComposeAggregateTestCheckFunc(
									resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
									resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
									resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.download.interval", durationEqualFunc(time.Second)),
									resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.download.timeout", durationEqualFunc(time.Minute)),
									resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.copy.interval", durationEqualFunc(time.Second)),
									resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.copy.timeout", durationEqualFunc(time.Minute)),
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
									source_url = "` + exampleFile.source_url_or_content + `"
									destination_path = "` + exampleFile.filepath + `"
									checksum = "` + exampleFile.digest + `"

									polling = {
										download = {
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
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
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

		Context("from a remote file", func() {
			It("should copy, verify the checksum and delete the file", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: providerBlock + `
								resource "freebox_remote_file" "` + resourceName + `" {
									source_remote_file = "` + existingDisk.filepath + `"
									destination_path = "` + exampleFile.filepath + `"
									checksum = "` + existingDisk.digest + `"

									polling = {
										copy = {
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
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_remote_file", existingDisk.filepath),
								resource.TestCheckNoResourceAttr("freebox_remote_file."+resourceName, "source_url"),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", existingDisk.digest),
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
									source_url = "` + existingDisk.source_url_or_content + `"
									destination_path = "` + existingDisk.filepath + `"
									checksum = "` + existingDisk.digest + `"

									polling = {
										download = {
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
		var sourceAttribute string

		BeforeEach(func(ctx SpecContext) {
			newFile = exampleFile
			sourceAttribute = "source_url"
		})

		JustBeforeEach(func(ctx SpecContext) {
			config = providerBlock + `
				resource "freebox_remote_file" "` + resourceName + `" {
					source_url = "` + exampleFile.source_url_or_content + `"
					destination_path = "` + exampleFile.filepath + `"
					checksum = "` + exampleFile.digest + `"

					polling = {
						download = {
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
					` + sourceAttribute + ` = "` + newFile.source_url_or_content + `"
					destination_path = "` + newFile.filepath + `"
					checksum = "` + newFile.digest + `"

					polling = {
						download = {
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
					newFile.source_url_or_content = exampleFile.source_url_or_content + "?new=true"
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", newFile.source_url_or_content),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", newFile.source_url_or_content),
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

			Context("the source is a remote file", func() {
				BeforeEach(func(ctx SpecContext) {
					sourceAttribute = "source_remote_file"
					newFile.source_url_or_content = existingDisk.filepath
					newFile.digest = existingDisk.digest
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
										resource.TestCheckNoResourceAttr("freebox_remote_file."+resourceName, "source_remote_file"),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", existingDisk.digest),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", newFile.filepath),
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_remote_file", existingDisk.filepath),
										resource.TestCheckNoResourceAttr("freebox_remote_file."+resourceName, "source_url"),
										func(s *terraform.State) error {
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

			Context("the source is a local file", func() {
				BeforeEach(func(ctx SpecContext) {
					sourceAttribute = "source_content"
					newFile.source_url_or_content = "data"
					newFile.digest = "sha256:3a6eb0790f39ac87c94f3856b2dd2c5d110e6811602261a9a923d3bb23adc8b7" // $ echo -n data | sha256sum
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
										resource.TestCheckNoResourceAttr("freebox_remote_file."+resourceName, "source_content"),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_content", newFile.source_url_or_content),
										resource.TestCheckNoResourceAttr("freebox_remote_file."+resourceName, "source_url"),
										func(s *terraform.State) error {
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", newFile.source_url_or_content),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", newFile.source_url_or_content),
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
					newFile.source_url_or_content = exampleFile.source_url_or_content + "?new=true"
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", newFile.source_url_or_content),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", newFile.source_url_or_content),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", newFile.source_url_or_content),
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
				DownloadURLs:      []string{exampleFile.source_url_or_content},
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
									source_url = "` + exampleFile.source_url_or_content + `"
									destination_path = "` + exampleFile.filepath + `"
									checksum = "` + exampleFile.digest + `"

									polling = {
										download = {
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
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
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
	Context("create and extract (CE)", func() {
		BeforeEach(func(ctx SpecContext) {
			resourceName = "test-" + uuid.NewString() // prefix with test- so the name start with a letter
			filename := resourceName + ".raw.xz"
			exampleFile = file{
				filename:              filename,
				directory:             existingDisk.directory,
				filepath:              path.Join(root, existingDisk.directory, filename),
				digest:                "sha256:2d0a2d75cea581c8799f156e22ac5cb2fed2a88fb5cc0d0af26d46e556b1d85d",
				source_url_or_content: "https://factory.talos.dev/image/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba/v1.9.4/nocloud-arm64.raw.xz",
			}
		})

		It("should download and extract the file", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerBlock + `
							resource "freebox_remote_file" "` + resourceName + `" {
								source_url = "` + exampleFile.source_url_or_content + `"
								destination_path = "` + exampleFile.filepath + `"

								extract = {
									destination_path = "` + path.Dir(exampleFile.filepath) + `"
									overwrite = true
								}
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
							resource.TestCheckNoResourceAttr("freebox_remote_file."+resourceName, "source_remote_file"),
							resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
							resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
							func(s *terraform.State) error {
								fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
								Expect(err).To(BeNil())
								Expect(fileInfo.Name).To(Equal(exampleFile.filename))
								Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))

								extractedPath := strings.TrimSuffix(exampleFile.filepath, path.Ext(exampleFile.filepath))
								extractedInfo, err := freeboxClient.GetFileInfo(ctx, extractedPath)
								Expect(err).To(BeNil())
								Expect(extractedInfo.Type).To(BeEquivalentTo(types.FileTypeFile))

								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
					Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)

					// TODO: remove extract file at the end of the test
					// extractedPath := strings.TrimSuffix(exampleFile.filepath, path.Ext(exampleFile.filepath))
					// _, err = freeboxClient.GetFileInfo(ctx, extractedPath)
					// Expect(err).To(MatchError(client.ErrPathNotFound), "extracted directory %s should not exist", extractedPath)

					return nil
				},
			})
		})
	})
})
