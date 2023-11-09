// Copyright (c) 2019-2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type testOutputWriter struct {
	fully bool
	err   error
}

func (tor testOutputWriter) Write(p []byte) (int, error) {
	if tor.fully {
		return len(p), tor.err
	}
	return len(p) - 1, tor.err
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
		ctx             context.Context //nolint:containedctx
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
	for _, useTLS := range []bool{true, false} {
		useTLS := useTLS

		name := func() string {
			if useTLS {
				return "WithTLS"
			}
			return "WithoutTLS"
		}()

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for _, tt := range tests {
				tt := tt

				t.Run(tt.description, func(t *testing.T) {
					// Start a mock server
					m := mockService{t: t}
					mux := http.NewServeMux()
					mux.HandleFunc(wsPath, m.ServeWebsocket)

					clientOptions := []Option{}

					var s *httptest.Server
					if useTLS {
						s = httptest.NewTLSServer(mux)

						tr, ok := s.Client().Transport.(*http.Transport)
						if !ok {
							t.Fatal("Internal error- unable to typecast HTTP client transport")
						}
						tr = tr.Clone()

						clientOptions = append(clientOptions, OptHTTPTransport(tr))
					} else {
						s = httptest.NewServer(mux)
					}
					defer s.Close()

					// Mock server address is fixed for all tests
					m.httpAddr = s.Listener.Addr().String()

					c, err := NewClient(append(clientOptions, OptBaseURL(s.URL), OptBearerToken(authToken))...)
					if err != nil {
						t.Fatal(err)
					}

					// Set the response codes for each stage of the build
					m.wsResponseCode = tt.wsResponseCode
					m.wsCloseCode = tt.wsCloseCode

					tor := testOutputWriter{
						fully: tt.outputReadFully,
						err:   tt.outputReadErr,
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
		})
	}
}
