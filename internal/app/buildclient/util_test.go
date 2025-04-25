// Copyright (c) 2022-2025, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"os"
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
		{
			name:    "test",
			raw:     "library:",
			wantDef: "bootstrap: library\nfrom: \n",
			wantOK:  true,
		},
	}
	for _, tt := range tests {
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

func Test_getBuildDef(t *testing.T) {
	tests := []struct {
		name        string
		useTempFile bool
		fileName    string
		want        string
		expectError bool
	}{
		{"basic", false, "docker://alpine:3", "bootstrap: docker\nfrom: alpine:3\n", false},
		{"basicError", false, "\n", "", true},
		{"tempFile", true, "/tempfile", "bootstrap: docker\nfrom: alpine:3\n", false},
		{"tempFileError", true, "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string

			if tt.useTempFile {
				result = t.TempDir() + tt.fileName
				if tt.fileName != "" {
					fp, err := os.OpenFile(result, os.O_CREATE|os.O_WRONLY, 0o0644)
					if err != nil {
						t.Fatalf("%v", err)
					}

					if _, err := fp.Write([]byte(tt.want)); err != nil {
						t.Fatalf("%v", err)
					}

					defer fp.Close()
				}
			} else {
				result = tt.fileName
			}

			got, err := getBuildDef(result)
			if (err != nil) != tt.expectError {
				t.Fatalf("Unexpected error: %v", err)
			}

			if want := tt.want; string(got) != want {
				t.Fatalf("got: %v, want: %v", string(got), want)
			}
		})
	}
}
