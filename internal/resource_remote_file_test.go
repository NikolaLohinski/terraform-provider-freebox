package internal_test

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/nikolalohinski/free-go/client"
	"github.com/nikolalohinski/free-go/types"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Context(`resource "freebox_remote_file" { ... }`, Ordered, func() {
	var (
		files = map[string]file{
			"alpine": existingDisk,
			"example": {
				filename:  "terraform-provider-freebox-example.txt",
				directory: "VMs",
				filepath:  path.Join(root, "VMs/terraform-provider-freebox-example.txt"),
				digest:    "sha256:184725f66890632c7e67ec1713c50aa181c1bc60ee166c9ae13a48f1d60684b0",
				source:    "https://raw.githubusercontent.com/holyhope/terraform-provider-freebox/daef06ced800f81f0ae3355bae1ace9795d455b9/examples/file-to-download.txt",
			},
		}

		freeboxClient client.Client

		cleanAllFile func(context.Context, ...file)
	)

	BeforeAll(func(ctx SpecContext) {
		// Note: alpine already exists
		// Download the example file
		file := files["example"]
		// Create directory
		_, err := freeboxClient.CreateDirectory(ctx, root, file.directory)
		Expect(err).To(Or(BeNil(), Equal(client.ErrDestinationConflict)))
	})

	AfterAll(func(ctx SpecContext) {
		cleanAllFile(ctx, files["example"])
	})

	Context("create and delete (CD)", func() {
		It("should download and delete the file", func(ctx SpecContext) {
			splitName := strings.Split(("test-CD-" + uuid.New().String())[:30], "-")
			name := strings.Join(splitName[:len(splitName)-1], "-")

			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerBlock + `
							resource "freebox_remote_file" "` + name + `" {
								source_url = "` + files["example"].source + `"
								destination_path = "` + files["example"].filepath + `"
								checksum = "` + files["example"].digest + `"
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_remote_file."+name, "source_url", files["example"].source),
							resource.TestCheckResourceAttr("freebox_remote_file."+name, "destination_path", files["example"].filepath),
							resource.TestCheckResourceAttr("freebox_remote_file."+name, "checksum", files["example"].digest),
							resource.TestCheckResourceAttrWith("freebox_remote_file."+name, "task_id", func(value string) error {
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
								identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_remote_file."+name].Primary.Attributes["task_id"])
								Expect(err).To(BeNil())
								task, err := freeboxClient.GetDownloadTask(ctx, int64(identifier))
								Expect(err).To(BeNil())
								Expect(task.Name).To(Equal(files["example"].filename))
								Expect(task.Status).To(BeEquivalentTo(types.DownloadTaskStatusDone))

								fileInfo, err := freeboxClient.GetFileInfo(ctx, files["example"].filepath)
								Expect(err).To(BeNil())
								Expect(fileInfo.Name).To(Equal(files["example"].filename))
								Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))
								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					_, err := freeboxClient.GetFileInfo(ctx, files["example"].filepath)
					Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", files["example"].filepath)
					return nil
				},
			})
		})
	})

	Context("create, update and delete (CUD)", func() {
		It("should create, update and finally delete a file", func(ctx SpecContext) {
			splitName := strings.Split(("test-CUD-" + uuid.New().String())[:30], "-")
			name := strings.Join(splitName[:len(splitName)-1], "-")
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerBlock + `
							resource "freebox_remote_file" "` + name + `" {
								source_url = "` + files["example"].source + `"
								destination_path = "` + files["example"].filepath + `"
								checksum = "` + files["example"].digest + `"
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_remote_file."+name, "checksum", files["example"].digest),
							resource.TestCheckResourceAttr("freebox_remote_file."+name, "destination_path", files["example"].filepath),
							resource.TestCheckResourceAttr("freebox_remote_file."+name, "source_url", files["example"].source),
						),
					},
					{
						Config: providerBlock + `
							resource "freebox_remote_file" "` + name + `" {
								source_url = "` + files["alpine"].source + `"
								destination_path = "` + files["alpine"].filepath + `"
								checksum = "` + files["alpine"].digest + `"
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_remote_file."+name, "checksum", files["alpine"].digest),
							resource.TestCheckResourceAttr("freebox_remote_file."+name, "destination_path", files["alpine"].filepath),
							resource.TestCheckResourceAttr("freebox_remote_file."+name, "source_url", files["alpine"].source),
							func(s *terraform.State) error {
								identifier, err := strconv.Atoi(s.RootModule().Resources["freebox_remote_file."+name].Primary.Attributes["task_id"])
								Expect(err).To(BeNil())
								task, err := freeboxClient.GetDownloadTask(ctx, int64(identifier))
								Expect(err).To(BeNil())
								Expect(task.Name).To(Equal(files["alpine"].filename))
								Expect(task.Status).To(BeEquivalentTo(types.DownloadTaskStatusDone))

								fileInfo, err := freeboxClient.GetFileInfo(ctx, files["alpine"].filepath)
								Expect(err).To(BeNil())
								Expect(fileInfo.Name).To(Equal(files["alpine"].filename))
								Expect(fileInfo.Type).To(BeEquivalentTo(types.FileTypeFile))
								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					for _, file := range files {
						_, err := freeboxClient.GetFileInfo(ctx, file.filepath)
						Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", file.filepath)
					}
					return nil
				},
			})
		})
	})
	Context("import and delete (ID)", func() {
		var (
			remoteFileTaskID = new(int64)
			name             = new(string)
		)
		BeforeEach(func(ctx SpecContext) {
			splitName := strings.Split(("test-ID-" + uuid.New().String())[:30], "-")
			*name = strings.Join(splitName[:len(splitName)-1], "-")
			taskID := Must(freeboxClient.AddDownloadTask(ctx, types.DownloadRequest{
				DownloadDirectory: path.Join(root, files["example"].directory),
				DownloadURLs:      []string{files["example"].source},
				Filename: 		   files["example"].filename,
				Hash: 		       files["example"].digest,
			}))

			Eventually(func() types.DownloadTask {
				downloadTask, err := freeboxClient.GetDownloadTask(ctx, taskID)
				Expect(err).To(BeNil())
				return downloadTask
			}).Should(MatchFields(IgnoreExtras, Fields{
				"Status": BeEquivalentTo(types.DownloadTaskStatusDone),
			}))

			*remoteFileTaskID = taskID
		})

		It("should import and then delete a remote file", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerBlock + `
							resource "freebox_remote_file" "` + *name + `" {
								source_url = "` + files["example"].source + `"
								destination_path = "` + files["example"].filepath + `"
								checksum = "` + files["example"].digest + `"
							}
						`,
						ResourceName:       "freebox_remote_file." + *name,
						ImportState:        true,
						ImportStateId:      strconv.Itoa(int(*remoteFileTaskID)),
						ImportStatePersist: true,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_remote_file."+*name, "checksum", files["example"].digest),
							resource.TestCheckResourceAttr("freebox_remote_file."+*name, "destination_path", files["example"].filepath),
							resource.TestCheckResourceAttr("freebox_remote_file."+*name, "source_url", files["example"].source),
							resource.TestCheckResourceAttr("freebox_remote_file."+*name, "task_id", strconv.Itoa(int(*remoteFileTaskID))),
						),
						Destroy: true,
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					_, err := freeboxClient.GetFileInfo(ctx, files["example"].filepath)
					Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", files["example"].filepath)
					return nil
				},
			})
		})
	})

	cleanAllFile = func (ctx context.Context, files ...file) {
		diskToRemove := []string{}
		for _, file := range files {
			// Check that the file exists and if so, delete it
			_, err := freeboxClient.GetFileInfo(ctx, file.filepath)
			if err == nil {
				diskToRemove = append(diskToRemove, file.filepath)
				return
			}
		}

		if len(diskToRemove) == 0 {
			return
		}

		task, err := freeboxClient.RemoveFiles(ctx, diskToRemove)
		Expect(err).To(BeNil())

		Eventually(func() types.FileSystemTask {
			deleteTask, err := freeboxClient.GetFileSystemTask(ctx, task.ID)
			Expect(err).To(BeNil())
			return deleteTask
		}, "5m").Should(MatchFields(IgnoreExtras, Fields{
			"State": BeEquivalentTo(types.FileTaskStateDone),
		}))

		for _, file := range files {
			_, err := freeboxClient.GetFileInfo(ctx, file.filepath)
			Expect(err).To(MatchError(client.ErrPathNotFound), "file %s should not exist", file.filepath)
		}
	}
})
