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
