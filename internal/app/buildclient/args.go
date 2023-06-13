// Copyright (c) 2022-2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"fmt"
	"net/url"
	"strings"

	library "github.com/sylabs/scs-library-client/client"
)

func hasLibraryScheme(libraryRef string) bool {
	return strings.HasPrefix(libraryRef, library.Scheme+":")
}

type libraryrefArgParser struct {
	host     string
	ref      *library.Ref
	filename string
}

func (p libraryrefArgParser) Host() string      { return p.host }
func (p libraryrefArgParser) Ref() *library.Ref { return p.ref }
func (p libraryrefArgParser) FileName() string  { return p.filename }

func parseLibraryrefArg(libraryRef string) (libraryrefArgParser, error) {
	if hasLibraryScheme(libraryRef) {
		ref, err := library.ParseAmbiguous(libraryRef)
		if err != nil {
			return libraryrefArgParser{}, fmt.Errorf("malformed library ref: %w", err)
		}

		p := libraryrefArgParser{}

		if ref.Host != "" {
			// Ref contains a host. Note this to determine the front end URL, but don't include it
			// in the LibraryRef, since the Build Service expects a hostless format.
			p.host = ref.Host
			ref.Host = ""
		}

		p.ref = ref

		return p, nil
	}

	// Parse as URI
	u, err := url.Parse(libraryRef)
	if err != nil {
		return libraryrefArgParser{filename: libraryRef}, nil
	}
	if u.Scheme == "file" || u.Scheme == "" {
		return libraryrefArgParser{filename: strings.TrimPrefix(libraryRef, "file://")}, nil
	}

	return libraryrefArgParser{}, errMalformedLibraryRef
}
