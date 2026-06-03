package internal_test

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe(`data "freebox_system_info" { ... }`, func() {
	var (
		config     string
		resName    string
		systemInfo freeboxTypes.SystemConfig
	)

	BeforeEach(func(ctx SpecContext) {
		splitName := strings.Split(("test-" + uuid.New().String())[:30], "-")
		resName = strings.Join(splitName[:len(splitName)-1], "-")

		var err error
		systemInfo, err = freeboxClient.GetSystemInfo(ctx)
		Expect(err).To(BeNil())
	})

	JustBeforeEach(func() {
		config = providerBlock + `
			data "freebox_system_info" "` + resName + `" {
			}
		`
	})

	It("should fetch system information", func(ctx SpecContext) {
		resource.UnitTest(GinkgoT(), resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						func(s *terraform.State) error {
							state := s.RootModule().Resources["data.freebox_system_info."+resName].Primary.Attributes

							Expect(state["firmware_version"]).To(Equal(systemInfo.FirmwareVersion))
							Expect(state["mac"]).To(Equal(systemInfo.Mac))
							Expect(state["serial"]).To(Equal(systemInfo.Serial))
							Expect(state["board_name"]).To(Equal(systemInfo.BoardName))
							Expect(state["uptime_val"]).To(Equal(fmt.Sprintf("%d", systemInfo.UptimeVal)))
							Expect(state["temp_cpum"]).To(Equal(fmt.Sprintf("%d", systemInfo.TempCPUM)))
							Expect(state["temp_cpub"]).To(Equal(fmt.Sprintf("%d", systemInfo.TempCPUB)))
							Expect(state["temp_sw"]).To(Equal(fmt.Sprintf("%d", systemInfo.TempSW)))
							Expect(state["fan_rpm"]).To(Equal(fmt.Sprintf("%d", systemInfo.FanRPM)))
							Expect(state["user_main_storage"]).To(Equal(systemInfo.UserMainStorage))

							return nil
						},
					),
				},
			},
		})
	})
})
