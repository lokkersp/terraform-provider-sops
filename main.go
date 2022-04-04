package main

import (
	"context"
	"flag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/lokkersp/terraform-provider-sops/sops"
	"log"
)

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := &plugin.ServeOpts{ProviderFunc: sops.Provider}
	if debugMode {
		err := plugin.Debug(context.Background(), "registry.terraform.io/lokkersp/sops", opts)
		if err != nil {
			log.Fatal(err.Error())
		}
		return
	}

	plugin.Serve(opts)
}
