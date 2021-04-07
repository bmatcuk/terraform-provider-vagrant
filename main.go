package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

var (
	version string = "dev"
)

//go:generate terraform fmt -recursive ./examples/
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs
func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "support for debuggers like delve")
	flag.Parse()

	opts := &plugin.ServeOpts{ProviderFunc: NewVagrantProvider(version)}

	if debug {
		err := plugin.Debug(context.Background(), "registry.terraform.io/bmatcuk/vagrant", opts)
		if err != nil {
			log.Fatal(err.Error())
		}
		return
	}

	plugin.Serve(opts)
}
