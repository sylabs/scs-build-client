// Copyright (c) 2022-2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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

func (app *App) retrieveArtifact(ctx context.Context, bi *build.BuildInfo, filename, arch string) error {
	fp, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o770)
	if err != nil {
		return fmt.Errorf("error opening file %s for writing: %w", filename, err)
	}
	defer func() {
		_ = fp.Close()
	}()

	h := sha256.New()

	w := io.MultiWriter(fp, h)

	path, tag := splitLibraryRef(bi.LibraryRef())

	if err := app.libraryClient.DownloadImage(ctx, w, arch, path, tag, nil); err != nil {
		return fmt.Errorf("error downloading image %v: %w", bi.LibraryRef(), err)
	}

	// Verify image checksum
	if values := strings.Split(bi.ImageChecksum(), "."); len(values) == 2 {
		if strings.ToLower(values[0]) == "sha256" {
			imageChecksum := hex.EncodeToString(h.Sum(nil))
			if values[1] != imageChecksum {
				fmt.Fprintf(os.Stderr, "Error: image checksum mismatch (expecting %v, got %v)\n", values[1], imageChecksum)
			} else {
				fmt.Fprintf(os.Stderr, "Image checksum verified successfully.\n")
			}
		}
	}

	return nil
}
