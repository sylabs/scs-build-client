// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package client

import (
	"errors"
	"net/http"
	"testing"
)

func TestHTTPError(t *testing.T) {
	tests := []struct {
		name        string
		code        int
		err         error
		wantMessage string
	}{
		{
			name:        "BadRequest",
			code:        http.StatusBadRequest,
			wantMessage: "400 Bad Request",
		},
		{
			name:        "BadRequestWithMessage",
			code:        http.StatusBadRequest,
			err:         errors.New("more good needed"),
			wantMessage: "400 Bad Request: more good needed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &httpError{
				Code: tt.code,
				err:  tt.err,
			}

			if got, want := err.Code, tt.code; got != want {
				t.Errorf("got code %v, want %v", got, want)
			}
			if got, want := err.Unwrap(), tt.err; got != want {
				t.Errorf("got unwrapped error %v, want %v", got, want)
			}
			if got, want := err.Error(), tt.wantMessage; got != want {
				t.Errorf("got message %v, want %v", got, want)
			}
		})
	}
}
