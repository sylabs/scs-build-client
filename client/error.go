// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package client

import (
	"errors"
	"fmt"
	"net/http"

	jsonresp "github.com/sylabs/json-resp"
)

// httpError represents an error returned from an HTTP server.
type httpError struct {
	Code int
	err  error
}

// Unwrap returns the error wrapped by e.
func (e *httpError) Unwrap() error { return e.err }

// Error returns a human-readable representation of e.
func (e *httpError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%v %v: %v", e.Code, http.StatusText(e.Code), e.err.Error())
	}
	return fmt.Sprintf("%v %v", e.Code, http.StatusText(e.Code))
}

// Is compares e against target. If target is a HTTPError with the same code as e, true is returned.
func (e *httpError) Is(target error) bool {
	t, ok := target.(*httpError)
	return ok && (t.Code == e.Code)
}

// errorFromResponse returns an HTTPError containing the status code and detailed error message (if
// available) from res.
func errorFromResponse(res *http.Response) error {
	httpErr := httpError{Code: res.StatusCode}

	var jerr *jsonresp.Error
	if err := jsonresp.ReadError(res.Body); errors.As(err, &jerr) {
		httpErr.err = errors.New(jerr.Message)
	}

	return &httpErr
}
