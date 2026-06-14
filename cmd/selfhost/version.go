package main

import (
	"flag"
	"fmt"
	"os"
)

// version is the build version of the selfhost binary. It is overridden at
// build time via -ldflags "-X main.version=<value>" (see the selfhost
// Dockerfile and the Build Selfhost workflow). It defaults to "dev" for local
// builds.
var version = "dev"

// handleVersionFlag prints the version and exits when -version/--version is
// passed. It runs before any other startup work so `cairn-selfhost --version`
// is cheap and free of side effects.
func handleVersionFlag() {
	showVersion := flag.Bool("version", false, "print the cairn-selfhost version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}
}
