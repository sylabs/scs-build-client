// Copyright (c) 2022, Sylabs, Inc. All rights reserved.

package buildclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const validBuildDef = `Bootstrap: docker
From: alpine
`
const invalidBuildDef = "invalid build definition"

var validResponse = []byte(`{
    "data": {
        "appOrder": [],
        "buildData": {
            "buildScripts": {
                "post": {
                    "args": "",
                    "script": ""
                },
                "pre": {
                    "args": "",
                    "script": ""
                },
                "setup": {
                    "args": "",
                    "script": ""
                },
                "test": {
                    "args": "",
                    "script": ""
                }
            },
            "files": []
        },
        "customData": null,
        "header": {
            "bootstrap": "docker",
            "from": "alpine"
        },
        "imageData": {
            "imageScripts": {
                "environment": {
                    "args": "",
                    "script": ""
                },
                "help": {
                    "args": "",
                    "script": ""
                },
                "runScript": {
                    "args": "",
                    "script": ""
                },
                "startScript": {
                    "args": "",
                    "script": ""
                },
                "test": {
                    "args": "",
                    "script": ""
                }
            },
            "labels": {},
            "metadata": null
        },
        "raw": "Qm9vdHN0cmFwOiBkb2NrZXIKRnJvbTogYWxwaW5lCgo="
    }
}
`)

var malformedResponse = []byte("malformed build response")

// TestValidateBuildDef tests the function 'App.validateBuildDef()'. It is not intended to
// test remote build server response, only that it's being called correctly with the correct
// payload.
func TestValidateBuildDef(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		buildDef string

		// this flag allows simulation of successful validation w/invalid response
		// set to 'true' if 'buildDef' defines valid build definition
		isValid bool

		response           []byte
		expectedStatusCode int
	}{
		{"Success", validBuildDef, true, validResponse, http.StatusOK},
		{"SuccessWithInvalidResponse", validBuildDef, true, malformedResponse, http.StatusBadRequest},
		{"FailedValidation", invalidBuildDef, false, nil, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// tsBuildServer provides mock for remote build server `/v1/convert-def-file` endpoint
			tsBuildServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/convert-def-file" {
					t.Fatalf("Request made to invalid endpoint: %v", r.URL.String())
				}

				if _, err := io.ReadAll(r.Body); assert.NoError(t, err, "error reading POST payload") {
					if !tt.isValid {
						w.WriteHeader(tt.expectedStatusCode)
					} else {
						// if 'isValid' is 'true', return http status ok (200) regardless if 'response' data is valid or not.
						w.WriteHeader(http.StatusOK)
					}

					if tt.response != nil {
						_, err := w.Write(tt.response)
						assert.NoError(t, err, "error writing response")
					}
				}
			}))
			defer tsBuildServer.Close()

			// tsFrontend provides mock for '/assets/config/config.prod.json' endpoint and returns
			// a minimal subset of the valid response. The values returned by this http server is
			// dependent on the URL from 'tsBuildServer'.
			tsFrontend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Return valid canned `config.json` output
				w.WriteHeader(http.StatusOK)

				w.Header().Add("Content-Type", "application/json")

				_, err := w.Write([]byte(fmt.Sprintf("{\"libraryAPI\": {\"uri\": \"%s\"}, \"builderAPI\": {\"uri\": \"%s\"}}", "http://localhost:1234", tsBuildServer.URL)))
				assert.NoError(t, err)
			}))
			defer tsFrontend.Close()

			if app, err := New(ctx, &Config{URL: tsFrontend.URL}); assert.NoError(t, err) {
				err := app.validateBuildDef(ctx, []byte(tt.buildDef))
				if tt.expectedStatusCode != http.StatusOK {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}
