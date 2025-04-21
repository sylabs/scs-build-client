// Copyright (c) 2022-2025, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	jsonresp "github.com/sylabs/json-resp"
)

type mockVersion struct {
	t       *testing.T
	code    int
	message string
	version string
}

func (m *mockVersion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.code/100 != 2 { // non-2xx status code
		if err := jsonresp.WriteError(w, m.message, m.code); err != nil {
			m.t.Fatalf("failed to write error: %v", err)
		}

		return
	}

	if got, want := r.Method, http.MethodGet; got != want {
		m.t.Errorf("got method %v, want %v", got, want)
	}

	if got, want := r.URL.Path, "/version"; got != want {
		m.t.Errorf("got path %v, want %v", got, want)
	}

	vi := struct {
		Version string `json:"version"`
	}{
		Version: m.version,
	}
	if err := jsonresp.WriteResponse(w, vi, m.code); err != nil {
		m.t.Fatalf("failed to write response: %v", err)
	}
}

func TestClient_GetVersion(t *testing.T) {
	t.Parallel()

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name    string
		ctx     context.Context //nolint:containedctx
		code    int
		message string
		version string
		wantErr error
	}{
		{
			name:    "OK",
			ctx:     context.Background(),
			code:    http.StatusOK,
			version: "1.2.3",
		},
		{
			name:    "NonAuthoritativeInfo",
			ctx:     context.Background(),
			code:    http.StatusNonAuthoritativeInfo,
			version: "1.2.3",
		},
		{
			name:    "HTTPError",
			ctx:     context.Background(),
			code:    http.StatusBadRequest,
			wantErr: &httpError{Code: http.StatusBadRequest},
		},
		{
			name:    "HTTPErrorMessage",
			ctx:     context.Background(),
			code:    http.StatusBadRequest,
			message: "blah",
			wantErr: &httpError{Code: http.StatusBadRequest},
		},
		{
			name:    "ContextCanceled",
			ctx:     cancelled,
			code:    http.StatusOK,
			wantErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := httptest.NewServer(&mockVersion{
				t:       t,
				code:    tt.code,
				message: tt.message,
				version: tt.version,
			})
			defer s.Close()

			c, err := NewClient(OptBaseURL(s.URL))
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			v, err := c.GetVersion(tt.ctx)

			if got, want := err, tt.wantErr; !errors.Is(got, want) {
				t.Fatalf("got error %v, want %v", got, want)
			}

			if got, want := v, tt.version; got != want {
				t.Errorf("got version %v, want %v", got, want)
			}
		})
	}
}
