// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "scs-build",
	Short: "Sylabs Cloud Build Client",
}

func Execute(version string) error {
	// Add version subcommand
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print scs-build version",
		Run: func(cmd *cobra.Command, args []string) {
			product := "scs-build"
			verStr := product
			if version == "" {
				verStr += " <unknown version>"
			} else {
				verStr += " " + version
			}
			fmt.Fprintln(os.Stderr, verStr)
		},
	})

	// Add build subcommand
	addBuildCommand(rootCmd)

	return rootCmd.Execute()
}
