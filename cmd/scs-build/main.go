// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"fmt"
	"os"

	"github.com/sylabs/scs-build-client/cmd/scs-build/cmd"
)

var (
	version = "unknown"
	date    = ""
	builtBy = ""
	commit  = ""
	state   = ""
)

func main() {
	if err := cmd.Execute(version, date, builtBy, commit, state); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
