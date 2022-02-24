// Copyright (c) 2022, Sylabs, Inc. All rights reserved.

package buildclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitLibraryRef(t *testing.T) {
	tests := []struct {
		name         string
		libraryRef   string
		expectedPath string
		expectedTag  string
	}{
		{"Simple", "library://entity/collection/container:tag", "entity/collection/container", "tag"},
		{"MissingPrefix", "entity/collection/container:tag", "entity/collection/container", "tag"},
		{"MissingTag", "entity/collection/container", "entity/collection/container", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, tag := splitLibraryRef(tt.libraryRef)

			if assert.Equal(t, tt.expectedPath, path) {
				assert.Equal(t, tt.expectedTag, tag)
			}
		})
	}
}
