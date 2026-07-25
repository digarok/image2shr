// Command image2shr converts modern images to Apple IIgs Super Hi-Res.
// All logic lives in internal/cli; main only wires the process streams.
package main

import (
	"os"

	"github.com/digarok/image2shr/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr, os.Stdin))
}
