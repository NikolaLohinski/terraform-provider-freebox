package internal_test

import (
	"errors"
	"path"
	go_path "path"
	"regexp"
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
	"github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Context(`resource "freebox_remote_file" { source_bytes = ... }`, func() {
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
			digest:                "sha256:3a6eb0790f39ac87c94f3856b2dd2c5d110e6811602261a9a923d3bb23adc8b7", // $ echo -n data | sha256sum
			source_url_or_content: `data`,
		}
	})

	Context("create and delete (CD)", func() {
		Context("without a checksum", func() {
			It("should upload and delete the file", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: providerBlock + `
								resource "freebox_remote_file" "` + resourceName + `" {
									source_bytes = "` + exampleFile.source_url_or_content + `"
									destination_path = "` + exampleFile.filepath + `"

									polling = {
										upload = {
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
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_bytes", exampleFile.source_url_or_content),
								resource.TestCheckNoResourceAttr("freebox_remote_file."+resourceName, "source_remote_file"),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
								func(s *terraform.State) error {
									fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
									Expect(err).To(BeNil())
									Expect(fileInfo.Name).To(Equal(exampleFile.filename))
									Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))

									file, err := freeboxClient.GetFile(ctx, exampleFile.filepath)
									Expect(err).To(BeNil())
									p := make([]byte, len(exampleFile.source_url_or_content))
									Expect(gbytes.TimeoutReader(file.Content, time.Second).Read(p)).To(Equal(len(exampleFile.source_url_or_content)))
									Expect(p).To(BeEquivalentTo(exampleFile.source_url_or_content))

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
			It("should upload and delete the file with the defaults", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: providerBlock + `
								resource "freebox_remote_file" "` + resourceName + `" {
									source_bytes = "` + exampleFile.source_url_or_content + `"
									destination_path = "` + exampleFile.filepath + `"
								}
							`,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_bytes", exampleFile.source_url_or_content),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
								resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.download.interval", durationEqualFunc(3*time.Second)),
								resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.download.timeout", durationEqualFunc(30*time.Minute)),
								resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.upload.interval", durationEqualFunc(3*time.Second)),
								resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.upload.timeout", durationEqualFunc(30*time.Minute)),
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

									file, err := freeboxClient.GetFile(ctx, exampleFile.filepath)
									Expect(err).To(BeNil())
									p := make([]byte, len(exampleFile.source_url_or_content))
									Expect(gbytes.TimeoutReader(file.Content, time.Second).Read(p)).To(Equal(len(exampleFile.source_url_or_content)))
									Expect(p).To(BeEquivalentTo(exampleFile.source_url_or_content))

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
				It("should upload and delete the file with the defaults", func(ctx SpecContext) {
					resource.UnitTest(GinkgoT(), resource.TestCase{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Steps: []resource.TestStep{
							{
								Config: providerBlock + `
									resource "freebox_remote_file" "` + resourceName + `" {
										source_bytes = "` + exampleFile.source_url_or_content + `"
										destination_path = "` + exampleFile.filepath + `"

										polling = {
											upload = {
												interval = "1s"
												timeout = "1m"
											}
											delete = null
										}
									}
								`,
								Check: resource.ComposeAggregateTestCheckFunc(
									resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_bytes", exampleFile.source_url_or_content),
									resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
									resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.download.interval", durationEqualFunc(3*time.Second)),
									resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.download.timeout", durationEqualFunc(30*time.Minute)),
									resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.upload.interval", durationEqualFunc(time.Second)),
									resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "polling.upload.timeout", durationEqualFunc(time.Minute)),
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

										file, err := freeboxClient.GetFile(ctx, exampleFile.filepath)
										Expect(err).To(BeNil())
										p := make([]byte, len(exampleFile.source_url_or_content))
										Expect(gbytes.TimeoutReader(file.Content, time.Second).Read(p)).To(Equal(len(exampleFile.source_url_or_content)))
										Expect(p).To(BeEquivalentTo(exampleFile.source_url_or_content))

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
			It("should upload, verify the checksum and delete the file", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: providerBlock + `
								resource "freebox_remote_file" "` + resourceName + `" {
									source_bytes = "` + exampleFile.source_url_or_content + `"
									destination_path = "` + exampleFile.filepath + `"
									checksum = "` + exampleFile.digest + `"

									polling = {
										upload = {
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
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_bytes", exampleFile.source_url_or_content),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
								func(s *terraform.State) error {
									fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
									Expect(err).To(BeNil())
									Expect(fileInfo.Name).To(Equal(exampleFile.filename))
									Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))

									file, err := freeboxClient.GetFile(ctx, exampleFile.filepath)
									Expect(err).To(BeNil())
									p := make([]byte, len(exampleFile.source_url_or_content))
									Expect(gbytes.TimeoutReader(file.Content, time.Second).Read(p)).To(Equal(len(exampleFile.source_url_or_content)))
									Expect(p).To(BeEquivalentTo(exampleFile.source_url_or_content))

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
									source_bytes = "` + existingDisk.source_url_or_content + `"
									destination_path = "` + existingDisk.filepath + `"
									checksum = "` + existingDisk.digest + `"

									polling = {
										upload = {
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
			sourceAttribute = "source_bytes"
		})

		JustBeforeEach(func(ctx SpecContext) {
			config = providerBlock + `
				resource "freebox_remote_file" "` + resourceName + `" {
					source_bytes = "` + exampleFile.source_url_or_content + `"
					destination_path = "` + exampleFile.filepath + `"
					checksum = "` + exampleFile.digest + `"

					polling = {
						upload = {
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
						upload = {
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
				newFile.digest = "sha256:28369943a1e0316edf2c8d7fb01cc70899b7234c7d5f8940c75fae7693d3f757" // $ echo -n data-new | sha256sum
			})

			Context("the source changes", func() {
				BeforeEach(func(ctx SpecContext) {
					newFile.source_url_or_content = exampleFile.source_url_or_content + "-new"
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_bytes", exampleFile.source_url_or_content),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_bytes", newFile.source_url_or_content),
										func(s *terraform.State) error {
											_, err := freeboxClient.GetFileInfo(ctx, newFile.filepath)
											Expect(err).To(BeNil(), "file %s should exist", newFile.filepath)

											_, err = freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
											Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)

											file, err := freeboxClient.GetFile(ctx, newFile.filepath)
											Expect(err).To(BeNil(), "file %s should exist", newFile.filepath)
											p := make([]byte, len(newFile.source_url_or_content))
											Expect(gbytes.TimeoutReader(file.Content, time.Second).Read(p)).To(Equal(len(newFile.source_url_or_content)))
											Expect(p).To(BeEquivalentTo(newFile.source_url_or_content))

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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_bytes", exampleFile.source_url_or_content),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_bytes", newFile.source_url_or_content),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_bytes", exampleFile.source_url_or_content),
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
										resource.TestCheckNoResourceAttr("freebox_remote_file."+resourceName, "source_bytes"),
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
		})

		Context("the checksum remains unchanged", func() {
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_bytes", exampleFile.source_url_or_content),
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
										resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_bytes", newFile.source_url_or_content),
										func(s *terraform.State) error {
											_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
											Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should no more exist", exampleFile.filepath)

											fileInfo, err := freeboxClient.GetFileInfo(ctx, newFile.filepath)
											Expect(err).To(BeNil())
											Expect(fileInfo.Name).To(Equal(newFile.filename))
											Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))

											file, err := freeboxClient.GetFile(ctx, newFile.filepath)
											Expect(err).To(BeNil(), "file %s should exist", newFile.filepath)
											p := make([]byte, len(newFile.source_url_or_content))
											Expect(gbytes.TimeoutReader(file.Content, time.Second).Read(p)).To(Equal(len(newFile.source_url_or_content)))
											Expect(p).To(BeEquivalentTo(newFile.source_url_or_content))

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
			writter, taskID, err := freeboxClient.FileUploadStart(ctx, types.FileUploadStartActionInput{
				Size:     len(exampleFile.source_url_or_content),
				Dirname:  freeboxTypes.Base64Path(go_path.Dir(exampleFile.filepath)),
				Filename: go_path.Base(exampleFile.filepath),
				Force:    freeboxTypes.FileUploadStartActionForceOverwrite,
			})
			Expect(err).To(BeNil())
			Expect(writter).ToNot(BeNil())
			Expect(taskID).ToNot(BeZero())

			Expect(gbytes.TimeoutWriter(writter, 30*time.Second).Write([]byte(exampleFile.source_url_or_content))).To(Equal(len(exampleFile.source_url_or_content)))
			Expect(gbytes.TimeoutCloser(writter, 30*time.Second).Close()).To(Succeed())

			DeferCleanup(func(ctx SpecContext, taskID int64) {
				Expect(freeboxClient.DeleteUploadTask(ctx, taskID)).To(Or(Succeed(), MatchError(client.ErrTaskNotFound)))
			}, int64(taskID))

			Eventually(func() types.UploadTask {
				uploadTask, err := freeboxClient.GetUploadTask(ctx, int64(taskID))
				if errors.Is(err, client.ErrTaskNotFound) { // The task is deleted by the time we get it
					return types.UploadTask{
						Status: types.UploadTaskStatusDone,
					}
				}
				Expect(err).To(BeNil())
				return uploadTask
			}, "30s").Should(MatchFields(IgnoreExtras, Fields{
				"Status": BeEquivalentTo(types.UploadTaskStatusDone),
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
									source_bytes = "` + exampleFile.source_url_or_content + `"
									destination_path = "` + exampleFile.filepath + `"
									checksum = "` + exampleFile.digest + `"

									polling = {
										upload = {
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
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_bytes", exampleFile.source_url_or_content),
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
