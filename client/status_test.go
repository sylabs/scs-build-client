// Copyright (c) 2018-2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStatus(t *testing.T) {
	// Craft an expired context
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()

	// Table of tests to run
	tests := []struct {
		description  string
		wantErr      error
		responseCode int
		ctx          context.Context //nolint:containedctx
	}{
		{"Success", nil, http.StatusOK, context.Background()},
		{"NotFound", &httpError{Code: http.StatusNotFound}, http.StatusNotFound, context.Background()},
		{"ContextExpired", context.DeadlineExceeded, http.StatusOK, ctx},
	}

	// Start a mock server
	m := mockService{t: t}
	s := httptest.NewServer(&m)
	defer s.Close()

	c, err := NewClient(OptBaseURL(s.URL))
	if err != nil {
		t.Fatal(err)
	}

	// ID to test with
	id := newObjectID()

	// Loop over test cases
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			m.statusResponseCode = tt.responseCode

			// Call the handler
			bi, err := c.GetStatus(tt.ctx, id)

			if got, want := err, tt.wantErr; !errors.Is(got, want) {
				t.Fatalf("got error %v, want %v", got, want)
			}

			if err == nil {
				if bi.ID != id {
					t.Errorf("mismatched ID: %v/%v", bi.ID, id)
				}
				if bi.LibraryRef == "" {
					t.Errorf("empty Library ref")
				}
				if bi.LibraryURL == "" {
					t.Errorf("empty Library URL")
				}
			}
		})
	}
}
