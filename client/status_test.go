// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/sylabs/scs-build-client/client"
)

func TestStatus(t *testing.T) {
	// Craft an expired context
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()

	// Table of tests to run
	tests := []struct {
		description   string
		expectSuccess bool
		responseCode  int
		ctx           context.Context
	}{
		{"Success", true, http.StatusOK, context.Background()},
		{"NotFound", false, http.StatusNotFound, context.Background()},
		{"ContextExpired", false, http.StatusOK, ctx},
	}

	// Start a mock server
	m := mockService{t: t}
	s := httptest.NewServer(&m)
	defer s.Close()

	// Enough of a struct to test with
	url, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	c, err := client.New(&client.Config{
		BaseURL: url.String(),
	})
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	// ID to test with
	id := newObjectID()

	// Loop over test cases
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			m.statusResponseCode = tt.responseCode

			// Call the handler
			bi, err := c.GetStatus(tt.ctx, id)

			if tt.expectSuccess {
				// Ensure the handler returned no error, and the response is as expected
				if err != nil {
					t.Fatalf("unexpected failure: %v", err)
				}
				if bi.ID != id {
					t.Errorf("mismatched ID: %v/%v", bi.ID, id)
				}
				if bi.LibraryRef == "" {
					t.Errorf("empty Library ref")
				}
				if bi.LibraryURL == "" {
					t.Errorf("empty Library URL")
				}
			} else {
				// Ensure the handler returned an error
				if err == nil {
					t.Fatalf("unexpected success")
				}
			}
		})
	}
}
