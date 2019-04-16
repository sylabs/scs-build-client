// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
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

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	jsonresp "github.com/sylabs/json-resp"
)

// Build submits a build job to the Build Service. The context controls the
// lifetime of the request.
func (c *Client) SubmitBuild(ctx context.Context, d Definition, libraryRef string, libraryURL string) (rd ResponseData, err error) {

	b, err := json.Marshal(RequestData{
		Definition: d,
		LibraryRef: libraryRef,
		LibraryURL: libraryURL,
	})
	if err != nil {
		return
	}

	req, err := c.newRequest(http.MethodPost, "/v1/build", "", bytes.NewReader(b))
	if err != nil {
		return
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	glog.V(2).Infof("Sending build request to %s", req.URL.String())

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	err = jsonresp.ReadResponse(res.Body, &rd)
	if err == nil {
		glog.V(2).Infof("Build response - id: %s, wsurl: %s, libref: %s",
			rd.ID, rd.WSURL, rd.LibraryRef)
	}
	return rd, err
}

// StreamOutput reads log output from the websocket URL. The context controls
// the lifetime of the request.
func (c *Client) StreamOutput(ctx context.Context, wsURL string) error {
	h := http.Header{}
	c.setRequestHeaders(h)

	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, h)
	if err != nil {
		glog.V(2).Infof("websocket dial err - %s, partial response: %+v", err, resp)
		return err
	}
	defer ws.Close()

	for {
		// Check if context has expired
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read from websocket
		mt, msg, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return nil
			}
			glog.V(2).Infof("websocket read message err - %s", err)
			return err
		}

		// Print to terminal
		switch mt {
		case websocket.TextMessage:
			fmt.Printf("%s", msg)
		case websocket.BinaryMessage:
			fmt.Print("Ignoring binary message")
		}
	}
}

// GetBuildStatus gets the status of a build from the Remote Build Service
func (c *Client) GetBuildStatus(ctx context.Context, buildID string) (rd ResponseData, err error) {
	req, err := c.newRequest(http.MethodGet, "/v1/build/"+buildID, "", nil)
	if err != nil {
		return
	}
	req = req.WithContext(ctx)

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	err = jsonresp.ReadResponse(res.Body, &rd)
	return
}
