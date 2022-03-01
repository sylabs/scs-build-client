// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import "strings"

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
