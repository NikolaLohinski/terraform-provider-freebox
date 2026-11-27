package internal

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

func TestLanConfigPinConfiguredFields(t *testing.T) {
	t.Parallel()

	var model lanConfigResourceModel
	model.fromClientType(freeboxTypes.LanConfig{
		IP:          "192.0.2.1",
		Name:        "api-returned-name",
		NameDNS:     "api-returned-dns",
		NameMDNS:    "api-returned-mdns",
		NameNetBIOS: "api-returned-netbios",
		Mode:        freeboxTypes.LanModeRouter,
	})

	configured := lanConfigResourceModel{
		IP:          basetypes.NewStringValue("192.0.2.1"),
		Name:        basetypes.NewStringValue("configured-name"),
		NameDNS:     basetypes.NewStringNull(), // left unconfigured
		NameMDNS:    basetypes.NewStringValue("configured-mdns"),
		NameNetBIOS: basetypes.NewStringUnknown(), // unknown at plan time
		Mode:        basetypes.NewStringValue(freeboxTypes.LanModeRouter),
	}

	model.pinConfiguredFields(configured)

	if got := model.Name.ValueString(); got != "configured-name" {
		t.Fatalf("Name = %q, want configured value %q", got, "configured-name")
	}
	if got := model.NameMDNS.ValueString(); got != "configured-mdns" {
		t.Fatalf("NameMDNS = %q, want configured value %q", got, "configured-mdns")
	}
	// Left unconfigured (null): must keep whatever the API reported.
	if got := model.NameDNS.ValueString(); got != "api-returned-dns" {
		t.Fatalf("NameDNS = %q, want API-reported value %q (was left unconfigured)", got, "api-returned-dns")
	}
	// Unknown at plan time: must keep whatever the API reported.
	if got := model.NameNetBIOS.ValueString(); got != "api-returned-netbios" {
		t.Fatalf("NameNetBIOS = %q, want API-reported value %q (was unknown at plan time)", got, "api-returned-netbios")
	}
	if got := model.IP.ValueString(); got != "192.0.2.1" {
		t.Fatalf("IP = %q, want configured value %q", got, "192.0.2.1")
	}
	if got := model.Mode.ValueString(); got != freeboxTypes.LanModeRouter {
		t.Fatalf("Mode = %q, want configured value %q", got, freeboxTypes.LanModeRouter)
	}
}
