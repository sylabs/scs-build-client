// Copyright (c) 2022, Sylabs, Inc. All rights reserved.

package endpoints

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

const frontendConfigPath = "assets/config/config.prod.json"

var errServerMisconfigured = errors.New("remote server is misconfigured")

type uri struct {
	URI string `json:"uri"`
}

type frontendConfig struct {
	LibraryAPI uri `json:"libraryAPI"`
	BuildAPI   uri `json:"builderAPI"`
}

func getFrontendConfigURL(frontendURL string) string {
	url := frontendURL
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	return url + frontendConfigPath
}

func GetFrontendConfig(ctx context.Context, httpClient *http.Client, frontendURL string) (*frontendConfig, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getFrontendConfigURL(frontendURL), nil)
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var cfg frontendConfig
	if err := json.NewDecoder(res.Body).Decode(&cfg); err != nil {
		return nil, err
	}

	if cfg.LibraryAPI.URI == "" || cfg.BuildAPI.URI == "" {
		return nil, errServerMisconfigured
	}

	return &cfg, nil
}
