// Copyright (c) 2022, Sylabs, Inc. All rights reserved.

package buildclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"

	build "github.com/sylabs/scs-build-client/client"
	"github.com/sylabs/scs-build-client/internal/pkg/endpoints"
	library "github.com/sylabs/scs-library-client/client"
	"github.com/sylabs/singularity/pkg/build/types"
	"github.com/sylabs/singularity/pkg/build/types/parser"
)

// DefaultBuildArch is defined as amd64
const DefaultBuildArch = "amd64"

// Config contains set up for application
type Config struct {
	URL              string
	AuthToken        string
	DefFileName      string
	ArtifactFileName string
	Arch             string
	SkipTLSVerify    bool
	ImageSpec        string
	Force            bool
}

// App represents the application instance
type App struct {
	httpClient       *http.Client
	buildClient      *build.Client
	libraryClient    *library.Client
	arch             string
	buildSpec        string
	artifactFileName string
	imageSpec        *url.URL
	force            bool
}

// New creates new application instance
func New(ctx context.Context, cfg *Config) (*App, error) {
	app := &App{
		arch:             cfg.Arch,
		buildSpec:        cfg.DefFileName,
		force:            cfg.Force,
		artifactFileName: cfg.ArtifactFileName,
	}

	u, err := url.Parse(cfg.ImageSpec)
	if err != nil {
		return nil, fmt.Errorf("error parsing image spec: %w", err)
	}
	app.imageSpec = u

	if app.arch == "" {
		app.arch = DefaultBuildArch
	}

	app.httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.SkipTLSVerify,
			},
		},
	}

	buildClient, libraryClient, err := getClients(ctx, app.httpClient, cfg.URL, cfg.AuthToken)
	if err != nil {
		return nil, fmt.Errorf("error initializing client(s): %w", err)
	}
	app.buildClient = buildClient
	app.libraryClient = libraryClient

	return app, nil
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
func (app *App) Run(ctx context.Context) error {
	if _, err := os.Stat(app.artifactFileName); !os.IsNotExist(err) {
		if !app.force {
			return fmt.Errorf("file %v already exists", app.artifactFileName)
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
	bi, err := app.buildArtifact(ctx, def, app.arch, app.imageSpec)
	if err != nil {
		return fmt.Errorf("error building artifact: %w", err)
	}

	if app.artifactFileName == "" {
		if app.imageSpec.Scheme != requestTypeLibrary {
			fmt.Printf("Build artifact %v is available for 24 hours or less\n", bi.LibraryRef)
		}
		return nil
	}

	if err := app.retrieveArtifact(ctx, bi, app.artifactFileName, app.arch); err != nil {
		return fmt.Errorf("error retrieving build artifact: %w", err)
	}
	return nil
}
