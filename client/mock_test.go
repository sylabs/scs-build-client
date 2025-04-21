// Copyright (c) 2018-2025, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	jsonresp "github.com/sylabs/json-resp"
)

type mockService struct {
	t                  *testing.T
	buildResponseCode  int
	wsResponseCode     int
	wsCloseCode        int
	statusResponseCode int
	imageResponseCode  int
	cancelResponseCode int
	httpAddr           string
}

var upgrader = websocket.Upgrader{}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

const (
	authToken         = "auth_token"
	stdoutContents    = "some_output"
	imageContents     = "image_contents"
	buildPath         = "/v1/build"
	wsPath            = "/v1/build-ws/"
	imagePath         = "/v1/image"
	buildCancelSuffix = "/_cancel"
)

func newResponse(m *mockService, id string, libraryRef string) rawBuildInfo {
	libraryURL := url.URL{
		Scheme: "http",
		Host:   m.httpAddr,
	}
	if libraryRef == "" {
		libraryRef = "library://user/collection/image"
	}

	return rawBuildInfo{
		ID:         id,
		LibraryURL: libraryURL.String(),
		LibraryRef: libraryRef,
		IsComplete: true,
		ImageSize:  1,
	}
}

func (m *mockService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set the response body, depending on the type of operation
	if r.Method == http.MethodPost && r.RequestURI == buildPath {
		// Mock new build endpoint
		var br struct {
			LibraryRef string `json:"libraryRef"`
		}
		if err := json.NewDecoder(r.Body).Decode(&br); err != nil {
			m.t.Fatalf("failed to parse request: %v", err)
		}
		if m.buildResponseCode == http.StatusCreated {
			id := newObjectID()
			if err := jsonresp.WriteResponse(w, newResponse(m, id, br.LibraryRef), m.buildResponseCode); err != nil {
				m.t.Fatal(err)
			}
		} else {
			if err := jsonresp.WriteError(w, "", m.buildResponseCode); err != nil {
				m.t.Fatal(err)
			}
		}
	} else if r.Method == http.MethodGet && strings.HasPrefix(r.RequestURI, buildPath) {
		// Mock status endpoint
		id := r.RequestURI[strings.LastIndexByte(r.RequestURI, '/')+1:]
		if id == "" {
			m.t.Fatalf("failed to parse ID '%v'", id)
		}
		if m.statusResponseCode == http.StatusOK {
			if err := jsonresp.WriteResponse(w, newResponse(m, id, ""), m.statusResponseCode); err != nil {
				m.t.Fatal(err)
			}
		} else {
			if err := jsonresp.WriteError(w, "", m.statusResponseCode); err != nil {
				m.t.Fatal(err)
			}
		}
	} else if r.Method == http.MethodGet && strings.HasPrefix(r.RequestURI, imagePath) {
		// Mock get image endpoint
		if m.imageResponseCode == http.StatusOK {
			if _, err := strings.NewReader(imageContents).WriteTo(w); err != nil {
				m.t.Fatalf("failed to write image - %v", err)
			}
		} else {
			if err := jsonresp.WriteError(w, "", m.imageResponseCode); err != nil {
				m.t.Fatal(err)
			}
		}
	} else if r.Method == http.MethodPut && strings.HasSuffix(r.RequestURI, buildCancelSuffix) {
		// Mock build cancellation endpoint
		if m.cancelResponseCode == http.StatusNoContent {
			w.WriteHeader(http.StatusNoContent)
		} else {
			if err := jsonresp.WriteError(w, "", m.cancelResponseCode); err != nil {
				m.t.Fatal(err)
			}
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *mockService) ServeWebsocket(w http.ResponseWriter, r *http.Request) {
	if m.wsResponseCode != http.StatusOK {
		w.WriteHeader(m.wsResponseCode)
	} else {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			m.t.Fatalf("failed to upgrade websocket: %v", err)
		}
		defer ws.Close()

		// Write some output and then cleanly close the connection
		if err = ws.WriteMessage(websocket.TextMessage, []byte(stdoutContents)); err != nil {
			m.t.Fatalf("error writing websocket message - %v", err)
		}
		if err = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(m.wsCloseCode, "")); err != nil {
			m.t.Fatalf("error writing websocket close message - %v", err)
		}
	}
}

