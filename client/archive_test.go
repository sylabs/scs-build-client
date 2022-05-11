// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"bytes"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/sebdah/goldie/v2"
)

var testTime = time.Unix(1504657553, 0)

func Test_archiver_WriteFiles(t *testing.T) {
	tests := []struct {
		name    string
		fs      fs.FS
		paths   []string
		wantErr error
	}{
		{
			name:    "NotExist",
			fs:      fstest.MapFS{},
			paths:   []string{"a/b"},
			wantErr: fs.ErrNotExist,
		},
		{
			name: "NamedPipe",
			fs: fstest.MapFS{
				"a/b": &fstest.MapFile{
					Mode: 0o755 | fs.ModeNamedPipe,
				},
			},
			paths:   []string{"a/b"},
			wantErr: errUnsupportedType,
		},
		{
			name: "Device",
			fs: fstest.MapFS{
				"a/b": &fstest.MapFile{
					Mode: 0o755 | fs.ModeDevice,
				},
			},
			paths:   []string{"a/b"},
			wantErr: errUnsupportedType,
		},
		{
			name: "CharDevice",
			fs: fstest.MapFS{
				"a/b": &fstest.MapFile{
					Mode: 0o755 | fs.ModeDevice | fs.ModeCharDevice,
				},
			},
			paths:   []string{"a/b"},
			wantErr: errUnsupportedType,
		},
		{
			name: "Regular",
			fs: fstest.MapFS{
				"a": &fstest.MapFile{
					Mode:    0o755 | fs.ModeDir,
					ModTime: testTime,
				},
				"a/b": &fstest.MapFile{
					Data:    []byte("hello"),
					Mode:    0o755,
					ModTime: testTime,
				},
			},
			paths: []string{"a/b"},
		},
		{
			name: "Symlink",
			fs: fstest.MapFS{
				"a": &fstest.MapFile{
					Mode:    0o755 | fs.ModeDir,
					ModTime: testTime,
				},
				"a/b": &fstest.MapFile{
					Data:    []byte("hello"),
					Mode:    0o755 | fs.ModeSymlink,
					ModTime: testTime,
				},
			},
			paths: []string{"a/b"},
		},
		{
			name: "WalkDirRoot",
			fs: fstest.MapFS{
				"a": &fstest.MapFile{
					Mode:    0o755 | fs.ModeDir,
					ModTime: testTime,
				},
				"a/b": &fstest.MapFile{
					Data:    []byte("hello"),
					Mode:    0o755,
					ModTime: testTime,
				},
				"a/c": &fstest.MapFile{
					Data:    []byte("goodbye"),
					Mode:    0o755,
					ModTime: testTime,
				},
			},
			paths: []string{"."},
		},
		{
			name: "WalkDirPath",
			fs: fstest.MapFS{
				"a": &fstest.MapFile{
					Mode:    0o755 | fs.ModeDir,
					ModTime: testTime,
				},
				"a/b": &fstest.MapFile{
					Data:    []byte("hello"),
					Mode:    0o755,
					ModTime: testTime,
				},
				"a/c": &fstest.MapFile{
					Data:    []byte("goodbye"),
					Mode:    0o755,
					ModTime: testTime,
				},
			},
			paths: []string{"a"},
		},
		{
			name: "FileGlob",
			fs: fstest.MapFS{
				"a": &fstest.MapFile{
					Mode:    0o755 | fs.ModeDir,
					ModTime: testTime,
				},
				"a/b": &fstest.MapFile{
					Data:    []byte("hello"),
					Mode:    0o755,
					ModTime: testTime,
				},
				"a/c": &fstest.MapFile{
					Data:    []byte("goodbye"),
					Mode:    0o755,
					ModTime: testTime,
				},
			},
			paths: []string{"a/*"},
		},
		{
			name: "DirGlob",
			fs: fstest.MapFS{
				"a": &fstest.MapFile{
					Mode:    0o755 | fs.ModeDir,
					ModTime: testTime,
				},
				"a/b": &fstest.MapFile{
					Data:    []byte("hello"),
					Mode:    0o755,
					ModTime: testTime,
				},
				"c": &fstest.MapFile{
					Mode:    0o755 | fs.ModeDir,
					ModTime: testTime,
				},
				"c/b": &fstest.MapFile{
					Data:    []byte("goodbye"),
					Mode:    0o755,
					ModTime: testTime,
				},
			},
			paths: []string{"*/b"},
		},
		{
			name: "Duplicates",
			fs: fstest.MapFS{
				"a": &fstest.MapFile{
					Mode:    0o755 | fs.ModeDir,
					ModTime: testTime,
				},
				"a/b": &fstest.MapFile{
					Mode:    0o755 | fs.ModeDir,
					ModTime: testTime,
				},
				"a/b/c": &fstest.MapFile{
					Data:    []byte("hello"),
					Mode:    0o755,
					ModTime: testTime,
				},
			},
			paths: []string{
				"a/b/c",
				"a/b",
				"a",
				"*",
				".",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bytes.Buffer{}

			ar := newArchiver(tt.fs, &b)

			for _, path := range tt.paths {
				if got, want := ar.WriteFiles(path), tt.wantErr; !errors.Is(got, want) {
					t.Errorf("got error %v, want %v", got, want)
				}
			}

			if err := ar.Close(); err != nil {
				t.Fatal(err)
			}

			if tt.wantErr == nil {
				g := goldie.New(t, goldie.WithTestNameForDir(true))
				g.Assert(t, tt.name, b.Bytes())
			}
		})
	}
}
