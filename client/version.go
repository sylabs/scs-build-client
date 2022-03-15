// Copyright (c) 2019-2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	jsonresp "github.com/sylabs/json-resp"
)

// GetVersion gets version information from the build service. The context controls the lifetime of
// the request.
func (c *Client) GetVersion(ctx context.Context) (string, error) {
	ref := &url.URL{
		Path: "version",
	}

	req, err := c.newRequest(ctx, http.MethodGet, ref, nil)
	if err != nil {
		return "", fmt.Errorf("%w", err)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w", err)
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 { // non-2xx status code
		return "", fmt.Errorf("%w", errorFromResponse(res))
	}

	vi := struct {
		Version string `json:"version"`
	}{}
	if err := jsonresp.ReadResponse(res.Body, &vi); err != nil {
		return "", fmt.Errorf("%w", err)
	}
	return vi.Version, nil
}
