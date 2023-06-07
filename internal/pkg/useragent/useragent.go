// Copyright (c) 2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package useragent

import "fmt"

var value string

func Init(version string) {
	value = fmt.Sprintf("scs-build/%v", version)
}

func Value() string {
	return value
}
