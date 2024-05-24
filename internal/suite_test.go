package internal_test

import (
	"os"
	"testing"

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
	endpoint      = os.Getenv("FREEBOX_ENDPOINT")
	version       = os.Getenv("FREEBOX_VERSION")
	appID         = os.Getenv("FREEBOX_APP_ID")
	token         = os.Getenv("FREEBOX_TOKEN")
	providerBlock = `
		provider "freebox" {
			app_id = "` + appID + `"
			token  = "` + token + `"
		}
	`
)

func Must[T interface{}](r T, err error) T {
	if err != nil {
		panic(err)
	}
	return r
}
