// Copyright (c) 2022, Sylabs, Inc. All rights reserved.

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/sylabs/scs-build-client/internal/app/buildclient"
	"golang.org/x/sync/errgroup"
)

const (
	keyAccessToken      = "auth-token"
	keySkipTLSVerify    = "skip-verify"
	keyArch             = "arch"
	keyArtifactFileName = "output"
	keyFrontendURL      = "url"
	keyForceOverwrite   = "force"
	keyImageSpec        = "image-spec"
)

var Usage = func() {
	fmt.Fprintf(os.Stderr, `
Usage: buildclient [opts] <path to build definition>

    Build and push artifact to cloud library:

        buildclient --image-spec library://user/project/image:tag alpine.def

    Build, push artfifact to cloud library and download locally:

        buildclient --image-spec library://user/project/image:tag --output /tmp/alpine_3.sif alpine.def

    Build and push ephemeral artifact to cloud library:

        buildclient alpine.def

    Note: ephemeral artifacts are short-lived and are usually deleted within 24 hours.

Options:
`)
	pflag.PrintDefaults()
}

func getFlagSet() (*pflag.FlagSet, error) {
	fs := pflag.CommandLine

	fs.String(keyAccessToken, "", "Access token")
	fs.Bool(keySkipTLSVerify, false, "Skip SSL/TLS certificate verification")
	fs.String(keyArch, buildclient.DefaultBuildArch, "Requested build architecture")
	fs.String(keyFrontendURL, "https://cloud.sylabs.io", "Sylabs Cloud or Singularity Enterprise URL")
	fs.String(keyImageSpec, "", "Image spec")
	fs.Bool(keyForceOverwrite, false, "Overwrite image file if it exists")
	fs.StringP(keyArtifactFileName, "o", "", "Build artifact output file")
	return fs, fs.Parse(os.Args[1:])
}

func getConfig() (*viper.Viper, error) {
	fs, err := getFlagSet()
	if err != nil {
		return nil, err
	}
	v := viper.New()
	v.SetEnvPrefix("sylabs")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return v, v.BindPFlags(fs)
}

func run() error {
	pflag.Usage = Usage
	v, err := getConfig()
	if err != nil {
		panic(err)
	}

	if pflag.CommandLine.Arg(0) == "" {
		pflag.Usage()
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app, err := buildclient.New(ctx, &buildclient.Config{
		URL:              v.GetString(keyFrontendURL),
		AuthToken:        v.GetString(keyAccessToken),
		Arch:             v.GetString(keyArch),
		DefFileName:      pflag.CommandLine.Arg(0),
		ArtifactFileName: v.GetString(keyArtifactFileName),
		ImageSpec:        v.GetString(keyImageSpec),
		SkipTLSVerify:    v.GetBool(keySkipTLSVerify),
		Force:            v.GetBool(keyForceOverwrite),
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
		return app.Run(ctx)
	})

	if err := g.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	return err
}

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}
