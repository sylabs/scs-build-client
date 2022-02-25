// Copyright (c) 2022, Sylabs, Inc. All rights reserved.

package buildclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	build "github.com/sylabs/scs-build-client/client"
	"github.com/sylabs/scs-build-client/internal/pkg/endpoints"
	library "github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/build/types/parser"
)

// Config contains set up for application
type Config struct {
	URL           string
	AuthToken     string
	DefFileName   string
	SkipTLSVerify bool
	LibraryRef    string
	Force         bool
}

// App represents the application instance
type App struct {
	httpClient    *http.Client
	buildClient   *build.Client
	libraryClient *library.Client
	buildSpec     string
	LibraryRef    *url.URL
	force         bool
}

// New creates new application instance
func New(ctx context.Context, cfg *Config) (*App, error) {
	// Parse/validate image spec (local file or library ref)
	libraryRef, err := url.Parse(cfg.LibraryRef)
	if err != nil {
		return nil, fmt.Errorf("error parsing image spec: %w", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.SkipTLSVerify,
			},
		},
	}

	// Initialize build & library clients
	buildClient, libraryClient, err := getClients(ctx, httpClient, cfg.URL, cfg.AuthToken)
	if err != nil {
		return nil, fmt.Errorf("error initializing client(s): %w", err)
	}

	return &App{
		buildSpec:     cfg.DefFileName,
		force:         cfg.Force,
		LibraryRef:    libraryRef,
		httpClient:    httpClient,
		buildClient:   buildClient,
		libraryClient: libraryClient,
	}, nil
}

// getClients returns initialized clients for remote build and cloud library
func getClients(ctx context.Context, httpClient *http.Client, endpoint, authToken string) (*build.Client, *library.Client, error) {
	feCfg, err := endpoints.GetFrontendConfig(ctx, httpClient, endpoint)
	if err != nil {
		return nil, nil, err
	}

	// Initialize scs-build-client
	buildClient, err := build.New(&build.Config{
		BaseURL:    feCfg.BuildAPI.URI,
		AuthToken:  authToken,
		HTTPClient: httpClient,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error initializing build client: %w", err)
	}

	libraryClient, err := library.NewClient(&library.Config{
		BaseURL:    feCfg.LibraryAPI.URI,
		AuthToken:  authToken,
		HTTPClient: httpClient,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error initializing library client: %w", err)
	}

	return buildClient, libraryClient, nil
}

// Run is the main application entrypoint
func (app *App) Run(ctx context.Context, arch string) error {
	var artifactFileName string

	if app.LibraryRef.Scheme == "file" || app.LibraryRef.Scheme == "" {
		artifactFileName = app.LibraryRef.Path
		if _, err := os.Stat(artifactFileName); !os.IsNotExist(err) {
			if !app.force {
				return fmt.Errorf("file %v already exists", artifactFileName)
			}
		}
	}

	// Parse build spec (for example, either "docker://..." or filename)
	var def types.Definition
	var err error

	// Attempt to process build spec as URI
	def, err = types.NewDefinitionFromURI(app.buildSpec)
	if err != nil {
		// Attempt to process build spec as filename
		isValid, err := parser.IsValidDefinition(app.buildSpec)
		if err != nil {
			return fmt.Errorf("error validating def file: %w", err)
		}
		if !isValid {
			return fmt.Errorf("invalid def file %v", app.buildSpec)
		}

		// read build definition into buffer
		fp, err := os.Open(app.buildSpec)
		if err != nil {
			return fmt.Errorf("error reading def file %v: %w", app.buildSpec, err)
		}
		defer func() {
			_ = fp.Close()
		}()

		if def, err = parser.ParseDefinitionFile(fp); err != nil {
			return fmt.Errorf("error parsing definition file %v: %w", app.buildSpec, err)
		}
	}

	// send build request
	var libraryRef string
	if strings.HasPrefix(app.LibraryRef.String(), "library://") {
		libraryRef = app.LibraryRef.String()
	}
	bi, err := app.buildArtifact(ctx, def, arch, libraryRef)
	if err != nil {
		return fmt.Errorf("error building artifact: %w", err)
	}

	if artifactFileName == "" {
		if app.LibraryRef.Scheme != requestTypeLibrary {
			fmt.Printf("Build artifact %v is available for 24 hours or less\n", bi.LibraryRef)
		}
		return nil
	}

	if err := app.retrieveArtifact(ctx, bi, artifactFileName, arch); err != nil {
		return fmt.Errorf("error retrieving build artifact: %w", err)
	}
	return nil
}
