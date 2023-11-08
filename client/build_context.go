// Copyright (c) 2022-2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
)

// writeArchive writes a compressed archive containing paths read from fsys to w.
//
// Paths must be specified in the rootless format specified by the io/fs package. If a path
// contains a glob, it will be evaluated as per fs.Glob. If a path specifies a directory, its
// contents will be walked as per fs.WalkDir.
func writeArchive(w io.Writer, fsys fs.FS, paths []string) error {
	gw := gzip.NewWriter(w)
	defer gw.Close()

	ar := newArchiver(fsys, gw)
	defer ar.Close()

	for _, path := range paths {
		if err := ar.WriteFiles(path); err != nil {
			return err
		}
	}

	return nil
}

var errContextAlreadyPresent = errors.New("build context already present")

// getBuildContextUploadLocation obtains an upload location for a build context.
//
// If errContextAlreadyPresent is returned, (re)upload of build context is not required.
func (c *Client) getBuildContextUploadLocation(ctx context.Context, size int64, digest string) (*url.URL, error) {
	ref := &url.URL{
		Path: "v1/build-context",
	}

	body := struct {
		Size   int64  `json:"size"`
		Digest string `json:"digest"`
	}{
		Size:   size,
		Digest: digest,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, ref, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.buildContextHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 { // non-2xx status code
		return nil, fmt.Errorf("%w", errorFromResponse(res))
	}

	if res.Header.Get("Location") == "" {
		// "Location" header is not present; build context does not need to be uploaded
		return nil, errContextAlreadyPresent
	}

	return url.Parse(res.Header.Get("Location"))
}

// putBuildContext uploads the build context read from r to the specified location.
func (c *Client) putBuildContext(ctx context.Context, loc *url.URL, r io.Reader, size int64) error {
	req, err := c.newRequest(ctx, http.MethodPut, loc, r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Del("Authorization")

	req.ContentLength = size

	res, err := c.buildContextHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 {
		return fmt.Errorf("%w", errorFromResponse(res))
	}
	return nil
}

// uploadBuildContext generates an archive in rw containing the files at the specified paths in
// fsys, and uploads it to the Build Service.
//
// Paths must be specified in the rootless format specified by the io/fs package. If a path
// contains a glob, it will be evaluated as per fs.Glob. If a path specifies a directory, its
// contents will be walked as per fs.WalkDir.
func (c *Client) uploadBuildContext(ctx context.Context, rw io.ReadWriteSeeker, fsys fs.FS, paths []string) (digest string, err error) {
	// Write a compressed archive and accumulate its digest.
	h := sha256.New()
	if err := writeArchive(io.MultiWriter(rw, h), fsys, paths); err != nil {
		return "", fmt.Errorf("failed to write archive: %w", err)
	}

	// Obtain size of build context.
	size, err := rw.Seek(0, io.SeekCurrent)
	if err != nil {
		return "", fmt.Errorf("failed to seek: %w", err)
	}

	// Calculate digest of build context.
	digest = fmt.Sprintf("sha256.%x", h.Sum(nil))

	// Get the build context upload location.
	loc, err := c.getBuildContextUploadLocation(ctx, size, digest)
	if err != nil {
		if errors.Is(err, errContextAlreadyPresent) {
			return digest, nil
		}
		return "", fmt.Errorf("failed to get build context upload location: %w", err)
	}

	// Seek to the beginning of the build context file.
	if _, err := rw.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to seek: %w", err)
	}

	// Upload build context.
	if err := c.putBuildContext(ctx, loc, rw, size); err != nil {
		return "", fmt.Errorf("failed to upload build context: %w", err)
	}

	return digest, nil
}

type uploadBuildContextOptions struct {
	fsys fs.FS
}

type UploadBuildContextOption func(*uploadBuildContextOptions) error

// optUploadBuildContextFS sets fsys as the source filesystem to use when constructing the build
// context archive.
func optUploadBuildContextFS(fsys fs.FS) UploadBuildContextOption {
	return func(uo *uploadBuildContextOptions) error {
		uo.fsys = fsys
		return nil
	}
}

var errNoPathsSpecified = errors.New("no paths specified for build context")

// UploadBuildContext generates an archive containing the files at the specified paths, and uploads
// it to the Build Service. When the build context is no longer required, DeleteBuildContext should
// be called to notify the Build Service.
//
// Paths must be specified in the rootless format specified by the io/fs package. If a path
// contains a glob, it will be evaluated as per fs.Glob. If a path specifies a directory, its
// contents will be walked as per fs.WalkDir.
func (c *Client) UploadBuildContext(ctx context.Context, paths []string, opts ...UploadBuildContextOption) (digest string, err error) {
	uo := uploadBuildContextOptions{
		fsys: os.DirFS("/"),
	}

	for _, opt := range opts {
		if err := opt(&uo); err != nil {
			return "", fmt.Errorf("%w", err)
		}
	}

	if len(paths) == 0 {
		return "", errNoPathsSpecified
	}

	f, err := os.CreateTemp("", "scs-build-context-*")
	if err != nil {
		return "", fmt.Errorf("%w", err)
	}
	defer os.Remove(f.Name())

	return c.uploadBuildContext(ctx, f, uo.fsys, paths)
}

type deleteBuildContextOptions struct{}

type DeleteBuildContextOption func(*deleteBuildContextOptions) error

// DeleteBuildContext deletes the build context with the specified digest from the Build Service.
func (c *Client) DeleteBuildContext(ctx context.Context, digest string, opts ...DeleteBuildContextOption) error {
	do := deleteBuildContextOptions{}

	for _, opt := range opts {
		if err := opt(&do); err != nil {
			return fmt.Errorf("%w", err)
		}
	}

	ref := &url.URL{
		Path: "v1/build-context/" + digest,
	}

	req, err := c.newRequest(ctx, http.MethodDelete, ref, nil)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	res, err := c.buildContextHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 { // non-2xx status code
		return fmt.Errorf("%w", errorFromResponse(res))
	}

	return nil
}
