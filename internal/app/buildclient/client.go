// Copyright (c) 2022-2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	build "github.com/sylabs/scs-build-client/client"
	"github.com/sylabs/scs-build-client/internal/pkg/endpoints"
	library "github.com/sylabs/scs-library-client/client"
)

// Config contains set up for application
type Config struct {
	URL           string
	AuthToken     string
	DefFileName   string
	SkipTLSVerify bool
	LibraryRef    string
	Force         bool
	UserAgent     string
}

// App represents the application instance
type App struct {
	buildClient   *build.Client
	libraryClient *library.Client
	buildSpec     string
	LibraryRef    *library.Ref
	dstFileName   string
	force         bool
	buildURL      string
	skipTLSVerify bool
}

const defaultFrontendURL = "https://cloud.sylabs.io"

// New creates new application instance
func New(ctx context.Context, cfg *Config) (*App, error) {
	app := &App{
		buildSpec:     cfg.DefFileName,
		force:         cfg.Force,
		skipTLSVerify: cfg.SkipTLSVerify,
	}
	var libraryRefHost string

	// Parse/validate image spec (local file or library ref)
	if strings.HasPrefix(cfg.LibraryRef, library.Scheme+":") {
		ref, err := library.ParseAmbiguous(cfg.LibraryRef)
		if err != nil {
			return nil, fmt.Errorf("malformed library ref: %w", err)
		}

		if ref.Host != "" {
			// Ref contains a host. Note this to determine the front end URL, but don't include it
			// in the LibraryRef, since the Build Service expects a hostless format.
			libraryRefHost = ref.Host
			ref.Host = ""
		}

		app.LibraryRef = ref
	} else if cfg.LibraryRef != "" {
		// Parse as URL
		ref, err := url.Parse(cfg.LibraryRef)
		if err != nil {
			return nil, fmt.Errorf("error parsing %v as URL: %w", cfg.LibraryRef, err)
		}
		if ref.Scheme != "file" && ref.Scheme != "" {
			return nil, fmt.Errorf("unsupported library ref scheme %v", ref.Scheme)
		}
		app.dstFileName = ref.Path
	}

	// Determine frontend URL either from library ref, if provided or url, if provided, or default.
	feURL, err := getFrontendURL(cfg.URL, libraryRefHost)
	if err != nil {
		return nil, err
	}

	// Initialize build & library clients
	feCfg, err := endpoints.GetFrontendConfig(ctx, cfg.SkipTLSVerify, feURL)
	if err != nil {
		return nil, err
	}
	app.buildURL = feCfg.BuildAPI.URI

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.SkipTLSVerify}

	app.buildClient, err = build.NewClient(
		build.OptBaseURL(feCfg.BuildAPI.URI),
		build.OptBearerToken(cfg.AuthToken),
		build.OptUserAgent(cfg.UserAgent),
		build.OptHTTPClient(&http.Client{Transport: tr}),
	)
	if err != nil {
		return nil, fmt.Errorf("error initializing build client: %w", err)
	}

	app.libraryClient, err = library.NewClient(&library.Config{
		BaseURL:    feCfg.LibraryAPI.URI,
		AuthToken:  cfg.AuthToken,
		HTTPClient: &http.Client{Transport: tr},
		UserAgent:  cfg.UserAgent,
	})
	if err != nil {
		return nil, fmt.Errorf("error initializing library client: %w", err)
	}

	return app, nil
}

// getFrontendURL determines the front end value based on urlOverride and/or libraryRefHost.
func getFrontendURL(urlOverride, libraryRefHost string) (string, error) {
	if urlOverride != "" {
		if libraryRefHost == "" {
			return urlOverride, nil
		}

		u, err := url.Parse(urlOverride)
		if err != nil {
			return "", err
		}

		if u.Host != libraryRefHost {
			return "", errors.New("conflicting arguments")
		}

		return urlOverride, nil
	}

	if libraryRefHost != "" {
		return "https://" + libraryRefHost, nil
	}

	return defaultFrontendURL, nil
}

var errNoBuildContextFiles = errors.New("no files referenced in build definition")

