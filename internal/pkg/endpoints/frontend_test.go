// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package endpoints

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getFrontendConfigURL(t *testing.T) {
	const baseURL = "https://thisisaurl:1234/stuffgoeshere"

	tests := []struct {
		name        string
		baseURL     string
		expectedURL string
	}{
		{"Simple", "https://host.DOMAIN", "https://host.DOMAIN" + "/" + frontendConfigPath},
		{"WithTrailingSlash", baseURL + "/", baseURL + "/" + frontendConfigPath},
		{"FullyQualfiied", baseURL, baseURL + "/" + frontendConfigPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, getFrontendConfigURL(tt.baseURL), tt.expectedURL)
		})
	}
}

func TestGetFrontendConfig(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name               string
		cfg                *FrontendConfig
		expectedLibraryURI string
		expectedBuildURI   string
		expectedErr        error
	}{
		{
			"Simple",
			&FrontendConfig{
				LibraryAPI: uri{URI: "https://library.sylabs.io"},
				BuildAPI:   uri{URI: "https://build.sylabs.io"},
			},
			"https://library.sylabs.io",
			"https://build.sylabs.io",
			nil,
		},
		{
			"Misconfigured",
			&FrontendConfig{},
			"https://library.sylabs.io",
			"https://build.sylabs.io",
			errServerMisconfigured,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewEncoder(w).Encode(tt.cfg); err != nil {
					t.Fatalf("json encoding error: %v", err)
				}
			}))
			defer ts.Close()

			result, err := GetFrontendConfig(ctx, false, ts.URL)
			if tt.expectedErr == nil && assert.NoError(t, err) {
				assert.Equal(t, result.LibraryAPI.URI, tt.expectedLibraryURI)
				assert.Equal(t, result.BuildAPI.URI, tt.expectedBuildURI)
			}
			if tt.expectedErr != nil {
				assert.Nil(t, result)
				assert.ErrorIs(t, err, tt.expectedErr)
			}
		})
	}
}
