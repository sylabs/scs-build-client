// Copyright (c) 2022, Sylabs Inc. All rights reserved.
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
}

const defaultFrontendURL = "https://cloud.sylabs.io"

// New creates new application instance
func New(ctx context.Context, cfg *Config) (*App, error) {
	app := &App{
		buildSpec: cfg.DefFileName,
		force:     cfg.Force,
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
	if app.buildClient, app.libraryClient, err = getClients(ctx, cfg.SkipTLSVerify, feURL, cfg.AuthToken, cfg.UserAgent); err != nil {
		return nil, fmt.Errorf("error initializing client(s): %w", err)
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

// getClients returns initialized clients for remote build and cloud library
func getClients(ctx context.Context, skipVerify bool, endpoint, authToken, userAgent string) (*build.Client, *library.Client, error) {
	feCfg, err := endpoints.GetFrontendConfig(ctx, skipVerify, endpoint)
	if err != nil {
		return nil, nil, err
	}

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: skipVerify}

	buildClient, err := build.NewClient(
		build.OptBaseURL(feCfg.BuildAPI.URI),
		build.OptBearerToken(authToken),
		build.OptUserAgent(userAgent),
		build.OptHTTPClient(&http.Client{Transport: tr}),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("error initializing build client: %w", err)
	}

	libraryClient, err := library.NewClient(&library.Config{
		BaseURL:    feCfg.LibraryAPI.URI,
		AuthToken:  authToken,
		HTTPClient: &http.Client{Transport: tr},
		UserAgent:  userAgent,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error initializing library client: %w", err)
	}

	return buildClient, libraryClient, nil
}

// definitionFromURI attempts to parse a URI from raw. If raw contains a URI, a definition file
// representing it is returned, and ok is set to true. Otherwise, ok is set to false.
func definitionFromURI(raw string) (def []byte, ok bool) {
	var u []string
	if strings.Contains(raw, "://") {
		u = strings.SplitN(raw, "://", 2)
	} else if strings.Contains(raw, ":") {
		u = strings.SplitN(raw, ":", 2)
	} else {
		return nil, false
	}

	var b bytes.Buffer

	fmt.Fprintln(&b, "bootstrap:", u[0])
	fmt.Fprintln(&b, "from:", u[1])

	return b.Bytes(), true
}

// Run is the main application entrypoint
func (app *App) Run(ctx context.Context, arch string) error {
	var libraryRef string
	var artifactFileName string

	if app.dstFileName != "" {
		// Ensure destination file doesn't already exist (or --force is specified)
		if _, err := os.Stat(app.dstFileName); !os.IsNotExist(err) && !app.force {
			return fmt.Errorf("file %v already exists", app.dstFileName)
		}
		artifactFileName = app.dstFileName
	} else if app.LibraryRef != nil {
		libraryRef = app.LibraryRef.String()
	}

	var def []byte

	// Build spec could be a URI, or the path to a definition file.
	if b, ok := definitionFromURI(app.buildSpec); ok {
		def = b
	} else {
		b, err := os.ReadFile(app.buildSpec)
		if err != nil {
			return fmt.Errorf("error reading def file %v: %w", app.buildSpec, err)
		}
		def = b
	}

	// send build request
	bi, err := app.buildArtifact(ctx, def, arch, libraryRef)
	if err != nil {
		return fmt.Errorf("error building artifact: %w", err)
	}

	if artifactFileName == "" {
		if libraryRef == "" {
			fmt.Printf("Build artifact %v is available for 24 hours or less\n", bi.LibraryRef())
		}
		return nil
	}

	if err := app.retrieveArtifact(ctx, bi, artifactFileName, arch); err != nil {
		return fmt.Errorf("error retrieving build artifact: %w", err)
	}
	return nil
}
