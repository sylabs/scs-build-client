// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package endpoints

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const frontendConfigPath = "assets/config/config.prod.json"

var errServerMisconfigured = errors.New("remote server is misconfigured")

type uri struct {
	URI string `json:"uri"`
}

type FrontendConfig struct {
	LibraryAPI uri `json:"libraryAPI"`
	BuildAPI   uri `json:"builderAPI"`
}

func getFrontendConfigURL(frontendURL string) string {
	return fmt.Sprintf("%v/%v", strings.TrimSuffix(frontendURL, "/"), frontendConfigPath)
}

func GetFrontendConfig(ctx context.Context, skipVerify bool, frontendURL string) (*FrontendConfig, error) {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: skipVerify}

	httpClient := &http.Client{Transport: tr}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getFrontendConfigURL(frontendURL), nil)
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 { // non-2xx status code
		return nil, fmt.Errorf("error getting configuration (HTTP status code %d)", res.StatusCode)
	}

	var cfg FrontendConfig
	if err := json.NewDecoder(res.Body).Decode(&cfg); err != nil {
		return nil, err
	}

	if cfg.LibraryAPI.URI == "" || cfg.BuildAPI.URI == "" {
		return nil, errServerMisconfigured
	}

	return &cfg, nil
}
