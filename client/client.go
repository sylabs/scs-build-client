// Copyright (c) 2019-2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// errUnsupportedProtocolScheme is returned when an unsupported protocol scheme is encountered.
var errUnsupportedProtocolScheme = errors.New("unsupported protocol scheme")

// normalizeURL parses rawURL, and ensures the path component is terminated with a separator.
func normalizeURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%w %s", errUnsupportedProtocolScheme, u.Scheme)
	}

	// Ensure path is terminated with a separator, to prevent url.ResolveReference from stripping
	// the final path component of BaseURL when constructing request URL from a relative path.
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}

	return u, nil
}

// clientOptions describes the options for a Client.
type clientOptions struct {
	baseURL     string
	bearerToken string
	userAgent   string
	transport   http.RoundTripper
}

// Option are used to populate co.
type Option func(co *clientOptions) error

// OptBaseURL sets the base URL of the build server to url.
func OptBaseURL(url string) Option {
	return func(co *clientOptions) error {
		co.baseURL = url
		return nil
	}
}

// OptBearerToken sets the bearer token to include in the "Authorization" header of each request.
func OptBearerToken(token string) Option {
	return func(co *clientOptions) error {
		co.bearerToken = token
		return nil
	}
}

// OptUserAgent sets the HTTP user agent to include in the "User-Agent" header of each request.
func OptUserAgent(agent string) Option {
	return func(co *clientOptions) error {
		co.userAgent = agent
		return nil
	}
}

// OptHTTPTransport sets the transport for HTTP requests to use.
func OptHTTPTransport(tr http.RoundTripper) Option {
	return func(co *clientOptions) error {
		co.transport = tr
		return nil
	}
}

// Client describes the client details.
type Client struct {
	baseURL                *url.URL     // Parsed base URL.
	bearerToken            string       // Bearer token to include in "Authorization" header.
	userAgent              string       // Value to include in "User-Agent" header.
	httpClient             *http.Client // Client to use for HTTP requests.
	buildContextHTTPClient *http.Client // Client to use for build context HTTP requests.
}

const defaultBaseURL = "https://build.sylabs.io/"

// NewClient returns a Client configured according to opts.
//
// By default, the Sylabs Build Service is used. To override this behaviour, use OptBaseURL.
//
// By default, requests are not authenticated. To override this behaviour, use OptBearerToken.
func NewClient(opts ...Option) (*Client, error) {
	co := clientOptions{
		baseURL:   defaultBaseURL,
		transport: http.DefaultTransport,
	}

	// Apply options.
	for _, opt := range opts {
		if err := opt(&co); err != nil {
			return nil, fmt.Errorf("%w", err)
		}
	}

	c := Client{
		bearerToken: co.bearerToken,
		userAgent:   co.userAgent,
		httpClient: &http.Client{
			Transport: co.transport,
			Timeout:   30 * time.Second, // use default from singularity
		},
		buildContextHTTPClient: &http.Client{Transport: co.transport},
	}

	// Normalize base URL.
	u, err := normalizeURL(co.baseURL)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	c.baseURL = u

	return &c, nil
}

// newRequest returns a new Request given a method, ref, and optional body.
//
// The context controls the entire lifetime of a request and its response: obtaining a connection,
// sending the request, and reading the response headers and body.
func (c *Client) newRequest(ctx context.Context, method string, ref *url.URL, body io.Reader) (*http.Request, error) {
	u := c.baseURL.ResolveReference(ref)

	r, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}

	c.setRequestHeaders(r.Header)

	return r, nil
}

// setRequestHeaders sets HTTP headers according to c.
func (c *Client) setRequestHeaders(h http.Header) {
	if v := c.bearerToken; v != "" {
		h.Set("Authorization", fmt.Sprintf("BEARER %s", v))
	}
	if v := c.userAgent; v != "" {
		h.Set("User-Agent", v)
	}
}
