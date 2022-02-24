// Copyright (c) 2022, Sylabs, Inc. All rights reserved.

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
