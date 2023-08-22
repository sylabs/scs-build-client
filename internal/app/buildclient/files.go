// Copyright (c) 2018-2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	jsonresp "github.com/sylabs/json-resp"
)

// definition defines subset of def file
type definition struct {
	BuildData buildData `json:"buildData"`
}

type buildData struct {
	Files []files `json:"files"`
}

type files struct {
	Args  string          `json:"args"`
	Files []FileTransport `json:"files"`
}

func (f files) Stage() string {
	// Trim comments from args.
	cleanArgs := strings.SplitN(f.Args, "#", 2)[0]

	// If "stage <name>", return "<name>".
	if args := strings.Fields(cleanArgs); len(args) == 2 && args[0] != "stage" {
		return args[1]
	}

	return ""
}

type FileTransport struct {
	Src string `json:"source"`
	Dst string `json:"destination"`
}

// SourcePath returns the source path in the format as specified by the io/fs package.
func (ft FileTransport) SourcePath() (string, error) {
	path, err := filepath.Abs(ft.Src)
	if err != nil {
		return "", err
	}

	// Paths are slash-separated.
	path = filepath.ToSlash(path)

	// Special case: the root directory is named ".".
	if path == "/" {
		return ".", nil
	}

	// Paths must not start with a slash.
	return strings.TrimPrefix(path, "/"), nil
}

// SourceFiles extracts source file names for parsed def file
func (d definition) SourceFiles() (result []string) {
	for _, e := range d.BuildData.Files {
		for _, f := range e.Files {
			result = append(result, f.Src)
		}
	}
	return
}

// parseDefinition calls /v1/convert-def-file API to parse definition file (read from 'r'),
// returns parsed definition
func (app *App) parseDefinition(ctx context.Context, r io.Reader) (definition, error) {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: app.skipTLSVerify}
	httpClient := &http.Client{Transport: tr}

	loc := fmt.Sprintf("%v/%v", strings.TrimSuffix(app.buildURL, "/"), "v1/convert-def-file")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loc, r)
	if err != nil {
		return definition{}, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", app.libraryClient.AuthToken))

	res, err := httpClient.Do(req)
	if err != nil {
		return definition{}, err
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 { // non-2xx status code
		err = fmt.Errorf("build server error (HTTP status %d)", res.StatusCode)
		return definition{}, err
	}

	d := definition{}
	if err = jsonresp.ReadResponse(res.Body, &d); err != nil {
		err = fmt.Errorf("%w", err)
	}
	return d, err
}

// ExtractFiles makes request to remote build server to parse specified def file and returns
// files referenced in '%files' section(s)
func (app *App) getFiles(ctx context.Context, r io.Reader) (files []string, err error) {
	d, err := app.parseDefinition(ctx, r)
	if err != nil {
		err = fmt.Errorf("def file parse error: %w", err)
		return
	}

	for _, f := range d.BuildData.Files {
		if f.Stage() != "" {
			// ignore files from stages
			continue
		}

		for _, ft := range f.Files {
			updFileName, err := ft.SourcePath()
			if err != nil {
				err = fmt.Errorf("error parsing def file: %w", err)
				return []string{}, err
			}

			files = append(files, updFileName)
		}
	}
	return
}
