// Copyright (c) 2022, Sylabs Inc. All rights reserved.
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

	build "github.com/sylabs/scs-build-client/client"
)

// buildArtifact sends a build request for the specified def and arch, optionally publishing it to
// libraryRef. Output is streamed to standard output. If the build cannot be submitted, or does not
// succeed, an error is returned.
func (app *App) buildArtifact(ctx context.Context, def []byte, arch string, libraryRef string) (*build.BuildInfo, error) {
	bi, err := app.buildClient.Submit(ctx, bytes.NewReader(def),
		build.OptBuildLibraryRef(libraryRef),
		build.OptBuildArchitecture(arch),
	)
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
		return bi, errors.New("failed to build image")
	}

	return bi, nil
}
