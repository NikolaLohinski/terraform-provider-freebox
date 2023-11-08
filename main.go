package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/nikolalohinski/terraform-provider-freebox/provider"
)

var (
	// This will be set by the goreleaser configuration to appropriate values for the compiled binary.
	version string = "dev"
)

// Run golang-ci reporting on the source code
//go:generate go run github.com/golangci/golangci-lint/cmd/golangci-lint run --fix --timeout 3m ./...
// Run the docs generation tool
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/nikolalohinski/freebox",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
