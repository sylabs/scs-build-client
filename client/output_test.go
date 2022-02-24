// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sylabs/scs-build-client/client"
)

type TestOutputReader struct {
	ReadFully bool
	ReadErr   error
}

func (tor TestOutputReader) Read(messageType int, p []byte) (int, error) {
	// Print to terminal
	switch messageType {
	case websocket.TextMessage:
		fmt.Printf("%s", string(p))
	case websocket.BinaryMessage:
		fmt.Print("Ignoring binary message")
	}

	if tor.ReadFully {
		return len(p), tor.ReadErr
	}
	return len(p) - 1, tor.ReadErr
}

func TestOutput(t *testing.T) {
	// Craft an expired context
	expiredCtx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()

	// Table of tests to run
	// nolint:maligned
	tests := []struct {
		description     string
		expectSuccess   bool
		wsResponseCode  int
		wsCloseCode     int
		ctx             context.Context
		outputReadFully bool
		outputReadErr   error
	}{
		{"Success", true, http.StatusOK, websocket.CloseNormalClosure, context.Background(), true, nil},
		{"OutputReaderUnread", false, http.StatusOK, websocket.CloseNormalClosure, context.Background(), false, nil},
		{"OutputReaderFailure", false, http.StatusOK, websocket.CloseNormalClosure, context.Background(), true, fmt.Errorf("failed to read")},
		{"WebsocketFailure", false, http.StatusUnauthorized, websocket.CloseNormalClosure, context.Background(), true, nil},
		{"WebsocketAbnormalClosure", false, http.StatusOK, websocket.CloseAbnormalClosure, context.Background(), true, nil},
		{"ContextExpired", false, http.StatusOK, websocket.CloseNormalClosure, expiredCtx, true, nil},
	}

	// Loop over test cases
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			// Start a mock server
			m := mockService{t: t}
			mux := http.NewServeMux()
			mux.HandleFunc(wsPath, m.ServeWebsocket)
			s := httptest.NewServer(mux)
			defer s.Close()

			// Mock server address is fixed for all tests
			m.httpAddr = s.Listener.Addr().String()

			url, err := url.Parse(s.URL)
			if err != nil {
				t.Fatalf("failed to parse URL: %v", err)
			}

			c, err := client.New(&client.Config{
				BaseURL:   url.String(),
				AuthToken: authToken,
			})
			if err != nil {
				t.Fatalf("failed to get new builder: %v", err)
			}

			// Set the response codes for each stage of the build
			m.wsResponseCode = tt.wsResponseCode
			m.wsCloseCode = tt.wsCloseCode

			tor := TestOutputReader{
				ReadFully: tt.outputReadFully,
				ReadErr:   tt.outputReadErr,
			}
			err = c.GetOutput(tt.ctx, "id", tor)
			if tt.expectSuccess {
				// Ensure the handler returned no error, and the response is as expected
				if err != nil {
					t.Fatalf("unexpected stream failure: %v", err)
				}
			} else {
				// Ensure the handler returned an error
				if err == nil {
					t.Fatalf("unexpected stream success")
				}
			}
		})
	}
}
