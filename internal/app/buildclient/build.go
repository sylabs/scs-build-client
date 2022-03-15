// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"bytes"
	"context"
	"fmt"
	"os"

	build "github.com/sylabs/scs-build-client/client"
)

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
	return bi, nil
}
