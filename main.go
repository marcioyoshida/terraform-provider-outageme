// Terraform provider for outage.me Monitor (ADR 013 M3-2, #225): uptime checks,
// alert channels, maintenance windows, domains and the status page, as code.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/marcioyoshida/terraform-provider-outageme/internal/provider"
)

var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run with support for debuggers like delve")
	flag.Parse()
	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/marcioyoshida/outageme",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
