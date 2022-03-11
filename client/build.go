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
	"runtime"

	jsonresp "github.com/sylabs/json-resp"
)

type buildOptions struct {
	libraryRef string
	arch       string
	libraryURL string
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
func (c *Client) Submit(ctx context.Context, definition io.Reader, opts ...BuildOption) (BuildInfo, error) {
	bo := buildOptions{
		arch: runtime.GOARCH,
	}

	for _, opt := range opts {
		if err := opt(&bo); err != nil {
			return BuildInfo{}, fmt.Errorf("%w", err)
		}
	}

	raw, err := io.ReadAll(definition)
	if err != nil {
		return BuildInfo{}, fmt.Errorf("%w", err)
	}

	v := struct {
		DefinitionRaw       []byte            `json:"definitionRaw"`
		LibraryRef          string            `json:"libraryRef"`
		LibraryURL          string            `json:"libraryURL,omitempty"`
		BuilderRequirements map[string]string `json:"builderRequirements,omitempty"`
	}{
		DefinitionRaw: raw,
		LibraryRef:    bo.libraryRef,
		LibraryURL:    bo.libraryURL,
	}

	if bo.arch != "" {
		v.BuilderRequirements = map[string]string{
			"arch": bo.arch,
		}
	}

	b, err := json.Marshal(v)
	if err != nil {
		return BuildInfo{}, fmt.Errorf("%w", err)
	}

	ref := &url.URL{
		Path: "v1/build",
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
