// Copyright (c) 2018-2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	jsonresp "github.com/sylabs/json-resp"
)

// Submit sends a build job to the Build Service. The context controls the lifetime of the request.
func (c *Client) Submit(ctx context.Context, br BuildRequest) (BuildInfo, error) {
	ref := &url.URL{
		Path: "v1/build",
	}

	b, err := json.Marshal(br)
	if err != nil {
		return BuildInfo{}, fmt.Errorf("%w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, ref, bytes.NewReader(b))
	if err != nil {
		return BuildInfo{}, fmt.Errorf("%w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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

// Cancel cancels an existing build. The context controls the lifetime of the request.
func (c *Client) Cancel(ctx context.Context, buildID string) error {
	ref := &url.URL{
		Path: fmt.Sprintf("v1/build/%v/_cancel", buildID),
	}

	req, err := c.newRequest(ctx, http.MethodPut, ref, nil)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 { // non-2xx status code
		return fmt.Errorf("%w", errorFromResponse(res))
	}

	return nil
}
