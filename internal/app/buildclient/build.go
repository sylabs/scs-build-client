// Copyright (c) 2022, Sylabs, Inc. All rights reserved.

package buildclient

import (
	"context"
	"errors"
	"fmt"

	"github.com/gorilla/websocket"
	build "github.com/sylabs/scs-build-client/client"
	"github.com/sylabs/singularity/pkg/build/types"
)

var errUnknownMessageType = errors.New("unknown message type")

type stdoutLogger struct{}

func (s stdoutLogger) Read(messageType int, msg []byte) (int, error) {
	switch messageType {
	case websocket.TextMessage:
		return fmt.Print(string(msg))
	case websocket.BinaryMessage:
		return fmt.Print("Ignoring binary message")
	}
	return 0, errUnknownMessageType
}

func (app *App) buildArtifact(ctx context.Context, def types.Definition, arch string, libraryRef string) (*build.BuildInfo, error) {
	bi, err := app.buildClient.Submit(ctx, build.BuildRequest{
		LibraryRef:    libraryRef,
		LibraryURL:    app.libraryClient.BaseURL.String(),
		DefinitionRaw: def.Raw,
		BuilderRequirements: map[string]string{
			"arch": arch,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error submitting remote build: %w", err)
	}
	if err := app.buildClient.GetOutput(ctx, bi.ID, stdoutLogger{}); err != nil {
		return nil, fmt.Errorf("error streaming remote build output: %w", err)
	}
	if bi, err = app.buildClient.GetStatus(ctx, bi.ID); err != nil {
		return nil, fmt.Errorf("error getting remote build status: %w", err)
	}
	return &bi, nil
}
