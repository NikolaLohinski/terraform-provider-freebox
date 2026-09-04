package internal_test

import (
	go_path "path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe(`resource "freebox_remote_directory" { ... }`, func() {
	var (
		resourceName  string
		directoryName string
		directoryPath string
		initialConfig string
	)

	BeforeEach(func(ctx SpecContext) {
		splitName := strings.Split(("test-" + uuid.New().String())[:30], "-")
		resourceName = strings.Join(splitName[:len(splitName)-1], "-")

		directoryName = resourceName
		directoryPath = go_path.Join(root, existingDisk.directory, directoryName)
	})

	JustBeforeEach(func(ctx SpecContext) {
		initialConfig = providerBlock + `
			resource "freebox_remote_directory" "` + resourceName + `" {
				destination_path = "` + directoryPath + `"
			}
		`
	})

	Context("create and delete", func() {
		It("should create and delete the directory", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: initialConfig,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_remote_directory."+resourceName, "destination_path", directoryPath),
							resource.TestCheckResourceAttr("freebox_remote_directory."+resourceName, "parents", "true"),
							resource.TestCheckResourceAttr("freebox_remote_directory."+resourceName, "remove_all", "true"),
							func(s *terraform.State) error {
								fileInfo, err := freeboxClient.GetFileInfo(ctx, directoryPath)
								Expect(err).To(BeNil())
								Expect(fileInfo.Name).To(Equal(directoryName))
								Expect(fileInfo.Type).To(BeEquivalentTo(freeboxTypes.FileTypeDirectory))
								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					_, err := freeboxClient.GetFileInfo(ctx, directoryPath)
					Expect(err).To(MatchError(client.ErrPathNotFound), "directory %s should not exist", directoryPath)
					return nil
				},
			})
		})
	})

	Context("with missing parent directories", func() {
		var nestedPath string

		JustBeforeEach(func(ctx SpecContext) {
			nestedPath = go_path.Join(directoryPath, "nested", "leaf")

			initialConfig = providerBlock + `
				resource "freebox_remote_directory" "` + resourceName + `" {
					destination_path = "` + nestedPath + `"
					parents           = true
				}
			`
		})

		It("should create the missing parent directories", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: initialConfig,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_remote_directory."+resourceName, "destination_path", nestedPath),
							func(s *terraform.State) error {
								fileInfo, err := freeboxClient.GetFileInfo(ctx, nestedPath)
								Expect(err).To(BeNil())
								Expect(fileInfo.Type).To(BeEquivalentTo(freeboxTypes.FileTypeDirectory))

								parentInfo, err := freeboxClient.GetFileInfo(ctx, go_path.Join(directoryPath, "nested"))
								Expect(err).To(BeNil())
								Expect(parentInfo.Type).To(BeEquivalentTo(freeboxTypes.FileTypeDirectory))

								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					_, err := freeboxClient.GetFileInfo(ctx, nestedPath)
					Expect(err).To(MatchError(client.ErrPathNotFound), "directory %s should not exist", nestedPath)
					return nil
				},
			})
		})
	})

	Context("update", func() {
		It("should move the directory when the destination path changes", func(ctx SpecContext) {
			renamedPath := directoryPath + "-renamed"

			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: initialConfig,
						Check:  resource.TestCheckResourceAttr("freebox_remote_directory."+resourceName, "destination_path", directoryPath),
					},
					{
						Config: providerBlock + `
							resource "freebox_remote_directory" "` + resourceName + `" {
								destination_path = "` + renamedPath + `"
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_remote_directory."+resourceName, "destination_path", renamedPath),
							func(s *terraform.State) error {
								_, err := freeboxClient.GetFileInfo(ctx, directoryPath)
								Expect(err).To(MatchError(client.ErrPathNotFound), "directory %s should not exist anymore", directoryPath)

								fileInfo, err := freeboxClient.GetFileInfo(ctx, renamedPath)
								Expect(err).To(BeNil())
								Expect(fileInfo.Type).To(BeEquivalentTo(freeboxTypes.FileTypeDirectory))

								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					_, err := freeboxClient.GetFileInfo(ctx, renamedPath)
					Expect(err).To(MatchError(client.ErrPathNotFound), "directory %s should not exist", renamedPath)
					return nil
				},
			})
		})

		It("should preserve configured attributes when moving the directory", func(ctx SpecContext) {
			renamedPath := directoryPath + "-renamed"

			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: providerBlock + `
							resource "freebox_remote_directory" "` + resourceName + `" {
								destination_path = "` + directoryPath + `"
								parents          = false
								remove_all       = false

								polling = {
									move = {
										interval = "2s"
										timeout  = "2m"
									}
									delete = {
										interval = "2s"
										timeout  = "2m"
									}
								}
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_remote_directory."+resourceName, "parents", "false"),
							resource.TestCheckResourceAttr("freebox_remote_directory."+resourceName, "remove_all", "false"),
							resource.TestCheckResourceAttrWith("freebox_remote_directory."+resourceName, "polling.move.interval", durationEqualFunc(2*time.Second)),
							resource.TestCheckResourceAttrWith("freebox_remote_directory."+resourceName, "polling.move.timeout", durationEqualFunc(2*time.Minute)),
						),
					},
					{
						Config: providerBlock + `
							resource "freebox_remote_directory" "` + resourceName + `" {
								destination_path = "` + renamedPath + `"
								parents          = false
								remove_all       = false

								polling = {
									move = {
										interval = "2s"
										timeout  = "2m"
									}
									delete = {
										interval = "2s"
										timeout  = "2m"
									}
								}
							}
						`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_remote_directory."+resourceName, "destination_path", renamedPath),
							resource.TestCheckResourceAttr("freebox_remote_directory."+resourceName, "parents", "false"),
							resource.TestCheckResourceAttr("freebox_remote_directory."+resourceName, "remove_all", "false"),
							resource.TestCheckResourceAttrWith("freebox_remote_directory."+resourceName, "polling.move.interval", durationEqualFunc(2*time.Second)),
							resource.TestCheckResourceAttrWith("freebox_remote_directory."+resourceName, "polling.move.timeout", durationEqualFunc(2*time.Minute)),
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					_, err := freeboxClient.GetFileInfo(ctx, renamedPath)
					Expect(err).To(MatchError(client.ErrPathNotFound), "directory %s should not exist", renamedPath)
					return nil
				},
			})
		})
	})

	Context("import", func() {
		BeforeEach(func(ctx SpecContext) {
			_, err := freeboxClient.CreateDirectory(ctx, go_path.Join(root, existingDisk.directory), directoryName)
			Expect(err).To(BeNil())
		})

		It("should import the directory by its path", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:             initialConfig,
						ResourceName:       "freebox_remote_directory." + resourceName,
						ImportState:        true,
						ImportStateId:      directoryPath,
						ImportStatePersist: true,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_remote_directory."+resourceName, "destination_path", directoryPath),
						),
						Destroy: true,
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					_, err := freeboxClient.GetFileInfo(ctx, directoryPath)
					Expect(err).To(MatchError(client.ErrPathNotFound), "directory %s should not exist", directoryPath)
					return nil
				},
			})
		})
	})
})
