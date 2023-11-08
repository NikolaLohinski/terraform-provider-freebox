package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nikolalohinski/terraform-provider-freebox/provider"
)

func TestProvider(t *testing.T) {
	t.Setenv("TF_ACC", "true")
	RegisterFailHandler(Fail)

	RunSpecs(t, "provider")
}

var (
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"freebox": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
)
