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
	"strings"

	"github.com/globalsign/mgo/bson"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	jsonresp "github.com/sylabs/json-resp"
	"github.com/sylabs/scs-library-client/client"
)

// CloudURI holds the URI of the Library web front-end.
const CloudURI = "https://cloud.sylabs.io"

func (c *Client) Build(ctx context.Context, imagePath string, definition Definition, isDetached bool) (string, error) {
	var libraryRef string

	if strings.HasPrefix(imagePath, "library://") {
		// Image destination is Library.
		libraryRef = imagePath
	}

	// Send build request to Remote Build Service
	rd, err := c.doBuildRequest(ctx, definition, libraryRef)
	if err != nil {
		err = errors.Wrap(err, "failed to post request to remote build service")
		glog.Warningf("%v", err)
		return "", err
	}

	// If we're doing an detached build, print help on how to download the image
	libraryRefRaw := strings.TrimPrefix(rd.LibraryRef, "library://")
	if isDetached {
		// TODO - move this code outside this client, should be in singularity
		fmt.Printf("Build submitted! Once it is complete, the image can be retrieved by running:\n")
		fmt.Printf("\tsingularity pull --library %v library://%v\n\n", rd.LibraryURL, libraryRefRaw)
		fmt.Printf("Alternatively, you can access it from a browser at:\n\t%v/library/%v\n", CloudURI, libraryRefRaw)
	}

	// If we're doing an attached build, stream output and then download the resulting file
	if !isDetached {
		err = c.streamOutput(ctx, rd.WSURL)
		if err != nil {
			err = errors.Wrap(err, "failed to stream output from remote build service")
			glog.Warningf("%v", err)
			return "", err
		}

		// Get build status
		rd, err = c.doStatusRequest(ctx, rd.ID)
		if err != nil {
			err = errors.Wrap(err, "failed to get status from remote build service")
			glog.Warningf("%v", err)
			return "", err
		}

		// Do not try to download image if not complete or image size is 0
		if !rd.IsComplete {
			return "", errors.New("build has not completed")
		}
		if rd.ImageSize <= 0 {
			return "", errors.New("build image size <= 0")
		}
	}

	return rd.LibraryRef, nil
}

// streamOutput attaches via websocket and streams output to the console
func (c *Client) streamOutput(ctx context.Context, url string) (err error) {
	h := http.Header{}
	c.setRequestHeaders(h)

	ws, resp, err := websocket.DefaultDialer.Dial(url, h)
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

// doBuildRequest creates a new build on a Remote Build Service
func (c *Client) doBuildRequest(ctx context.Context, d Definition, libraryRef string) (rd ResponseData, err error) {
	if libraryRef != "" && !client.IsLibraryPushRef(libraryRef) {
		err = fmt.Errorf("invalid library reference: %v", libraryRef)
		glog.Warningf("%v", err)
		return ResponseData{}, err
	}

	b, err := json.Marshal(RequestData{
		Definition: d,
		LibraryRef: libraryRef,
		LibraryURL: c.LibraryURL.String(),
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
			rd.ID.Hex(), rd.WSURL, rd.LibraryRef)
	}
	return
}

// doStatusRequest gets the status of a build from the Remote Build Service
func (c *Client) doStatusRequest(ctx context.Context, id bson.ObjectId) (rd ResponseData, err error) {
	req, err := c.newRequest(http.MethodGet, "/v1/build/"+id.Hex(), "", nil)
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
