package main

import (
	"fmt"
	"os"
)

// version is the build version of the selfhost binary. It is overridden at
// build time via -ldflags "-X main.version=<value>" (see the selfhost
// Dockerfile and the Build Selfhost workflow). It defaults to "dev" for local
// builds.
var version = "dev"

// handleVersionFlag prints the version and exits when -version/--version is
// passed. It scans os.Args directly (rather than using the flag package) so it
// stays free of side effects and never interferes with the global flag set.
func handleVersionFlag() {
	for _, arg := range os.Args[1:] {
		if arg == "-version" || arg == "--version" {
			fmt.Println(version)
			os.Exit(0)
		}
	}
}
