package internal_test

import (
	"os"
	go_path "path"
	"time"

	"github.com/nikolalohinski/free-go/client"
	"github.com/nikolalohinski/free-go/types"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Context(`resource "freebox_remote_file" { source_local_file = ... }`, func() {
	var (
		localFilePath string
		exampleFile   file
		resourceName  string
		initialConfig string
	)

	const (
		fileContent = "data"
		// echo -n data | sha256sum
		fileDigest = "sha256:3a6eb0790f39ac87c94f3856b2dd2c5d110e6811602261a9a923d3bb23adc8b7"
	)

	BeforeEach(func(ctx SpecContext) {
		resourceName = "test-" + uuid.NewString()
		filename := resourceName + ".txt"
		exampleFile = file{
			filename:  filename,
			directory: existingDisk.directory,
			filepath:  go_path.Join(root, existingDisk.directory, filename),
			digest:    fileDigest,
		}

		localFileHandle, err := os.CreateTemp("", "freebox-local-upload-*.txt")
		Expect(err).To(BeNil())
		localFilePath = localFileHandle.Name()
		DeferCleanup(func() { os.Remove(localFilePath) })

		_, err = localFileHandle.WriteString(fileContent)
		Expect(err).To(BeNil())
		Expect(localFileHandle.Close()).To(Succeed())
	})

	JustBeforeEach(func(ctx SpecContext) {
		initialConfig = providerBlock + `
			resource "freebox_remote_file" "` + resourceName + `" {
				source_local_file = "` + localFilePath + `"
				destination_path  = "` + exampleFile.filepath + `"
				checksum          = "` + exampleFile.digest + `"

				polling = {
					upload = {
						interval = "1s"
						timeout  = "1m"
					}
					delete = {
						interval = "1s"
						timeout  = "1m"
					}
					checksum_compute = {
						interval = "1s"
						timeout  = "1m"
					}
				}
			}
		`
	})

	Context("create and delete", func() {
		It("should upload, verify the checksum and delete the file", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: initialConfig,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_local_file", localFilePath),
							resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
							resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
							func(s *terraform.State) error {
								fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
								Expect(err).To(BeNil())
								Expect(fileInfo.Name).To(Equal(exampleFile.filename))
								Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))

								f, err := freeboxClient.GetFile(ctx, exampleFile.filepath)
								Expect(err).To(BeNil())
								p := make([]byte, len(fileContent))
								Expect(gbytes.TimeoutReader(f.Content, time.Second).Read(p)).To(Equal(len(fileContent)))
								Expect(string(p)).To(Equal(fileContent))

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

	Context("create, update and delete", func() {
		Context("when source_local_file changes", func() {
			var newLocalFilePath string

			const (
				newFileContent = "data-new"
				// echo -n data-new | sha256sum
				newFileDigest = "sha256:28369943a1e0316edf2c8d7fb01cc70899b7234c7d5f8940c75fae7693d3f757"
			)

			BeforeEach(func(ctx SpecContext) {
				newFileHandle, err := os.CreateTemp("", "freebox-local-upload-new-*.txt")
				Expect(err).To(BeNil())
				newLocalFilePath = newFileHandle.Name()
				DeferCleanup(func() { os.Remove(newLocalFilePath) })

				_, err = newFileHandle.WriteString(newFileContent)
				Expect(err).To(BeNil())
				Expect(newFileHandle.Close()).To(Succeed())
			})

			It("should replace the resource", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", fileDigest),
							),
						},
						{
							Config: terraformConfigWithAttribute("source_local_file", newLocalFilePath)(
								terraformConfigWithAttribute("checksum", newFileDigest)(initialConfig),
							),
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply: []plancheck.PlanCheck{
									plancheck.ExpectResourceAction("freebox_remote_file."+resourceName, plancheck.ResourceActionReplace),
								},
							},
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_local_file", newLocalFilePath),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", newFileDigest),
								func(s *terraform.State) error {
									f, err := freeboxClient.GetFile(ctx, exampleFile.filepath)
									Expect(err).To(BeNil())
									p := make([]byte, len(newFileContent))
									Expect(gbytes.TimeoutReader(f.Content, time.Second).Read(p)).To(Equal(len(newFileContent)))
									Expect(string(p)).To(Equal(newFileContent))
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

	Context("with parents=true and a nested destination path", func() {
		BeforeEach(func(ctx SpecContext) {
			resourceName = "test-" + uuid.NewString()
			filename := resourceName + ".txt"
			subdir := resourceName + "-sub"
			exampleFile = file{
				filename:  filename,
				directory: go_path.Join(existingDisk.directory, subdir),
				filepath:  go_path.Join(root, existingDisk.directory, subdir, filename),
				digest:    fileDigest,
			}
		})

		It("should create parent directories and upload the file", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: initialConfig,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "parents", "true"),
							resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
							func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
								Expect(err).To(BeNil(), "file %s should exist after upload", exampleFile.filepath)
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
