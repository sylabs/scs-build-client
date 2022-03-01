// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sylabs/scs-library-client/client"
)

func TestGetFrontendURL(t *testing.T) {
	tests := []struct {
		name        string
		libraryRef  string
		overrideURL string
		expectedURL string
		expectError bool
	}{
		{"LibraryRefWithoutHostNoOverride", "library:entity/collection/container", "", defaultFrontendURL, false},
		{"LibraryRefVariationWithoutHostNoOverride", "library:///entity/collection/container", "", defaultFrontendURL, false},
		{"LibraryRefWithHostNoOverride", "library://myhost/entity/collection/container", "", "https://myhost", false},
		{"NoLibraryRefWithoutHostNoOverride", "", "", defaultFrontendURL, false},
		{"LibraryRefWithoutHostWithOverride", "library:entity/collection/container", "https://myhost", "https://myhost", false},
		{"LibraryRefWithHostAndOverride", "library://myhost/entity/collection/container", "https://notmyhost", "", true},
		{"LocalBuildSpecWithOverride", "test.sif", "https://myhost", "https://myhost", false},
		{"LocalBuildSpecWithoutOverride", "test.sif", "", defaultFrontendURL, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r *client.Ref

			if tt.libraryRef != "" && strings.HasPrefix(tt.libraryRef, client.Scheme+":") {
				var err error
				r, err = client.Parse(tt.libraryRef)
				assert.NoError(t, err)
			}

			result, err := getFrontendURL(r, tt.overrideURL)
			if !tt.expectError {
				if assert.NoError(t, err) {
					assert.Equal(t, tt.expectedURL, result)
				}
			} else {
				assert.Error(t, err)
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
