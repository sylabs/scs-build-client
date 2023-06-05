// Copyright (c) 2022-2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	build "github.com/sylabs/scs-build-client/client"
)

// buildArtifact sends a build request for the specified arch, optionally publishing it to
// libraryRef. Output is streamed to standard output. If the build cannot be submitted, or does not
// succeed, an error is returned.
func (app *App) buildArtifact(ctx context.Context, arch string, def []byte, buildContext string, libraryRef string) (*build.BuildInfo, error) {
	opts := []build.BuildOption{build.OptBuildArchitecture(arch), build.OptBuildContext(buildContext)}
	if libraryRef != "" {
		opts = append(opts, build.OptBuildLibraryRef(libraryRef))
	}

	bi, err := app.buildClient.Submit(ctx, bytes.NewReader(def), opts...)
	if err != nil {
		return nil, fmt.Errorf("error submitting remote build: %w", err)
	}
	if err := app.buildClient.GetOutput(ctx, bi.ID(), os.Stdout); err != nil {
		return nil, fmt.Errorf("error streaming remote build output: %w", err)
	}
	if bi, err = app.buildClient.GetStatus(ctx, bi.ID()); err != nil {
		return nil, fmt.Errorf("error getting remote build status: %w", err)
	}

	// The returned info doesn't indicate an exit code, but a zero-sized image tells us something
	// went wrong.
	if bi.ImageSize() <= 0 {
		return nil, errors.New("failed to build image")
	}

	if buildContext != "" {
		_ = app.buildClient.DeleteBuildContext(ctx, buildContext)
	}

	return bi, nil
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

func (app *App) getBuildDef(uri string) ([]byte, error) {
	// Build spec could be a URI, or the path to a definition file.
	if b, ok := definitionFromURI(uri); ok {
		return b, nil
	}

	// Attempt to read app.buildSpec as a file
	return os.ReadFile(uri)
}
