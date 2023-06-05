// Copyright (c) 2022-2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

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

func Test_definitionFromURI(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantDef string
		wantOK  bool
	}{
		{
			name:   "NonURI",
			raw:    "file.txt",
			wantOK: false,
		},
		{
			name:    "Docker",
			raw:     "docker://alpine",
			wantDef: "bootstrap: docker\nfrom: alpine\n",
			wantOK:  true,
		},
		{
			name:    "Library",
			raw:     "library://alpine",
			wantDef: "bootstrap: library\nfrom: alpine\n",
			wantOK:  true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			def, ok := definitionFromURI(tt.raw)

			if got, want := string(def), tt.wantDef; got != want {
				t.Errorf("got def %#v, want %#v", got, want)
			}

			if got, want := ok, tt.wantOK; got != want {
				t.Errorf("got OK %v, want %v", got, want)
			}
		})
	}
}
