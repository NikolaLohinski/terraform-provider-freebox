package internal_test

import (
	"os"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/nikolalohinski/free-go/client"
	"github.com/nikolalohinski/free-go/types"
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
	if !ok {
		panic("FREEBOX_TOKEN environment variable is not set")
	}
	providerBlock = heredoc.Doc(`
		provider "freebox" {
			app_id = "` + appID + `"
			token  = "` + token + `"
		}
	`)
	existingDisk = file{
		filename:  "terraform-provider-freebox-alpine-3.20.0-aarch64.qcow2",
		directory: "VMs",
		filepath:  root + "/VMs/terraform-provider-freebox-alpine-3.20.0-aarch64.qcow2",
		digest:    "sha256:c7adb3d1fa28cd2abc208e83358a7d065116c6fce1c631ff1d03ace8a992bb69",
		source:    "https://raw.githubusercontent.com/NikolaLohinski/terraform-provider-freebox/main/examples/alpine-virt-3.20.0-aarch64.qcow2",
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
	// Download disk
	taskID, err := freeboxClient.AddDownloadTask(ctx, types.DownloadRequest{
		DownloadURLs:      []string{existingDisk.source},
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
	Eventually(func() types.DownloadTask {
		downloadTask, err := freeboxClient.GetDownloadTask(ctx, taskID)
		Expect(err).To(BeNil())
		return downloadTask
	}, "5m").Should(MatchFields(IgnoreExtras, Fields{
		"Status": BeEquivalentTo(types.DownloadTaskStatusDone),
		"Error":  BeEquivalentTo(freeboxTypes.DownloadTaskErrorNone),
	}))
})

func Must[T interface{}](r T, err error) T {
	if err != nil {
		panic(err)
	}
	return r
}

type file struct {
	filename  string
	directory string
	filepath  string
	digest    string
	source    string
}
