// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	jsonresp "github.com/sylabs/json-resp"
)

type mockUploadBuildContext struct {
	t      *testing.T
	code1  int // for "/v1/build-context"
	code2  int // for "/upload-here"
	size   int64
	digest string
}

func (m *mockUploadBuildContext) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// The general flow is that the client POST to /v1/build-context to get an upload URL, and then
	// POST the archive to the upload URL.
	switch r.URL.Path {
	case "/v1/build-context":
		if got, want := r.Method, http.MethodPost; got != want {
			m.t.Errorf("got method %v, want %v", got, want)
		}

		if m.code1 != 0 {
			w.WriteHeader(m.code1)
			return
		}

		if got, want := r.Header.Get("Content-Type"), "application/json"; got != want {
			m.t.Errorf("got content type %v, want %v", got, want)
		}

		var body struct {
			Size   int64  `json:"size"`
			Digest string `json:"checksum"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			m.t.Fatalf("failed to decode request: %v", err)
		}

		// Record size and digest, so we can check them when the archive is uploaded.
		m.size = body.Size
		m.digest = body.Digest

		// Return upload URL to caller.
		w.Header().Set("Location", "/upload-here")

		w.WriteHeader(http.StatusAccepted)

	case "/upload-here":
		if got, want := r.Method, http.MethodPut; got != want {
			m.t.Errorf("got method %v, want %v", got, want)
		}

		if m.code2 != 0 {
			w.WriteHeader(m.code2)
			return
		}

		if got, want := r.Header.Get("Content-Type"), "application/octet-stream"; got != want {
			m.t.Errorf("got content type %v, want %v", got, want)
		}

		if got, want := r.ContentLength, m.size; got != want {
			m.t.Errorf("got content length %v, want %v", got, want)
		}

		h := sha256.New()

		n, err := io.Copy(h, r.Body)
		if err != nil {
			m.t.Fatal(err)
		}

		if got, want := n, m.size; got != want {
			m.t.Errorf("got size %v, want %v", got, want)
		}

		if got, want := fmt.Sprintf("sha256.%x", h.Sum(nil)), m.digest; got != want {
			m.t.Errorf("got digest %v, want %v", got, want)
		}

		w.WriteHeader(http.StatusCreated)

	default:
		m.t.Errorf("unexpected path: %v", r.URL.Path)
	}
}

func TestClient_UploadBuildContext(t *testing.T) {
	fsys := fstest.MapFS{
		"a": &fstest.MapFile{
			Mode:    0o755 | fs.ModeDir,
			ModTime: testTime,
		},
		"a/b": &fstest.MapFile{
			Data:    []byte("a"),
			Mode:    0o755,
			ModTime: testTime,
		},
		"c": &fstest.MapFile{
			Mode:    0o755 | fs.ModeDir,
			ModTime: testTime,
		},
		"c/d": &fstest.MapFile{
			Data:    []byte("b"),
			Mode:    0o755 | fs.ModeSymlink,
			ModTime: testTime,
		},
	}

	tests := []struct {
		name       string
		code1      int
		code2      int
		paths      []string
		wantErr    error
		wantDigest string
	}{
		{
			name:    "NoPaths",
			paths:   []string{},
			wantErr: errNoPathsSpecified,
		},
		{
			name:    "NotExistPath",
			paths:   []string{"b"},
			wantErr: fs.ErrNotExist,
		},
		{
			name:    "HTTPError",
			code1:   http.StatusBadRequest,
			paths:   []string{"."},
			wantErr: &httpError{Code: http.StatusBadRequest},
		},
		{
			name: "WalkDir",
			paths: []string{
				".",
			},
			wantDigest: "sha256.b59c5b1086aac46b5ca3c83e3b9cb1966b30f8681c77da044a6b81d6823ec893",
		},
		{
			name: "Glob",
			paths: []string{
				"*",
			},
			wantDigest: "sha256.b59c5b1086aac46b5ca3c83e3b9cb1966b30f8681c77da044a6b81d6823ec893",
		},
		{
			name: "OneFile",
			paths: []string{
				"a/b",
			},
			wantDigest: "sha256.37196eb7e4e93ba4ac97942e053dc31bb7132ad711ddc14e757fd096d389b97f",
		},
		{
			name: "TwoFiles",
			paths: []string{
				"a/b",
				"c/d",
			},
			wantDigest: "sha256.fc3acf5795d393a706682d78bedf02dc0674fd44b7dd7aa83d91e7560b64bb51",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(&mockUploadBuildContext{
				t:     t,
				code1: tt.code1,
				code2: http.StatusCreated,
			})
			t.Cleanup(s.Close)

			c, err := NewClient(OptBaseURL(s.URL))
			if err != nil {
				t.Fatal(err)
			}

			digest, err := c.UploadBuildContext(context.Background(), tt.paths, optUploadBuildContextFS(fsys))

			if got, want := err, tt.wantErr; !errors.Is(got, want) {
				t.Errorf("got error %v, want %v", got, want)
			}

			if got, want := digest, tt.wantDigest; got != want {
				t.Errorf("got digest %v, want %v", got, want)
			}
		})
	}
}

type mockDeleteBuildContext struct {
	t      *testing.T
	code   int
	digest string
}

func (m *mockDeleteBuildContext) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.code != 0 {
		if err := jsonresp.WriteError(w, "", m.code); err != nil {
			m.t.Fatalf("failed to write error: %v", err)
		}
		return
	}

	if got, want := r.Method, http.MethodDelete; got != want {
		m.t.Errorf("got method %v, want %v", got, want)
	}

	if got, want := r.URL.Path, fmt.Sprintf("/v1/build-context/%v", m.digest); got != want {
		m.t.Errorf("got path %v, want %v", got, want)
	}

	w.WriteHeader(http.StatusOK)
}

func TestClient_DeleteBuildContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		code    int
		digest  string
		wantErr error
	}{
		{
			name:   "OK",
			digest: "sha256.f2ca1bb6c7e907d06dafe4687e579fce76b37e4e93b7605022da52e6ccc26fd2",
		},
		{
			name:    "HTTPError",
			code:    http.StatusBadRequest,
			digest:  "sha256.f2ca1bb6c7e907d06dafe4687e579fce76b37e4e93b7605022da52e6ccc26fd2",
			wantErr: &httpError{Code: http.StatusBadRequest},
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := httptest.NewServer(&mockDeleteBuildContext{
				t:      t,
				code:   tt.code,
				digest: tt.digest,
			})
			defer s.Close()

			c, err := NewClient(OptBaseURL(s.URL))
			if err != nil {
				t.Fatal(err)
			}

			err = c.DeleteBuildContext(context.Background(), tt.digest)

			if got, want := err, tt.wantErr; !errors.Is(got, want) {
				t.Errorf("got error %v, want %v", got, want)
			}
		})
	}
}
