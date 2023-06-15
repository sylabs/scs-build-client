// Copyright (c) 2022-2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"testing"
)

func TestValidateBuildSpec(t *testing.T) {
	tests := []struct {
		name        string
		buildSpec   string
		expectError bool
	}{
		{"DockerBuild", "docker://alpine:3", false},
		{"MalformedButValid", "docke//alpine:3", false},
		{"MalformedAgainButValidFilename", "docker:alpine:3", false},
		{"File", "alpine_3.def", false},
		{"FileScheme", "file://alpine_3.def", false},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			_, err := parseBuildSpec(tt.buildSpec)
			if (err != nil) != tt.expectError {
				t.Fatal(err)
			}
		})
	}
}
