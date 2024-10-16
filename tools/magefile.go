//go:build mage

package main

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/magefile/mage/sh"
)

const (
	mdbookLinuxURL   = "https://github.com/rust-lang/mdBook/releases/download/v0.4.37/mdbook-v0.4.37-x86_64-unknown-linux-gnu.tar.gz"
	mdbookMacOSURL   = "https://github.com/rust-lang/mdBook/releases/download/v0.4.40/mdbook-v0.4.40-x86_64-apple-darwin.tar.gz"
)

var Default = Build

func Build() error {
	if err := sh.RunV("go", "mod", "tidy"); err != nil {
		return err
	}
	if err := sh.RunV("go", "run", "-mod=mod", "github.com/izumin5210/gex/cmd/gex", "--build"); err != nil {
		return err
	}
	here, err := os.Getwd()
	if err != nil {
		return err
	}

	mdbookURL := mdbookLinuxURL

	if runtime.GOOS == "darwin" {
		mdbookURL = mdbookMacOSURL
	}

	response, err := http.Get(mdbookURL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	gzr, err := gzip.NewReader(response.Body)
	if err != nil {
		return err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)

LOOP:
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			break LOOP
		case err != nil:
			return err
		case header == nil:
			continue
		}
		target := filepath.Join(here, "bin", header.Name)

		switch header.Typeflag {
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
			f.Close()
		}
	}
	return nil
}
