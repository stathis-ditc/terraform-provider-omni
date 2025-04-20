// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package main is the entrypoint for the provider server.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/stathis-ditc/terraform-provider-omni/internal/omni"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/ditc/siderolabs-omni",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), omni.New(), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
