package internal_test

import (
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/nikolalohinski/free-go/client"
	"github.com/nikolalohinski/free-go/types"
	freeboxTypes "github.com/nikolalohinski/free-go/types"

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

		initialConfig string
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

	JustBeforeEach(func(ctx SpecContext) {
		initialConfig = providerBlock + `
			resource "freebox_remote_file" "` + resourceName + `" {
				source_url = "` + exampleFile.source_url_or_content + `"
				source_content = null
				source_remote_file = null
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

				extract = null
			}
		`
	})

	Context("create and delete", func() {
		Context("without a checksum", func() {
			It("should download and delete the file", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
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

		Context("with no checksum", func() {
			It("should download, verify the checksum and delete the file", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: terraformConfigWithoutAttribute("checksum")(initialConfig),
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

		Context("with an invalid checksum", func() {
			AfterEach(func(ctx SpecContext) {
				// cleanup the download task if it exists
				tasks, err := freeboxClient.ListDownloadTasks(ctx)
				Expect(err).To(BeNil())

				for _, task := range tasks {
					if task.Name == exampleFile.filename && task.Status == freeboxTypes.DownloadTaskStatusError && task.Error == freeboxTypes.DownloadTaskErrorBadHash {
						Expect(freeboxClient.DeleteDownloadTask(ctx, task.ID)).To(Succeed())
					}
				}
			})

			It("should download, verify the checksum and delete the file", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config:      terraformConfigWithAttribute("checksum", "sha256:"+strings.Repeat("a", 64))(initialConfig),
							ExpectError: regexp.MustCompile(regexp.QuoteMeta(string(freeboxTypes.DownloadTaskErrorBadHash))),
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
							Config: terraformConfigWithoutAttribute("source_url")(terraformConfigWithAttribute("source_remote_file", existingDisk.filepath)(terraformConfigWithAttribute("checksum", existingDisk.digest)(initialConfig))),
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
							Config:      terraformConfigWithAttribute("destination_path", existingDisk.filepath)(initialConfig),
							ExpectError: regexp.MustCompile(`File already exists`),
						},
					},
				})
			})
		})
	})
	Context("create, update and delete", func() {
		var newFile file
		var newConfig string
		var sourceAttribute string

		BeforeEach(func(ctx SpecContext) {
			newFile = exampleFile
			sourceAttribute = "source_url"
		})

		JustBeforeEach(func(ctx SpecContext) {
			newConfig = terraformConfigWithAttribute("source_url", nil)(initialConfig)
			newConfig = terraformConfigWithAttribute(sourceAttribute, newFile.source_url_or_content)(newConfig)

			if newFile.filepath != exampleFile.filepath {
				newConfig = terraformConfigWithAttribute("destination_path", newFile.filepath)(newConfig)
			}
			if newFile.digest != exampleFile.digest {
				newConfig = terraformConfigWithAttribute("checksum", newFile.digest)(newConfig)
			}
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
									Config: initialConfig,
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
									Config: initialConfig,
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
									Config: initialConfig,
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
									Config: initialConfig,
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
									Config: initialConfig,
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
									Config: initialConfig,
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
									Config: initialConfig,
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
									Config: initialConfig,
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
									Config: initialConfig,
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
	Context("import and delete", func() {
		Context("the file exists", func() {
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

			Describe("from destination_path", func() {
				It("should work", func(ctx SpecContext) {
					resource.UnitTest(GinkgoT(), resource.TestCase{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Steps: []resource.TestStep{
							{
								Config:             initialConfig,
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
	})
	Context("create and extract", func() {
		var (
			destinationPath string
			extractedPath   string
		)

		Context("from an online xz file", func() {
			BeforeEach(func(ctx SpecContext) {
				filename := resourceName + ".raw.xz"
				exampleFile = file{
					filename:              filename,
					directory:             existingDisk.directory,
					filepath:              path.Join(root, existingDisk.directory, filename),
					digest:                "sha256:19b07d31088b3beea83948506201d1648a31d859d177a8c5ebedfdfc44155a29",
					source_url_or_content: "https://factory.talos.dev/image/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba/v1.9.4/nocloud-arm64.raw.xz",
				}
				destinationPath = path.Join(root, existingDisk.directory)
				extractedPath = path.Join(destinationPath, strings.TrimSuffix(path.Base(exampleFile.filepath), path.Ext(exampleFile.filepath)))
			})

			JustBeforeEach(func(ctx SpecContext) {
				initialConfig = terraformConfigWithAttribute("extract", []byte(`{
					destination_path = "`+destinationPath+`"
					overwrite = true
				}`))(initialConfig)
			})

			It("should download and extract the file", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source_url_or_content),
								resource.TestCheckNoResourceAttr("freebox_remote_file."+resourceName, "source_remote_file"),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
								func(s *terraform.State) error {
									fileInfo, err := freeboxClient.GetFileInfo(ctx, destinationPath)
									Expect(err).To(BeNil())
									Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeDirectory))

									_, err = freeboxClient.GetFileInfo(ctx, extractedPath)
									Expect(err).To(BeNil())

									return nil
								},
							),
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
						Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)

						// TODO: remove extract file at the end of the test
						// _, err = freeboxClient.GetFileInfo(ctx, extractedPath)
						// Expect(err).To(MatchError(client.ErrPathNotFound), "extracted file %a should not exist", extractedPath)

						return nil
					},
				})
			})
		})
	})
})
