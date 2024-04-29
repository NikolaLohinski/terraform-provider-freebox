package spellbook

import (
	"context"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Book mg.Namespace

// Render the mdBook
func (Book) Build(ctx context.Context) error {
	return sh.Run("mdbook", "build", "book")
}

// Watch and serve the mdBook
func (Book) Serve(ctx context.Context) error {
	return sh.Run("mdbook", "serve", "book")
}
