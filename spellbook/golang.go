package spellbook

import (
	"context"
	"runtime"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Go mg.Namespace

// Runs ginkgo for unit tests
func (Go) Test(ctx context.Context) error {
	return sh.Run("ginkgo", "run", "./...")
}

// Cleans dependencies and imports
func (Go) Tidy(ctx context.Context) error {
	return sh.Run("go", "mod", "tidy", "-v")
}

// Builds and opens a coverage report
func (Go) Cover(ctx context.Context) error {
	if err := sh.Run("go", "test", "-v", "-coverprofile", "coverage.txt", "./..."); err != nil {
		return err
	}
	if err := sh.Run("go", "tool", "cover", "-html", "coverage.txt", "-o", "coverage.html"); err != nil {
		return err
	}
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	default:
		cmd = "xdg-open"
	}
	args = append(args, "cover.html")

	return sh.Run(cmd, args...)
}
