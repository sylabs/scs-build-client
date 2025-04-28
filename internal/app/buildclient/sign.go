// Copyright (c) 2023-2025, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"github.com/sylabs/sif/v2/pkg/integrity"
	"github.com/sylabs/sif/v2/pkg/sif"
)

func sign(fileName string, opts ...integrity.SignerOpt) error {
	f, err := sif.LoadContainerFromPath(fileName)
	if err != nil {
		return err
	}

	defer func() {
		if err := f.UnloadContainer(); err != nil {
			return
		}
	}()

	is, err := integrity.NewSigner(f, opts...)
	if err != nil {
		return err
	}

	return is.Sign()
}
