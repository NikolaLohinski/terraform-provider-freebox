package internal_test

import (
	"fmt"
	"path"
	"regexp"
	"strconv"

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
		exampleFile file
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
			source:    "https://raw.githubusercontent.com/holyhope/terraform-provider-freebox/refs/heads/resources/remote_file/examples/file-to-download.txt",
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
								}
							`,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
								resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "task_id", func(value string) error {
									taskID, err := strconv.Atoi(value)
									if err != nil {
										return err
									}

									if taskID == 0 {
										return fmt.Errorf("task_id is not set")
									}

									return nil
								}),
								func(s *terraform.State) error {
									identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_remote_file."+resourceName].Primary.Attributes["task_id"])
									Expect(err).To(BeNil())
									task, err := freeboxClient.GetDownloadTask(ctx, int64(identifier))
									Expect(err).To(BeNil())
									Expect(task.Name).To(Equal(exampleFile.filename))
									Expect(task.Status).To(BeEquivalentTo(types.DownloadTaskStatusDone))

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
						identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_remote_file."+resourceName].Primary.Attributes["task_id"])
						Expect(err).To(BeNil())
						Expect(identifier).ToNot(BeZero())

						_, err = freeboxClient.GetDownloadTask(ctx, int64(identifier))
						Expect(err).To(MatchError(client.ErrTaskNotFound))

						_, err = freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
						Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)
						return nil
					},
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
								}
							`,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
								resource.TestCheckResourceAttrWith("freebox_remote_file."+resourceName, "task_id", func(value string) error {
									taskID, err := strconv.Atoi(value)
									if err != nil {
										return err
									}

									if taskID == 0 {
										return fmt.Errorf("task_id is not set")
									}

									return nil
								}),
								func(s *terraform.State) error {
									identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_remote_file."+resourceName].Primary.Attributes["task_id"])
									Expect(err).To(BeNil())
									task, err := freeboxClient.GetDownloadTask(ctx, int64(identifier))
									Expect(err).To(BeNil())
									Expect(task.Name).To(Equal(exampleFile.filename))
									Expect(task.Status).To(BeEquivalentTo(types.DownloadTaskStatusDone))

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
						identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_remote_file."+resourceName].Primary.Attributes["task_id"])
						Expect(err).To(BeNil())
						Expect(identifier).ToNot(BeZero())

						_, err = freeboxClient.GetDownloadTask(ctx, int64(identifier))
						Expect(err).To(MatchError(client.ErrTaskNotFound))

						_, err = freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
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
		It("should create, update and finally delete a file", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerBlock + `
							resource "freebox_remote_file" "` + resourceName + `" {
								source_url = "` + exampleFile.source + `"
								destination_path = "` + exampleFile.filepath + `"
								checksum = "` + exampleFile.digest + `"
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
							resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
							resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
							func(s *terraform.State) error {
								identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_remote_file."+resourceName].Primary.Attributes["task_id"])
								Expect(err).To(BeNil())
								task, err := freeboxClient.GetDownloadTask(ctx, int64(identifier))
								Expect(err).To(BeNil())
								Expect(task.Name).To(Equal(exampleFile.filename))
								Expect(task.Status).To(BeEquivalentTo(types.DownloadTaskStatusDone))

								fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
								Expect(err).To(BeNil())
								Expect(fileInfo.Name).To(Equal(exampleFile.filename))
								Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))
								return nil
							},
						),
					},
					{
						Config: providerBlock + `
							resource "freebox_remote_file" "` + resourceName + `" {
								source_url = "` + exampleFile.source + `"
								destination_path = "` + exampleFile.filepath + `.new"
								checksum = "` + exampleFile.digest + `"
							}
						`,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction("freebox_remote_file."+resourceName, plancheck.ResourceActionUpdate),
							},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
							resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath + ".new"),
							resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
							func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should no more exist", exampleFile.filepath)

								identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_remote_file."+resourceName].Primary.Attributes["task_id"])
								Expect(err).To(BeNil())
								task, err := freeboxClient.GetDownloadTask(ctx, int64(identifier))
								Expect(err).To(BeNil())
								Expect(task.Name).To(Equal(exampleFile.filename + ".new"))
								Expect(task.Status).To(BeEquivalentTo(types.DownloadTaskStatusDone))

								fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath + ".new")
								Expect(err).To(BeNil())
								Expect(fileInfo.Name).To(Equal(exampleFile.filename + ".new"))
								Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))
								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_remote_file."+resourceName].Primary.Attributes["task_id"])
					Expect(err).To(BeNil())
					Expect(identifier).ToNot(BeZero())

					_, err = freeboxClient.GetDownloadTask(ctx, int64(identifier))
					Expect(err).To(MatchError(client.ErrTaskNotFound))

					_, err = freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
					Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)

					_, err = freeboxClient.GetFileInfo(ctx, exampleFile.filepath + ".new")
					Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath+".new")

					return nil
				},
			})
		})
		Context("when only the task changed", func() {
			It("should update the resource", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: providerBlock + `
								resource "freebox_remote_file" "` + resourceName + `" {
									source_url = "` + exampleFile.source + `"
									destination_path = "` + exampleFile.filepath + `"
									checksum = "` + exampleFile.digest + `"
								}
							`,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
								func(s *terraform.State) error {
									identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_remote_file."+resourceName].Primary.Attributes["task_id"])
									Expect(err).To(BeNil())
									task, err := freeboxClient.GetDownloadTask(ctx, int64(identifier))
									Expect(err).To(BeNil())
									Expect(task.Name).To(Equal(exampleFile.filename))
									Expect(task.Status).To(BeEquivalentTo(types.DownloadTaskStatusDone))

									fileInfo, err := freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
									Expect(err).To(BeNil())
									Expect(fileInfo.Name).To(Equal(exampleFile.filename))
									Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))
									return nil
								},
							),
						},
						{
							Config: providerBlock + `
								resource "freebox_remote_file" "` + resourceName + `" {
									source_url = "` + exampleFile.source + `"
									destination_path = "` + exampleFile.filepath + `"
									checksum = "` + exampleFile.digest + `"
									task_id = null
								}
							`,
							ConfigPlanChecks: resource.ConfigPlanChecks{
								PreApply: []plancheck.PlanCheck{
									plancheck.ExpectEmptyPlan(),
								},
							},
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
								func(s *terraform.State) error {
									identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_remote_file."+resourceName].Primary.Attributes["task_id"])
									Expect(err).To(BeNil())
									task, err := freeboxClient.GetDownloadTask(ctx, int64(identifier))
									Expect(err).To(BeNil())
									Expect(task.Name).To(Equal(exampleFile.filename))
									Expect(task.Status).To(BeEquivalentTo(types.DownloadTaskStatusDone))

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
						identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_remote_file."+resourceName].Primary.Attributes["task_id"])
						Expect(err).To(BeNil())
						Expect(identifier).ToNot(BeZero())

						_, err = freeboxClient.GetDownloadTask(ctx, int64(identifier))
						Expect(err).To(MatchError(client.ErrTaskNotFound))

						_, err = freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
						Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)

						_, err = freeboxClient.GetFileInfo(ctx, exampleFile.filepath + ".new")
						Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath+".new")

						return nil
					},
				})
			})
		})
	})
	Context("import and delete (ID)", func() {
		var remoteFileTaskID int64

		BeforeEach(func(ctx SpecContext) {
			taskID, err := freeboxClient.AddDownloadTask(ctx, types.DownloadRequest{
				DownloadDirectory: path.Join(root, exampleFile.directory),
				DownloadURLs:      []string{exampleFile.source},
				Filename: 		   exampleFile.filename,
				Hash: 		       exampleFile.digest,
			})
			Expect(err).To(BeNil())
			Expect(taskID).ToNot(BeZero())

			remoteFileTaskID = taskID

			Eventually(func() types.DownloadTask {
				downloadTask, err := freeboxClient.GetDownloadTask(ctx, taskID)
				Expect(err).To(BeNil())
				return downloadTask
			}).Should(MatchFields(IgnoreExtras, Fields{
				"Status": BeEquivalentTo(types.DownloadTaskStatusDone),
			}))
		})

		Describe("import and delete with task ID", func() {
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
								}
							`,
							ResourceName:       "freebox_remote_file." + resourceName,
							ImportState:        true,
							ImportStateId:      strconv.Itoa(int(remoteFileTaskID)),
							ImportStatePersist: true,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "checksum", exampleFile.digest),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "destination_path", exampleFile.filepath),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "source_url", exampleFile.source),
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "task_id", strconv.Itoa(int(remoteFileTaskID))),
							),
							Destroy: true,
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_remote_file."+resourceName].Primary.Attributes["task_id"])
						Expect(err).To(BeNil())
						Expect(identifier).ToNot(BeZero())

						_, err = freeboxClient.GetDownloadTask(ctx, int64(identifier))
						Expect(err).To(MatchError(client.ErrTaskNotFound))

						_, err = freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
						Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)
						return nil
					},
				})
			})
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
								resource.TestCheckResourceAttr("freebox_remote_file."+resourceName, "task_id", strconv.Itoa(int(remoteFileTaskID))),
							),
							Destroy: true,
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_remote_file."+resourceName].Primary.Attributes["task_id"])
						Expect(err).To(BeNil())
						Expect(identifier).ToNot(BeZero())

						_, err = freeboxClient.GetDownloadTask(ctx, int64(identifier))
						Expect(err).To(MatchError(client.ErrTaskNotFound))

						_, err = freeboxClient.GetFileInfo(ctx, exampleFile.filepath)
						Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", exampleFile.filepath)
						return nil
					},
				})
			})
		})
	})
})
