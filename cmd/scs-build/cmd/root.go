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
