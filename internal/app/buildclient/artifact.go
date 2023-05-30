// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	build "github.com/sylabs/scs-build-client/client"
)

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