// uploadBuildContext parses definition file specified by 'rawDef' and uploads build context
// containing files referenced in '%files' section(s) to build server.
//
// Returns sha256 digest of uploaded build context if build context was uploaded successfully,
// otherwise returns errNoBuildContextFiles indicating no build context was uploaded/required.
func (app *App) uploadBuildContext(ctx context.Context, rawDef []byte) (string, error) {
	// Get list of files from def file '%files' section(s)
	files, err := app.getFiles(ctx, bytes.NewReader(rawDef))
	if err != nil {
		return "", fmt.Errorf("error getting build context files: %w", err)
	}
	if files == nil {
		return "", errNoBuildContextFiles
	}

	// Upload build context containing files referenced in def file to build server
	digest, err := app.buildClient.UploadBuildContext(ctx, files)
	if err != nil {
		return "", err
	}
	return digest, nil
}

// doBuild performs builds and (optionally) retrieves build artifacts.
//
// 'appendArchSuffix' toggles whether to append arch suffix to prevent
// filename collisions when building local artifacts for multiple archs.
func (app *App) doBuild(ctx context.Context, rawDef []byte, arch, digest string, appendArchSuffix bool) error {
	var libraryRef string
	var artifactFileName string

	if app.dstFileName != "" {
		// Ensure destination file doesn't already exist (or --force is specified)
		if _, err := os.Stat(app.dstFileName); !os.IsNotExist(err) && !app.force {
			return fmt.Errorf("file %v already exists", app.dstFileName)
		}

		artifactFileName = app.dstFileName
		if appendArchSuffix {
			// append arch to local file name if more than one arch is requested
			artifactFileName += "-" + arch
		}
	} else if app.LibraryRef != nil {
		libraryRef = app.LibraryRef.String()
	}

	// send build request
	bi, err := app.buildArtifact(ctx, arch, libraryRef, digest, rawDef)
	if err != nil {
		return err
	}

	// check if artifact should be downloaded (pulled) locally
	if artifactFileName == "" {
		// Build completed successfully

		// local (file) destination not specified
		if libraryRef == "" {
			// library ref not specified so build artfifact is transient
			fmt.Printf("Build artifact %v is available for 24 hours or less\n", bi.LibraryRef())
		}
		return nil
	}

	if err := app.retrieveArtifact(ctx, bi, artifactFileName, arch); err != nil {
		return fmt.Errorf("error retrieving build artifact: %w", err)
	}
	return nil
}

// Run is the main application entrypoint
func (app *App) Run(ctx context.Context, archs []string) error {
	rawDef, err := app.getBuildDef()
	if err != nil {
		return fmt.Errorf("build definition error: %w", err)
	}

	// Upload build context, as necessary
	var digest string
	digest, err = app.uploadBuildContext(ctx, rawDef)
	if err != nil && !errors.Is(err, errNoBuildContextFiles) {
		return fmt.Errorf("error uploading build context: %w", err)
	}

	if len(archs) == 1 {
		return app.doBuild(ctx, rawDef, archs[0], digest, false)
	}

	fmt.Printf("Performing builds for following architectures: %v\n", strings.Join(archs, " "))

	buildErrs := make(map[string]error)

	// Submit build request for each specified architecture
	for _, arch := range archs {
		fmt.Printf("Building for %v...\n", arch)

		if err := app.doBuild(ctx, rawDef, arch, digest, true); err != nil {
			buildErrs[arch] = err
			continue
		}
	}

	// Report any build errors
	if nBuildErrs := len(buildErrs); nBuildErrs != 0 {
		if nBuildErrs > 1 {
			fmt.Fprintf(os.Stderr, "\nBuild error(s):\n")

			for _, arch := range archs {
				if err, found := buildErrs[arch]; found {
					fmt.Fprintf(os.Stderr, "  - %v: %v\n", arch, err)
				}
			}
			fmt.Fprintln(os.Stderr)
			return errors.New("failed to build images")
		}
		for k := range buildErrs {
			if err, found := buildErrs[k]; found {
				return err
			}
		}
	}
	return nil
}
