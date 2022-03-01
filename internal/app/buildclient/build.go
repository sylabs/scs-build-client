// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"context"
	"fmt"
	"os"

	build "github.com/sylabs/scs-build-client/client"
	"github.com/sylabs/singularity/pkg/build/types"
)

func (app *App) buildArtifact(ctx context.Context, def types.Definition, arch string, libraryRef string) (*build.BuildInfo, error) {
	bi, err := app.buildClient.Submit(ctx, build.BuildRequest{
		LibraryRef:    libraryRef,
		LibraryURL:    app.libraryClient.BaseURL.String(),
		DefinitionRaw: def.Raw,
		BuilderRequirements: map[string]string{
			"arch": arch,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error submitting remote build: %w", err)
	}
	if err := app.buildClient.GetOutput(ctx, bi.ID, os.Stdout); err != nil {
		return nil, fmt.Errorf("error streaming remote build output: %w", err)
	}
	if bi, err = app.buildClient.GetStatus(ctx, bi.ID); err != nil {
		return nil, fmt.Errorf("error getting remote build status: %w", err)
	}
	return &bi, nil
}
