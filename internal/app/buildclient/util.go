// Copyright (c) 2022-2025, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

// splitLibraryRef extracts path and tag from library reference.
//
// "library://entity/collection/container:tag" returns "entity/collection/container", "tag"
func splitLibraryRef(libraryRef string) (string, string) {
	const libraryPrefix = "library://"

	comps := strings.SplitN(strings.TrimPrefix(libraryRef, libraryPrefix), ":", 2)
	if len(comps) < 2 {
		return comps[0], ""
	}

	return comps[0], comps[1]
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

func getBuildDef(uri string) ([]byte, error) {
	// Build spec could be a URI, or the path to a definition file.
	if b, ok := definitionFromURI(uri); ok {
		return b, nil
	}

	// Attempt to read app.buildSpec as a file
	return os.ReadFile(uri)
}
