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

const defaultFrontendURL = "https://cloud.sylabs.io"

// Config contains set up for application
type Config struct {
	URL           string
	AuthToken     string
	DefFileName   string
	SkipTLSVerify bool
	LibraryRef    string
	Force         bool
	UserAgent     string
	ArchsToBuild  []string
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
	archsToBuild  []string
}

type buildSpec struct {
	Def        []byte
	Context    string
	LibraryRef *library.Ref
	FileName   string
	Archs      []string
}

var errNoBuildContextFiles = errors.New("no files referenced in build definition")

// New creates new application instance
func New(ctx context.Context, cfg *Config) (*App, error) {
	app := &App{
		buildSpec:     cfg.DefFileName,
		force:         cfg.Force,
		skipTLSVerify: cfg.SkipTLSVerify,
		archsToBuild:  cfg.ArchsToBuild,
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

func appendFileSuffix(name, suffix string, appendSuffix bool) string {
	if !appendSuffix {
		return name
	}
	return fmt.Sprintf("%v-%v", name, suffix)
}

// Run is the main application entrypoint
func (app *App) Run(ctx context.Context) error {
	// Initialize build specification
	bs := buildSpec{
		LibraryRef: app.LibraryRef,
		FileName:   app.dstFileName,
		Archs:      app.archsToBuild,
	}

	if !app.force && bs.FileName != "" {
		// Check for existence of dst files
		for _, arch := range bs.Archs {
			fn := appendFileSuffix(bs.FileName, arch, len(bs.Archs) > 1)

			if _, err := os.Stat(fn); !os.IsNotExist(err) {
				return fmt.Errorf("destination file %q already exists", fn)
			}
		}
	}

	var err error
	bs.Def, err = app.getBuildDef(app.buildSpec)
	if err != nil {
		return fmt.Errorf("unable to get build definition: %w", err)
	}

	// Upload build context, as necessary
	bs.Context, err = app.uploadBuildContext(ctx, bs.Def)
	if err != nil && !errors.Is(err, errNoBuildContextFiles) {
		return fmt.Errorf("error uploading build context: %w", err)
	}

	if len(bs.Archs) > 1 {
		fmt.Printf("Performing builds for following architectures: %v\n", strings.Join(bs.Archs, " "))
	}

	return app.build(ctx, bs)
}

func (app *App) build(ctx context.Context, bs buildSpec) error {
	errs := make(map[string]error)

	for _, arch := range bs.Archs {
		fmt.Printf("Building for %v...\n", arch)

		// Submit build request
		bi, err := app.buildArtifact(ctx, arch, bs)
		if err != nil {
			errs[arch] = err
			continue
		}

		// Build completed successfully
		if app.dstFileName == "" {
			// Library ref specified; image pushed to library automatically
			if app.LibraryRef == nil {
				fmt.Printf("Build artifact %v is available for 24 hours or less\n", bi.LibraryRef())
			}
			continue
		}

		// Download file locally
		if err := app.retrieveArtifact(ctx, bi, appendFileSuffix(bs.FileName, arch, len(bs.Archs) > 1), arch); err != nil {
			errs[arch] = fmt.Errorf("error retrieving build artifact: %w", err)
			continue
		}
	}

	if len(errs) == 0 {
		// Build(s) completed successfully
		return nil
	}

	return app.reportErrs(errs)
}

// reportErrs iterates over arch/error map and outputs error(s) to console
func (app *App) reportErrs(errs map[string]error) error {
	// Report any build errors

	if len(errs) == 1 {
		// Return first (and only) error
		for _, err := range errs {
			return err
		}
	}

	fmt.Fprintf(os.Stderr, "\nBuild error(s):\n")

	for arch, err := range errs {
		fmt.Fprintf(os.Stderr, "  - %v: %v\n", arch, err)
	}

	fmt.Fprintln(os.Stderr)

	return errors.New("failed to build images")
}