func TestBuild(t *testing.T) {
	// Craft an expired context
	expiredCtx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()

	// Create a temporary file for testing
	f, err := os.CreateTemp("/tmp", "TestBuild")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	// Start a mock server
	m := mockService{t: t}
	mux := http.NewServeMux()
	mux.HandleFunc("/", m.ServeHTTP)
	mux.HandleFunc(wsPath, m.ServeWebsocket)
	s := httptest.NewServer(mux)
	defer s.Close()

	// Mock server address is fixed for all tests
	m.httpAddr = s.Listener.Addr().String()

	// Table of tests to run
	tests := []struct {
		description         string
		expectSubmitSuccess bool
		expectStreamSuccess bool
		expectStatusSuccess bool
		imagePath           string
		libraryURL          string
		buildResponseCode   int
		wsResponseCode      int
		wsCloseCode         int
		statusResponseCode  int
		imageResponseCode   int
		ctx                 context.Context //nolint:containedctx
	}{
		{"Success", true, true, true, f.Name(), "", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background()},
		{"SuccessLibraryRef", true, true, true, "library://user/collection/image", "", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background()},
		{"SuccessLibraryRefURL", true, true, true, "library://user/collection/image", m.httpAddr, http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background()},
		{"AddBuildFailure", false, false, false, f.Name(), "", http.StatusUnauthorized, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background()},
		{"WebsocketFailure", true, false, true, f.Name(), "", http.StatusCreated, http.StatusUnauthorized, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, context.Background()},
		{"WebsocketAbnormalClosure", true, false, true, f.Name(), "", http.StatusCreated, http.StatusOK, websocket.CloseAbnormalClosure, http.StatusOK, http.StatusOK, context.Background()},
		{"GetStatusFailure", true, true, false, f.Name(), "", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusUnauthorized, http.StatusOK, context.Background()},
		{"ContextExpired", false, false, false, f.Name(), "", http.StatusCreated, http.StatusOK, websocket.CloseNormalClosure, http.StatusOK, http.StatusOK, expiredCtx},
	}

	// Loop over test cases
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			c, err := NewClient(
				OptBaseURL(s.URL),
				OptBearerToken(authToken),
			)
			if err != nil {
				t.Fatal(err)
			}

			// Set the response codes for each stage of the build
			m.buildResponseCode = tt.buildResponseCode
			m.wsResponseCode = tt.wsResponseCode
			m.wsCloseCode = tt.wsCloseCode
			m.statusResponseCode = tt.statusResponseCode
			m.imageResponseCode = tt.imageResponseCode

			// Do it!
			bd, err := c.Submit(tt.ctx, strings.NewReader(""),
				OptBuildLibraryRef(tt.imagePath),
			)
			if !tt.expectSubmitSuccess {
				// Ensure the handler returned an error
				if err == nil {
					t.Fatalf("unexpected submit success")
				}
				return
			}
			// Ensure the handler returned no error, and the response is as expected
			if err != nil {
				t.Fatalf("unexpected submit failure: %v", err)
			}

			tor := testOutputWriter{
				fully: true,
				err:   nil,
			}
			err = c.GetOutput(tt.ctx, bd.ID(), tor)
			if tt.expectStreamSuccess {
				// Ensure the handler returned no error, and the response is as expected
				if err != nil {
					t.Fatalf("unexpected stream failure: %v", err)
				}
			} else {
				// Ensure the handler returned an error
				if err == nil {
					t.Fatalf("unexpected stream success")
				}
			}

			_, err = c.GetStatus(tt.ctx, bd.ID())
			if tt.expectStatusSuccess {
				// Ensure the handler returned no error, and the response is as expected
				if err != nil {
					t.Fatalf("unexpected status failure: %v", err)
				}
			} else {
				// Ensure the handler returned an error
				if err == nil {
					t.Fatalf("unexpected status success")
				}
			}
		})
	}
}
