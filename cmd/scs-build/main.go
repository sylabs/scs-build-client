package main

import (
	"fmt"
	"os"

	"github.com/sylabs/scs-build-client/cmd/scs-build/cmd"
)

var version = ""

func main() {
	cmd.InitApp(version)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
