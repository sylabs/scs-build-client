// Copyright (c) 2018-2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	httpClient := &http.Client{}

	tests := []struct {
		name            string
		opts            []Option
		wantErr         bool
		wantURL         string
		wantBearerToken string
		wantUserAgent   string
		wantHTTPClient  *http.Client
	}{
		{"NilConfig", nil, false, defaultBaseURL, "", "", http.DefaultClient},
		{"HTTPBaseURL", []Option{
			OptBaseURL("http://build.staging.sylabs.io"),
		}, false, "http://build.staging.sylabs.io/", "", "", http.DefaultClient},
		{"HTTPSBaseURL", []Option{
			OptBaseURL("https://build.staging.sylabs.io"),
		}, false, "https://build.staging.sylabs.io/", "", "", http.DefaultClient},
		{"HTTPSBaseURLWithPath", []Option{
			OptBaseURL("https://build.staging.sylabs.io/path"),
		}, false, "https://build.staging.sylabs.io/path/", "", "", http.DefaultClient},
		{"HTTPSBaseURLWithPathSlash", []Option{
			OptBaseURL("https://build.staging.sylabs.io/path/"),
		}, false, "https://build.staging.sylabs.io/path/", "", "", http.DefaultClient},
		{"UnsupportedBaseURL", []Option{
			OptBaseURL("bad:"),
		}, true, "", "", "", nil},
		{"BadBaseURL", []Option{
			OptBaseURL(":"),
		}, true, "", "", "", nil},
		{"BearerToken", []Option{
			OptBearerToken("blah"),
		}, false, defaultBaseURL, "blah", "", http.DefaultClient},
		{"UserAgent", []Option{
			OptUserAgent("Secret Agent Man"),
		}, false, defaultBaseURL, "", "Secret Agent Man", http.DefaultClient},
		{"HTTPClient", []Option{
			OptHTTPClient(httpClient),
		}, false, defaultBaseURL, "", "", httpClient},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClient(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("got err %v, want %v", err, tt.wantErr)
			}

			if err == nil {
				if got, want := c.baseURL.String(), tt.wantURL; got != want {
					t.Errorf("got host %v, want %v", got, want)
				}

				if got, want := c.bearerToken, tt.wantBearerToken; got != want {
					t.Errorf("got auth token %v, want %v", got, want)
				}

				if got, want := c.userAgent, tt.wantUserAgent; got != want {
					t.Errorf("got user agent %v, want %v", got, want)
				}

				if got, want := c.httpClient, tt.wantHTTPClient; got != want {
					t.Errorf("got HTTP client %v, want %v", got, want)
				}
			}
		})
	}
}

func TestNewRequest(t *testing.T) {
	tests := []struct {
		name            string
		opts            []Option
		method          string
		path            string
		body            string
		wantErr         bool
		wantURL         string
		wantBearerToken string
		wantUserAgent   string
	}{
		{"BadMethod", nil, "b@d	", "", "", true, "", "", ""},
		{"NilConfigGet", nil, http.MethodGet, "/path", "", false, "https://build.sylabs.io/path", "", ""},
		{"NilConfigPost", nil, http.MethodPost, "/path", "", false, "https://build.sylabs.io/path", "", ""},
		{"NilConfigPostBody", nil, http.MethodPost, "/path", "body", false, "https://build.sylabs.io/path", "", ""},
		{"HTTPBaseURL", []Option{
			OptBaseURL("http://build.staging.sylabs.io"),
		}, http.MethodGet, "/path", "", false, "http://build.staging.sylabs.io/path", "", ""},
		{"HTTPSBaseURL", []Option{
			OptBaseURL("https://build.staging.sylabs.io"),
		}, http.MethodGet, "/path", "", false, "https://build.staging.sylabs.io/path", "", ""},
		{"BaseURLWithPath", []Option{
			OptBaseURL("https://build.staging.sylabs.io/path"),
		}, http.MethodGet, "/path", "", false, "https://build.staging.sylabs.io/path/path", "", ""},
		{"BearerToken", []Option{
			OptBearerToken("blah"),
		}, http.MethodGet, "/path", "", false, "https://build.sylabs.io/path", "BEARER blah", ""},
		{"UserAgent", []Option{
			OptUserAgent("Secret Agent Man"),
		}, http.MethodGet, "/path", "", false, "https://build.sylabs.io/path", "", "Secret Agent Man"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClient(tt.opts...)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			r, err := c.newRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if (err != nil) != tt.wantErr {
				t.Fatalf("got err %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {
				if got, want := r.Method, tt.method; got != want {
					t.Errorf("got method %v, want %v", got, want)
				}

				if got, want := r.URL.String(), tt.wantURL; got != want {
					t.Errorf("got URL %v, want %v", got, want)
				}

				b, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("failed to read body: %v", err)
				}
				if got, want := string(b), tt.body; got != want {
					t.Errorf("got body %v, want %v", got, want)
				}

				authBearer, ok := r.Header["Authorization"]
				if got, want := ok, (tt.wantBearerToken != ""); got != want {
					t.Fatalf("presence of auth bearer %v, want %v", got, want)
				}
				if ok {
					if got, want := len(authBearer), 1; got != want {
						t.Fatalf("got %v auth bearer(s), want %v", got, want)
					}
					if got, want := authBearer[0], tt.wantBearerToken; got != want {
						t.Errorf("got auth bearer %v, want %v", got, want)
					}
				}

				userAgent, ok := r.Header["User-Agent"]
				if got, want := ok, (tt.wantUserAgent != ""); got != want {
					t.Fatalf("presence of user agent %v, want %v", got, want)
				}
				if ok {
					if got, want := len(userAgent), 1; got != want {
						t.Fatalf("got %v user agent(s), want %v", got, want)
					}
					if got, want := userAgent[0], tt.wantUserAgent; got != want {
						t.Errorf("got user agent %v, want %v", got, want)
					}
				}
			}
		})
	}
}
