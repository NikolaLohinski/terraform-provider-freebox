package internal_test

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	"github.com/nikolalohinski/terraform-provider-freebox/internal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

func TestProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "internal")
}

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"freebox": providerserver.NewProtocol6WithError(internal.NewProvider("test")()),
	}
	endpoint      string
	version       string
	appID         string
	token         string
	root          string
	providerBlock string

	freeboxClient client.Client

	existingDisk file

	randGenerator *rand.Rand
)

var _ = BeforeSuite(func(ctx SpecContext) {
	var ok bool
	endpoint, ok = os.LookupEnv("FREEBOX_ENDPOINT")
	if !ok {
		endpoint = "http://mafreebox.freebox.fr"
	}
	version, ok = os.LookupEnv("FREEBOX_VERSION")
	if !ok {
		version = "latest"
	}
	root, ok = os.LookupEnv("FREEBOX_ROOT")
	if !ok {
		root = "Freebox"
	}
	appID, ok = os.LookupEnv("FREEBOX_APP_ID")
	if !ok {
		appID = "terraform-provider-freebox"
	}

	token, ok = os.LookupEnv("FREEBOX_TOKEN")
	Expect(ok).To(BeTrue(), "FREEBOX_TOKEN environment variable is not set")

	providerBlock = heredoc.Doc(`
		provider "freebox" {
			app_id = "` + appID + `"
			token  = "` + token + `"
		}
	`)
	existingDisk = file{
		filename:              "terraform-provider-freebox-alpine-3.20.0-aarch64.qcow2",
		directory:             "VMs",
		filepath:              root + "/VMs/terraform-provider-freebox-alpine-3.20.0-aarch64.qcow2",
		digest:                "sha256:c7adb3d1fa28cd2abc208e83358a7d065116c6fce1c631ff1d03ace8a992bb69",
		source_url_or_content: "https://raw.githubusercontent.com/NikolaLohinski/terraform-provider-freebox/main/examples/alpine-virt-3.20.0-aarch64.qcow2",
	}

	fc, err := client.New(endpoint, version)
	Expect(err).To(BeNil())

	freeboxClient = fc.WithAppID(appID).WithPrivateToken(token)

	permissions, err := freeboxClient.Login(ctx)
	Expect(err).To(BeNil())
	Expect(permissions).To(MatchFields(IgnoreExtras, Fields{
		"Downloader": BeTrue(),
		"Explorer":   BeTrue(),
		"Settings":   BeTrue(),
	}))

	// Create directory
	_, err = freeboxClient.CreateDirectory(ctx, root, existingDisk.directory)
	Expect(err).To(Or(BeNil(), Equal(client.ErrDestinationConflict)))

	By("Checking the existingDisk", func() {
		hashTask, err := freeboxClient.AddHashFileTask(ctx, freeboxTypes.HashPayload{
			Path:     freeboxTypes.Base64Path(existingDisk.filepath),
			HashType: freeboxTypes.HashTypeSHA256,
		})
		Expect(err).To(BeNil())

		Eventually(func() freeboxTypes.FileSystemTask {
			hashTask, err = freeboxClient.GetFileSystemTask(ctx, hashTask.ID)
			Expect(err).To(BeNil())
			Expect(hashTask.Type).To(BeEquivalentTo(freeboxTypes.FileTaskTypeHash))
			return hashTask
		}, "1m").Should(MatchFields(IgnoreExtras, Fields{
			"State": Or(BeEquivalentTo(freeboxTypes.FileTaskStateDone), BeEquivalentTo(freeboxTypes.FileTaskStateFailed)),
			"Error": Or(BeEquivalentTo(freeboxTypes.FileTaskErrorNone), BeEquivalentTo(freeboxTypes.FileTaskErrorFileNotFound)),
		}))

		switch hashTask.State {
		case freeboxTypes.FileTaskStateFailed:
			Expect(hashTask.Error).To(BeEquivalentTo(freeboxTypes.FileTaskErrorFileNotFound))

			// Download disk
			taskID, err := freeboxClient.AddDownloadTask(ctx, freeboxTypes.DownloadRequest{
				DownloadURLs:      []string{existingDisk.source_url_or_content},
				Hash:              existingDisk.digest,
				DownloadDirectory: root + "/" + existingDisk.directory,
				Filename:          existingDisk.filename,
			})
			Expect(err).To(BeNil())

			// Cleanup download task
			DeferCleanup(func(ctx SpecContext) {
				Expect(freeboxClient.DeleteDownloadTask(ctx, taskID)).To(Succeed())
			})

			// Wait for download task to be done
			Eventually(func() freeboxTypes.DownloadTask {
				downloadTask, err := freeboxClient.GetDownloadTask(ctx, taskID)
				Expect(err).To(BeNil())
				return downloadTask
			}, "5m").Should(MatchFields(IgnoreExtras, Fields{
				"Status": BeEquivalentTo(freeboxTypes.DownloadTaskStatusDone),
				"Error":  BeEquivalentTo(freeboxTypes.DownloadTaskErrorNone),
			}))
		case freeboxTypes.FileTaskStateDone:
			result, err := freeboxClient.GetHashResult(ctx, hashTask.ID)
			Expect(err).To(BeNil())

			if fmt.Sprintf("sha256:%s", result) != existingDisk.digest {
				Fail(fmt.Sprintf("Hash result does not match expected digest. Please delete the file %s and run the test again: %s != %s", existingDisk.filepath, fmt.Sprintf("sha256:%s", result), existingDisk.digest))
			}
		default:
			Fail(fmt.Sprintf("Hash task state is not done or failed: %s", hashTask.State))
		}
	})
})

