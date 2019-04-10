// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the LICENSE.md file
// distributed with the sources of this project regarding your rights to use or distribute this
// software.

package client

import "github.com/sylabs/scs-library-client/client"

// TODO - remove this and have singularity use scs-library-client itself
func (c *Client) downloadImage(imagePath string, libraryRef string, force bool) error {
	lc, err := client.NewClient(&client.Config{
		AuthToken: c.AuthToken,
		BaseURL:   c.LibraryURL.String(),
	})
	if err != nil {
		return err
	}

	// TODO: add progess callback
	return client.DownloadImage(lc, imagePath, libraryRef, force, nil)
}
