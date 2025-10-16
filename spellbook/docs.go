package spellbook

import (
	"context"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Docs mg.Namespace

// Render the provider documentation
func (Docs) Build(ctx context.Context) error {
	if err := sh.Run("terraform", "fmt", "-recursive", "."); err != nil {
		return err
	}

	if err := sh.Run("tfplugindocs", "generate"); err != nil {
		return err
	}

	return nil
}
