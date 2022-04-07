// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFrontendURL(t *testing.T) {
	tests := []struct {
		name           string
		overrideURL    string
		libraryRefHost string
		expectedURL    string
		expectError    bool
	}{
		{
			name:        "WithoutOverride",
			expectedURL: defaultFrontendURL,
		},
		{
			name:        "WithOverride",
			overrideURL: "https://myhost",
			expectedURL: "https://myhost",
		},
		{
			name:           "HostWithoutOverride",
			libraryRefHost: "myhost",
			expectedURL:    "https://myhost",
		},
		{
			name:           "HostWithOverride",
			overrideURL:    "https://myhost",
			libraryRefHost: "myhost",
			expectedURL:    "https://myhost",
		},
		{
			name:           "HostWithConflictingOverride",
			overrideURL:    "https://myotherhost",
			libraryRefHost: "myhost",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getFrontendURL(tt.overrideURL, tt.libraryRefHost)
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
