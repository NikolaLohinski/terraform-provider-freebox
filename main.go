package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nikolalohinski/terraform-provider-freebox/internal"
)

const (
	authorizePositionalArgument = "authorize"
)

var (
	// This will be set by the goreleaser configuration to appropriate values for the compiled binary.
	version string = "dev"
)

func main() {
	var (
		debug bool
		err   error
	)

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	if flag.Arg(0) == authorizePositionalArgument {
		_, err = tea.NewProgram(internal.NewAuthorization(version)).Run()
	} else {
		err = providerserver.Serve(context.Background(), internal.NewProvider(version), providerserver.ServeOpts{
			Address: "registry.terraform.io/NikolaLohinski/freebox",
			Debug:   debug,
		})
	}
	if err != nil {
		log.Fatal(fmt.Errorf("an error occurred during the provider run: %s", err.Error()))
	}
}
