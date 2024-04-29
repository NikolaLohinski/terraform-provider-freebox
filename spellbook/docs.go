package spellbook

import (
	"context"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Docs mg.Namespace

// Render the provider documentation
func (Docs) Build(ctx context.Context) error {
	return sh.Run("tfplugindocs", "generate")
}
