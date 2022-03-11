// Copyright (c) 2018-2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	jsonresp "github.com/sylabs/json-resp"
)

// GetStatus gets the status of a build from the Build Service by build ID. The context controls
// the lifetime of the request.
func (c *Client) GetStatus(ctx context.Context, buildID string) (*BuildInfo, error) {
	ref := &url.URL{
		Path: "v1/build/" + buildID,
	}

	req, err := c.newRequest(ctx, http.MethodGet, ref, nil)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 { // non-2xx status code
		return nil, fmt.Errorf("%w", errorFromResponse(res))
	}

	var rbi rawBuildInfo
	if err = jsonresp.ReadResponse(res.Body, &rbi); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	return &BuildInfo{rbi}, nil
}
