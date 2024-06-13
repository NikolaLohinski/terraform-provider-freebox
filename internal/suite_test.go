package internal_test

import (
	"os"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/nikolalohinski/terraform-provider-freebox/internal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
)

func init() {
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
}

func Must[T interface{}](r T, err error) T {
	if err != nil {
		panic(err)
	}
	return r
}
