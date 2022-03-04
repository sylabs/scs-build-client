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

// VersionInfo contains version information.
type VersionInfo struct {
	Version string `json:"version"`
}

// GetVersion gets version information from the build service. The context controls the lifetime of
// the request.
func (c *Client) GetVersion(ctx context.Context) (VersionInfo, error) {
	ref := &url.URL{
		Path: "version",
	}

	req, err := c.newRequest(ctx, http.MethodGet, ref, nil)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("%w", err)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return VersionInfo{}, fmt.Errorf("%w", err)
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 { // non-2xx status code
		return VersionInfo{}, fmt.Errorf("%w", errorFromResponse(res))
	}

	var vi VersionInfo
	if err := jsonresp.ReadResponse(res.Body, &vi); err != nil {
		return VersionInfo{}, fmt.Errorf("%w", err)
	}

	return vi, nil
}
