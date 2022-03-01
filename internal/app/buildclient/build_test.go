// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestStdoutLogger(t *testing.T) {
	logger := stdoutLogger{}

	const testString = "this is a test"
	testStringLength := int64(len(testString))

	tests := []struct {
		name        string
		messageType int
		messageData []byte
		messageLen  int64
		expectError error
	}{
		{"TextMessage", websocket.TextMessage, []byte(testString), testStringLength, nil},
		{"BinaryMessage", websocket.BinaryMessage, []byte{1, 2, 3, 4}, 4, nil},
		{"InvalidMessage", websocket.PingMessage, nil, 0, errUnknownMessageType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			var bytesWritten int64

			reportedBytesWritten, err := logger.Read(tt.messageType, tt.messageData)
			if tt.expectError == nil && assert.NoError(t, err) {
				outC := make(chan string)

				var buf bytes.Buffer

				// copy output in goroutine so printing can't block
				go func() {
					var err error
					bytesWritten, err = io.Copy(&buf, r)
					assert.NoError(t, err)
					outC <- buf.String()
				}()

				// restore os.Stdout
				_ = w.Close()
				os.Stdout = old // restoring the real stdout
				<-outC

				output, err := io.ReadAll(&buf)
				if assert.NoError(t, err) {
					if tt.messageType == websocket.TextMessage {
						// Ensure stdout matches message data
						assert.Equal(t, tt.messageData, output)

						// Ensure reported bytes written matches bytes copied (this is likely redundant)
						assert.Equal(t, int64(reportedBytesWritten), bytesWritten)

						// Ensure reported bytes written matches test length
						assert.Equal(t, tt.messageLen, int64(reportedBytesWritten))
					} else if tt.messageType == websocket.BinaryMessage {
						// assert.Greater(t, )
						assert.True(t, bytesWritten > 0)
					}
				} else if tt.expectError != nil {
					assert.Error(t, err, tt.expectError)
				}
			}
		})
	}
}
