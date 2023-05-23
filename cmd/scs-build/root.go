// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"fmt"
	"io"
	"runtime"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "scs-build",
	Short:         "Singularity Container Services Build Client",
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Build metadata set by linker.
var (
	version = "unknown"
	date    = ""
	builtBy = ""
	commit  = ""
)

func writeVersion(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer tw.Flush()

	fmt.Fprintf(tw, "Version:\t%v\n", version)

	if builtBy != "" {
		fmt.Fprintf(tw, "By:\t%v\n", builtBy)
	}

	if commit != "" {
		fmt.Fprintf(tw, "Commit:\t%v\n", commit)
	}

	if date != "" {
		fmt.Fprintf(tw, "Date:\t%v\n", date)
	}

	fmt.Fprintf(tw, "Runtime:\t%v (%v/%v)\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

func execute() error {
	// Add version subcommand
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Long:  "Display binary version, and build info.",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			writeVersion(cmd.OutOrStdout())
		},
	})

	// Add build subcommand
	addBuildCommand(rootCmd)

	return rootCmd.Execute()
}
