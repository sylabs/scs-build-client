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
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	jsonresp "github.com/sylabs/json-resp"
)

// rawBuildInfo contains the details of an individual build.
type rawBuildInfo struct {
	ID            string `json:"id"`
	IsComplete    bool   `json:"isComplete"`
	ImageSize     int64  `json:"imageSize,omitempty"`
	ImageChecksum string `json:"imageChecksum,omitempty"`
	LibraryRef    string `json:"libraryRef"`
	LibraryURL    string `json:"libraryURL"`
}

// BuildInfo contains the details of an individual build.
type BuildInfo struct {
	raw rawBuildInfo
}

func (bi *BuildInfo) ID() string            { return bi.raw.ID }
func (bi *BuildInfo) IsComplete() bool      { return bi.raw.IsComplete }
func (bi *BuildInfo) ImageSize() int64      { return bi.raw.ImageSize }
func (bi *BuildInfo) ImageChecksum() string { return bi.raw.ImageChecksum }
func (bi *BuildInfo) LibraryRef() string    { return bi.raw.LibraryRef }
func (bi *BuildInfo) LibraryURL() string    { return bi.raw.LibraryURL }

type buildOptions struct {
	libraryRef    string
	arch          string
	libraryURL    string
	contextDigest string
	workingDir    string
}

type BuildOption func(*buildOptions) error

// OptBuildLibraryRef sets the Library image ref to push to.
func OptBuildLibraryRef(imageRef string) BuildOption {
	return func(bo *buildOptions) error {
		bo.libraryRef = imageRef
		return nil
	}
}

// OptBuildArchitecture sets the build architecture to arch.
func OptBuildArchitecture(arch string) BuildOption {
	return func(bo *buildOptions) error {
		bo.arch = arch
		return nil
	}
}

// OptBuildLibraryPullBaseURL sets the base URL to pull images from when a build involves pulling
// one or more image(s) from a Library source.
func OptBuildLibraryPullBaseURL(libraryURL string) BuildOption {
	return func(bo *buildOptions) error {
		bo.libraryURL = libraryURL
		return nil
	}
}

// OptBuildContext instructs the Build Service to expose the build context with the specified
// digest during the build. The build context must be uploaded using UploadBuildContext.
func OptBuildContext(digest string) BuildOption {
	return func(bo *buildOptions) error {
		bo.contextDigest = digest
		return nil
	}
}

// OptBuildWorkingDirectory sets dir as the current working directory to include in the request.
func OptBuildWorkingDirectory(dir string) BuildOption {
	return func(bo *buildOptions) error {
		dir, err := filepath.Abs(dir)
		if err != nil {
			return err
		}

		bo.workingDir = dir
		return nil
	}
}

// Submit sends a build job to the Build Service. The context controls the lifetime of the request.
//
// By default, the built image will be pushed to an ephemeral location in the Library associated
// with the Remote Builder. To publish to a non-ephemeral location, consider using
// OptBuildLibraryRef.
//
// By default, the image will be built for the architecture returned by runtime.GOARCH. To override
// this behaviour, consider using OptBuildArchitecture.
//
// By default, if definition involves pulling one or more images from a Library reference that does
// not contain a hostname, they will be pulled from the Library associated with the Remote Builder.
// To override this behaviour, consider using OptBuildLibraryPullBaseURL.
//
// By default, local files referenced in the supplied definition will not be available on the Build
// Service. To expose local files, consider using OptBuildContext.
//
// The client includes the current working directory in the request, since the supplied definition
// may include paths that are relative to it. By default, the client attempts to derive the current
// working directory using os.Getwd(), falling back to "/" on error. To override this behaviour,
// consider using OptBuildWorkingDirectory.
func (c *Client) Submit(ctx context.Context, definition io.Reader, opts ...BuildOption) (*BuildInfo, error) {
	bo := buildOptions{
		arch:       runtime.GOARCH,
		workingDir: "/",
	}

	if dir, err := os.Getwd(); err == nil {
		bo.workingDir = dir
	}

	for _, opt := range opts {
		if err := opt(&bo); err != nil {
			return nil, fmt.Errorf("%w", err)
		}
	}

	raw, err := io.ReadAll(definition)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	v := struct {
		DefinitionRaw       []byte            `json:"definitionRaw"`
		LibraryRef          string            `json:"libraryRef"`
		LibraryURL          string            `json:"libraryURL,omitempty"`
		BuilderRequirements map[string]string `json:"builderRequirements,omitempty"`
		ContextDigest       string            `json:"contextDigest,omitempty"`
		WorkingDir          string            `json:"workingDir,omitempty"`
	}{
		DefinitionRaw: raw,
		LibraryRef:    bo.libraryRef,
		LibraryURL:    bo.libraryURL,
		ContextDigest: bo.contextDigest,
		WorkingDir:    bo.workingDir,
	}

	if bo.arch != "" {
		v.BuilderRequirements = map[string]string{
			"arch": bo.arch,
		}
	}

	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	ref := &url.URL{
		Path: "v1/build",
	}

	req, err := c.newRequest(ctx, http.MethodPost, ref, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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
