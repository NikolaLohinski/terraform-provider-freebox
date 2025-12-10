package internal_test

import (
	"net/http"
	"os"
	"regexp"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/onsi/gomega/ghttp"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Context("provider freebox", func() {
	var (
		server   *ghttp.Server
		endpoint = new(string)
	)
	BeforeEach(func() {
		server = ghttp.NewServer()
		*endpoint = server.Addr()
	})
	Context("when the endpoint is invalid", func() {
		It("should fail to initialize the provider", func() {
			resource.Test(GinkgoT(), resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: heredoc.Doc(`
							provider "freebox" {
								endpoint = <<-EOF
									/!\ This is not an endpoint /!\"
								EOF
							}

							// unused data source but placed here to be able to unit test the provider logic
							data "freebox_api_version" "metadata" {}
						`),
						ExpectError: regexp.MustCompile("can not build base url from endpoint"),
					},
				},
			})
		})
	})
	Context("when everything is configured with environment variables", func() {
		BeforeEach(func() {
			os.Setenv("FREEBOX_ENDPOINT", *endpoint)
			os.Setenv("FREEBOX_VERSION", "v0")
			os.Setenv("FREEBOX_APP_ID", "test")
			os.Setenv("FREEBOX_TOKEN", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
			apiVersionHandler := ghttp.CombineHandlers(
				ghttp.VerifyHost(*endpoint),
				ghttp.VerifyRequest(http.MethodGet, "/api/v0/api_version"),
				ghttp.RespondWith(http.StatusOK, `{
					"box_model_name": "Freebox v7 (r1)",
					"api_base_url": "/api/",
					"https_port": 12345,
					"device_name": "Freebox Server",
					"https_available": true,
					"box_model": "fbxgw7-r1/full",
					"api_domain": "xxxxxxxx.fbxos.fr",
					"uid": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
					"api_version": "11.1",
					"device_type": "FreeboxServer7,1"
				}`),
			)
			server.AppendHandlers(
				apiVersionHandler,
				apiVersionHandler,
				apiVersionHandler,
			)
		})
		It("should run successfully", func() {
			resource.Test(GinkgoT(), resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: heredoc.Doc(`
							provider "freebox" {}

							// unused data source but placed here to be able to unit test the provider logic
							data "freebox_api_version" "metadata" {}
						`),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "uid", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"),
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "https_port", "12345"),
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "https_available", "true"),
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "box_model_name", "Freebox v7 (r1)"),
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "box_model", "fbxgw7-r1/full"),
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "api_domain", "xxxxxxxx.fbxos.fr"),
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "api_base_url", "/api/"),
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "api_version", "11.1"),
						),
					},
				},
			})
		})
	})
	Context("when endpoint and version are set from config", func() {
		BeforeEach(func() {
			os.Unsetenv("FREEBOX_ENDPOINT")
			os.Unsetenv("FREEBOX_VERSION")
			os.Unsetenv("FREEBOX_APP_ID")
			os.Unsetenv("FREEBOX_TOKEN")
			apiVersionHandler := ghttp.CombineHandlers(
				ghttp.VerifyHost(*endpoint),
				ghttp.VerifyRequest(http.MethodGet, "/api/v42/api_version"),
				ghttp.RespondWith(http.StatusOK, `{
					"box_model_name": "Freebox v7 (r1)",
					"api_base_url": "/api/",
					"https_port": 12345,
					"device_name": "Freebox Server",
					"https_available": true,
					"box_model": "fbxgw7-r1/full",
					"api_domain": "xxxxxxxx.fbxos.fr",
					"uid": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
					"api_version": "11.1",
					"device_type": "FreeboxServer7,1"
				}`),
			)
			server.AppendHandlers(
				apiVersionHandler,
				apiVersionHandler,
				apiVersionHandler,
			)
		})
		It("should run successfully", func() {
			resource.Test(GinkgoT(), resource.TestCase{
				IsUnitTest:               true,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: heredoc.Doc(`
							provider "freebox" {
								endpoint 	 = "` + *endpoint + `"
								api_version  = "v42"
							}

							// unused data source but placed here to be able to unit test the provider logic
							data "freebox_api_version" "metadata" {}
						`),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "uid", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"),
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "https_port", "12345"),
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "https_available", "true"),
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "box_model_name", "Freebox v7 (r1)"),
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "box_model", "fbxgw7-r1/full"),
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "api_domain", "xxxxxxxx.fbxos.fr"),
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "api_base_url", "/api/"),
							resource.TestCheckResourceAttr("data.freebox_api_version.metadata", "api_version", "11.1"),
						),
					},
				},
			})
		})
	})
})