var _ = AfterEach(func(ctx SpecContext) {
	By("Ensure the file still exists", func() {
		_, err := freeboxClient.GetFile(ctx, existingDisk.filepath)
		Expect(err).To(BeNil())
	})
})

func Must[T interface{}](r T, err error) T {
	if err != nil {
		panic(err)
	}
	return r
}

type file struct {
	filename              string
	directory             string
	filepath              string
	digest                string
	source_url_or_content string
}

var _ = BeforeEach(func(ctx SpecContext) {
	randGenerator = rand.New(rand.NewSource(GinkgoRandomSeed()))
	uuid.SetRand(randGenerator)
})

var _ = BeforeEach(func(ctx SpecContext) {
	dlTasks, err := freeboxClient.ListDownloadTasks(ctx)
	Expect(err).To(BeNil())

	DeferCleanup(func(ctx SpecContext, oldTasks []freeboxTypes.DownloadTask) {
		newTasks, err := freeboxClient.ListDownloadTasks(ctx)
		Expect(err).To(BeNil())

		Expect(newTasks).To(BeEquivalentTo(oldTasks), "Download tasks should be the same before and after test")
	}, dlTasks)

	fsTasks, err := freeboxClient.ListFileSystemTasks(ctx)
	Expect(err).To(BeNil())

	DeferCleanup(func(ctx SpecContext, oldTasks []freeboxTypes.FileSystemTask) {
		newTasks, err := freeboxClient.ListFileSystemTasks(ctx)
		Expect(err).To(BeNil())

		Expect(newTasks).To(BeEquivalentTo(oldTasks), "File system tasks should be the same before and after test")
	}, fsTasks)

	DeferCleanup(func(ctx SpecContext) {
		hashTask, err := freeboxClient.AddHashFileTask(ctx, freeboxTypes.HashPayload{
			Path:     freeboxTypes.Base64Path(existingDisk.filepath),
			HashType: freeboxTypes.HashTypeSHA256,
		})
		Expect(err).To(BeNil())

		defer func() {
			Expect(freeboxClient.DeleteFileSystemTask(ctx, hashTask.ID)).To(Succeed())
		}()

		Eventually(func() interface{} {
			hashTask, err = freeboxClient.GetFileSystemTask(ctx, hashTask.ID)
			Expect(err).To(BeNil())
			Expect(hashTask.Type).To(BeEquivalentTo(freeboxTypes.FileTaskTypeHash))

			if hashTask.State == freeboxTypes.FileTaskStateFailed {
				StopTrying("Hash task failed")
			}

			return hashTask.State
		}, "1m").Should(BeEquivalentTo(freeboxTypes.FileTaskStateDone))

		Expect(hashTask.State).To(BeEquivalentTo(freeboxTypes.FileTaskStateDone))

		result, err := freeboxClient.GetHashResult(ctx, hashTask.ID)
		Expect(err).To(BeNil())
		Expect(fmt.Sprintf("sha256:%s", result)).To(BeEquivalentTo(existingDisk.digest))
	})
})
