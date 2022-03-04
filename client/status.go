// Copyright (c) 2018-2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"context"
	"fmt"
	"net/http"

	jsonresp "github.com/sylabs/json-resp"
)

// GetStatus gets the status of a build from the Build Service by build ID
func (c *Client) GetStatus(ctx context.Context, buildID string) (BuildInfo, error) {
	req, err := c.newRequest(http.MethodGet, "/v1/build/"+buildID, nil)
	if err != nil {
		return BuildInfo{}, fmt.Errorf("%w", err)
	}
	req = req.WithContext(ctx)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return BuildInfo{}, fmt.Errorf("%w", err)
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 { // non-2xx status code
		return BuildInfo{}, fmt.Errorf("%w", errorFromResponse(res))
	}

	var bi BuildInfo
	if err = jsonresp.ReadResponse(res.Body, &bi); err != nil {
		return BuildInfo{}, fmt.Errorf("%w", err)
	}

	return bi, nil
}
