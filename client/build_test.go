// Copyright (c) 2018-2025, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSubmit(t *testing.T) {
	// Craft an expired context
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()

	// Table of tests to run
	tests := []struct {
		description  string
		wantErr      error
		libraryRef   string
		responseCode int
		ctx          context.Context //nolint:containedctx
	}{
		{"SuccessAttached", nil, "", http.StatusCreated, context.Background()},
		{"SuccessLibraryRef", nil, "library://user/collection/image", http.StatusCreated, context.Background()},
		{"NotFoundAttached", &httpError{Code: http.StatusNotFound}, "", http.StatusNotFound, context.Background()},
		{"ContextExpiredAttached", context.DeadlineExceeded, "", http.StatusCreated, ctx},
	}

	// Start a mock server
	m := mockService{t: t}

	s := httptest.NewServer(&m)
	defer s.Close()

	c, err := NewClient(OptBaseURL(s.URL))
	if err != nil {
		t.Fatal(err)
	}

	// Loop over test cases
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			m.buildResponseCode = tt.responseCode

			// Call the handler
			bi, err := c.Submit(tt.ctx, strings.NewReader(""),
				OptBuildLibraryRef(tt.libraryRef),
			)
			if got, want := err, tt.wantErr; !errors.Is(got, want) {
				t.Fatalf("got error %v, want %v", got, want)
			}

			if err == nil {
				if bi.ID() == "" {
					t.Fatalf("invalid ID")
				}

				if bi.LibraryRef() == "" {
					t.Errorf("empty Library ref")
				}

				if bi.LibraryURL() == "" {
					t.Errorf("empty Library URL")
				}
			}
		})
	}
}

func TestCancel(t *testing.T) {
	// Start a mock server
	m := mockService{t: t}

	s := httptest.NewServer(&m)
	defer s.Close()

	c, err := NewClient(OptBaseURL(s.URL))
	if err != nil {
		t.Fatal(err)
	}

	m.cancelResponseCode = 204

	err = c.Cancel(context.Background(), "00000000")
	if err != nil {
		t.Fatal(err)
	}
}
