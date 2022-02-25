// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// OutputReader interface is used to read the websocket output from the stream
type OutputReader interface {
	// Read is called when a websocket message is received
	Read(messageType int, p []byte) (int, error)
}

// GetOutput reads the build output log for the provided buildID - streaming to
// OutputReader. The context controls the lifetime of the request.
func (c *Client) GetOutput(ctx context.Context, buildID string, or OutputReader) error {
	u := c.BaseURL.ResolveReference(&url.URL{
		Path: "v1/build-ws/" + buildID,
	})

	wsScheme := "ws"
	if c.BaseURL.Scheme == "https" {
		wsScheme = "wss"
	}
	u.Scheme = wsScheme

	h := http.Header{}
	c.setRequestHeaders(h)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	dialer := websocket.DefaultDialer

	if c.HTTPClient != nil {
		if tr, ok := c.HTTPClient.Transport.(*http.Transport); ok {
			dialer.TLSClientConfig = tr.TLSClientConfig
		}
	}

	ws, resp, err := dialer.DialContext(ctx, u.String(), h)
	if err != nil {
		c.Logger.Logf("websocket dial err - %s, partial response: %+v", err, resp)
		return err
	}
	defer resp.Body.Close()
	defer ws.Close()

	errChan := make(chan error)

	go func() {
		defer close(errChan)
		errChan <- func() error {
			for {
				// Read from websocket
				mt, msg, err := ws.ReadMessage()
				if err != nil {
					if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
						return nil
					}
					c.Logger.Logf("websocket read message err - %s", err)
					return err
				}

				n, err := or.Read(mt, msg)
				if err != nil {
					return err
				}
				if n != len(msg) {
					return fmt.Errorf("did not read all message contents: %d != %d", n, len(msg))
				}
			}
		}()
	}()

	select {
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := c.Cancel(ctx, buildID); err != nil { //nolint
			c.Logger.Logf("build cancellation request failed: %v", err)
		}

		ws.Close()

		<-errChan
		return nil
	case err := <-errChan:
		return err
	}
}
