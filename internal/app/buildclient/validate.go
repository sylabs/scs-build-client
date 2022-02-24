// Copyright (c) 2022, Sylabs, Inc. All rights reserved.

package buildclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	jsonresp "github.com/sylabs/json-resp"
	"github.com/sylabs/singularity/pkg/build/types"
)

var errBuildDefValidationError = errors.New("error validating build definition")

func (app *App) validateBuildDef(ctx context.Context, def []byte) error {
	validateURL := *app.buildClient.BaseURL
	validateURL.Path = "/v1/convert-def-file"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, validateURL.String(), bytes.NewReader(def))
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", app.buildClient.AuthToken))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.ContentLength > 0 {
			return jsonresp.ReadError(resp.Body)
		}
		return errBuildDefValidationError
	}

	var validateResponse types.Definition

	if err := json.NewDecoder(resp.Body).Decode(&validateResponse); err != nil {
		return errBuildDefValidationError
	}

	return nil
}
