// Copyright (c) 2022-2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import "testing"

func TestParseLibraryrefArg(t *testing.T) {
	tests := []struct {
		name               string
		libraryRef         string
		expectedLibraryRef string
		expectError        bool
		expectHost         string
		expectPath         string
	}{
		{"Valid", "library://user/default/alpine:3", "library:user/default/alpine:3", false, "", ""},
		{"WithHost", "library://host/user/default/alpine:3", "library:user/default/alpine:3", false, "host", ""},
		{"MalformedLibraryPrefix", "librar://user/default/alpine:3", "", true, "", ""},
		{"File", "alpine_3.sif", "", false, "", "alpine_3.sif"},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			p, err := parseLibraryrefArg(tt.libraryRef)
			if (err != nil) != tt.expectError {
				t.Fatal(err)
			}

			if err != nil {
				return
			}

			if tt.expectHost != "" {
				if p.Host() != tt.expectHost {
					t.Fatalf("Got: %v, Want: %v", p.Host(), tt.expectHost)
				}
			}

			if tt.expectPath != "" {
				if p.FileName() != tt.expectPath {
					t.Fatalf("Got: %v, Want: %v", p.FileName(), tt.expectPath)
				}
			}

			if p.Ref() != nil {
				if p.Ref().String() != tt.expectedLibraryRef {
					t.Fatalf("Got: %v, Want: %v", p.Ref(), tt.libraryRef)
				}
			}
		})
	}
}
