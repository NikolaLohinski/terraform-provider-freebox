//go:build tools
// +build tools

package tools

import (
	_ "github.com/hashicorp/terraform"
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
	_ "github.com/onsi/ginkgo/v2/ginkgo"
	_ "github.com/x-izumin/gex/cmd/gex"
)
