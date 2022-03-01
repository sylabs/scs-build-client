// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sylabs/scs-build-client/internal/app/buildclient"
	"golang.org/x/sync/errgroup"
)

const (
	keyAccessToken    = "auth-token"
	keySkipTLSVerify  = "skip-verify"
	keyArch           = "arch"
	keyFrontendURL    = "url"
	keyForceOverwrite = "force"
)

var buildCmd = &cobra.Command{
	Use:   "build [flags] <build spec> <image path>",
	Short: "Perform remote build on Sylabs Cloud (https://cloud.sylabs.io) or Singularity Enterprise",
	Args:  cobra.MinimumNArgs(1),
	RunE:  executeBuildCmd,
	Example: `
  Build and push artifact to cloud library:

      scs-build build alpine.def library:user/project/image:tag

  Build and push artifact to Singularity Enterprise:

      scs-build build alpine.def library://cloud.enterprise.local/user/project/image:tag

  Build local artifact:

      scs-build build docker://alpine alpine_latest.sif

  Build local artifact on Singularity Enterprise:

      scs-build build --url https://cloud.enterprise.local --skip-verify docker://alpine alpine_latest.sif

  Build ephemeral artifact:

      scs-build build alpine.def

  Note: ephemeral artifacts are short-lived and are usually deleted within 24 hours.`,
}

func addBuildCommand(rootCmd *cobra.Command) {
	buildCmd.Flags().String(keyAccessToken, "", "Access token")
	buildCmd.Flags().Bool(keySkipTLSVerify, false, "Skip SSL/TLS certificate verification")
	buildCmd.Flags().StringSlice(keyArch, []string{runtime.GOARCH}, "Requested build architecture")
	buildCmd.Flags().String(keyFrontendURL, "", "Sylabs Cloud or Singularity Enterprise URL")
	buildCmd.Flags().Bool(keyForceOverwrite, false, "Overwrite image file if it exists")
	rootCmd.AddCommand(buildCmd)
}

func getConfig(cmd *cobra.Command) (*viper.Viper, error) {
	v := viper.New()
	v.SetEnvPrefix("sylabs")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return v, v.BindPFlags(cmd.Flags())
}

func executeBuildCmd(cmd *cobra.Command, args []string) error {
	defSpec := args[0]

	var libraryRef string
	if len(args) > 1 {
		libraryRef = args[1]
	}

	// Get command-line/envvars
	v, err := getConfig(cmd)
	if err != nil {
		return fmt.Errorf("error getting config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app, err := buildclient.New(ctx, &buildclient.Config{
		URL:           v.GetString(keyFrontendURL),
		AuthToken:     v.GetString(keyAccessToken),
		DefFileName:   defSpec,
		LibraryRef:    libraryRef,
		SkipTLSVerify: v.GetBool(keySkipTLSVerify),
		Force:         v.GetBool(keyForceOverwrite),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Application init error: %v\n", err)
		return fmt.Errorf("application init error: %w", err)
	}

	// set up signal handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Fprintf(os.Stderr, "Shutting down due to signal: %v\n", <-c)
		cancel()
	}()

	// run application
	g := new(errgroup.Group)

	g.Go(func() error {
		archsToBuild := v.GetStringSlice(keyArch)

		if len(archsToBuild) > 1 {
			fmt.Printf("Performing build for following architectures: %v\n", strings.Join(archsToBuild, " "))
		}

		var buildErr error

		for _, arch := range v.GetStringSlice(keyArch) {
			fmt.Printf("Building for %v...\n", arch)

			if err := app.Run(ctx, arch); err != nil {
				fmt.Fprintf(os.Stderr, "Build %v failed: %v\n", arch, err)
				if buildErr != nil {
					buildErr = err
				}
			}
		}

		if buildErr != nil {
			return buildErr
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	return err
}
