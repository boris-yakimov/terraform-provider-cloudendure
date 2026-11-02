package main

import (
	"context"
	"flag"
	"github.com/boris-yakimov/terraform-provider-cloudendure/cloudendure"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	var debugMode bool
	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := &plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return cloudendure.Provider()
		},
	}

	if debugMode {
		err := plugin.Debug(context.Background(), "hashicorp.com/edu/cloudendure", opts)
		if err != nil {
			log.Fatal(err.Error())
		}
		return
	}

	plugin.Serve(opts)
}
