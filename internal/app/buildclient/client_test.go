// Copyright (c) 2022-2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	jsonresp "github.com/sylabs/json-resp"
	"github.com/sylabs/scs-build-client/internal/pkg/endpoints"
)

const (
	defaultFEHost   = "cloud.sylabs.io"
	defaultBuildURI = "https://build.sylabs.io"
	testLibraryURI  = "http://library.domain"
	testBuildURI    = "http://build.domain"
)

func newTestFEServer(t *testing.T) *httptest.Server {
	t.Helper()

	router := http.NewServeMux()
	router.Handle("/assets/config/config.prod.json", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := json.NewEncoder(w).Encode(&endpoints.FrontendConfig{
			LibraryAPI: endpoints.URI{URI: testLibraryURI},
			BuildAPI:   endpoints.URI{URI: testBuildURI},
		}); err != nil {
			t.Fatalf("Error writing JSON encoded response: %v", err)
		}
	}))

	return httptest.NewTLSServer(router)
}

func TestNew(t *testing.T) {
	const dockerBuildSpec = "docker://alpine:3"
	const localSIFFileName = "alpine_3.sif"
	const defFile = "alpine.def"

	const libraryPath = "entity/collection/container"
	const tag = "tag"

	testLibraryRef := fmt.Sprintf("library:///%v:%v", libraryPath, tag)
	_ = testLibraryRef

	testFeSrv := newTestFEServer(t)
	defer testFeSrv.Close()

	testLibraryRefWithHost := fmt.Sprintf("library://%v/%v:%v", defaultFEHost, libraryPath, tag)
	_ = testLibraryRefWithHost

	for _, tt := range []struct {
		name        string
		buildSpec   string
		libraryRef  string
		uriOverride string
		expectError bool
	}{
		{"DstLocalFile", dockerBuildSpec, localSIFFileName, "", false},
		{"ReversedArgsLocalFile", localSIFFileName, dockerBuildSpec, "", true},
		{"LibraryRef", dockerBuildSpec, testLibraryRef, "", false},
		{"LibraryRefWithHost", dockerBuildSpec, testLibraryRefWithHost, "", false},
		{"LocalSpec", defFile, localSIFFileName, "", false},
		{"LocalSpecLibraryRef", defFile, testLibraryRef, "", false},
		{"LibraryRefWithUser", defFile, "library://user@host/" + libraryPath, "", true},
		{"LibraryRefMismatch", defFile, testLibraryRefWithHost, testFeSrv.URL, true},
		{"LibraryRefInvalidScheme", defFile, "filex://" + libraryPath, "", true},
		{"MalformedLibraryRef", defFile, "other://user@@@```@@////" + libraryPath, "", true},
		{"InvalidFrontendURI", defFile, testLibraryRef, "http://localhost:88888", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				BuildSpec:  tt.buildSpec,
				LibraryRef: tt.libraryRef,
			}

			if tt.uriOverride != "" {
				cfg.URL = tt.uriOverride
			}

			// Initialize build client.
			b, err := New(context.Background(), cfg)

			// Check if error was expected...
			if (err != nil) != tt.expectError {
				t.Fatalf("Unexpected error: %v", err)
			}
			if err != nil {
				// Received expected error.
				return
			}

			if tt.buildSpec == dockerBuildSpec {
				assert.Equal(t, dockerBuildSpec, b.buildSpec)
			}
			if tt.libraryRef == localSIFFileName {
				assert.Equal(t, localSIFFileName, b.dstFileName)
				assert.Nil(t, b.libraryRef)
			} else {
				if assert.NotNil(t, b.libraryRef) {
					assert.Equal(t, libraryPath, b.libraryRef.Path)
				}
			}

			if tt.uriOverride != "" {
				assert.Equal(t, tt.uriOverride, b.buildURL)
			} else {
				assert.Equal(t, defaultBuildURI, b.buildURL)
			}
		})
	}
}

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

var upgrader = websocket.Upgrader{} // use default options

// Test_build is a rudimentary unit test for (*App).build() method
func Test_build(t *testing.T) {
	const testBuildID = "6387923149ab6b512d0326f3"

	buildSrvMux := http.NewServeMux()

	// Handler for '/v1/build'
	buildSrvMux.HandleFunc("/v1/build", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		mockBuildResponse := struct {
			ID string `json:"id"`
		}{
			ID: testBuildID,
		}

		if err := jsonresp.WriteResponse(w, &mockBuildResponse, http.StatusCreated); err != nil {
			t.Fatalf("response encoding error: %v", err)
		}
	}))

	// Handler for '/v1/build/'
	buildSrvMux.HandleFunc("/v1/build/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		response := struct {
			ID         string `json:"id"`
			ImageSize  int64  `json:"imageSize"`
			LibraryRef string `json:"libraryRef"`
		}{
			ID:         testBuildID,
			ImageSize:  1234,
			LibraryRef: "entity/collection/container:tag",
		}

		if err := jsonresp.WriteResponse(w, &response, http.StatusOK); err != nil {
			t.Fatalf("response encoding error: %v", err)
		}
	}))

	// Handler for '/v1/build-ws/'
	buildSrvMux.HandleFunc("/v1/build-ws/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("ws upgrade error: %v", err)
		}
		defer c.Close()

		// Write 10 lines of sample build output
		for i := range 10 {
			if err := c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Sample remote build output: line #%d\n", i))); err != nil {
				t.Fatalf("error writing to websocket: %v", err)
			}
		}

		// Cleanly close websocket
		if err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
			t.Fatalf("error closing ws: %v", err)
		}
	}))

	buildSrv := httptest.NewServer(buildSrvMux)
	defer buildSrv.Close()

	frontendSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		feConfig := endpoints.FrontendConfig{
			LibraryAPI: endpoints.URI{URI: "http://cloud-library-server"},
			BuildAPI:   endpoints.URI{URI: buildSrv.URL},
		}

		if err := json.NewEncoder(w).Encode(&feConfig); err != nil {
			t.Fatalf("response encoding error: %v", err)
		}
	}))
	defer frontendSrv.Close()

	app, err := New(context.Background(), &Config{
		URL:          frontendSrv.URL,
		ArchsToBuild: []string{runtime.GOARCH},
	})
	if err != nil {
		t.Fatalf("initialization error: %v", err)
	}
	_ = app

	const buildDef = "bootstrap: docker\nfrom: alpine:3\n"

	if err := app.build(context.Background(), []byte(buildDef), "", app.archsToBuild); err != nil {
		t.Fatalf("build error: %v", err)
	}
}
