// Copyright (c) 2022, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

type archiver struct {
	fs       fs.FS
	w        *tar.Writer
	archived map[string]struct{}
}

// newArchiver returns an archiver that will write an archive to w.
func newArchiver(fsys fs.FS, w io.Writer) *archiver {
	return &archiver{
		fs:       fsys,
		w:        tar.NewWriter(w),
		archived: make(map[string]struct{}),
	}
}

var errUnsupportedType = errors.New("unsupported file type")

// writeEntry writes the named path from the file system to the archive.
func (ar *archiver) writeEntry(name string) (err error) {
	// If entry already exists, skip it.
	if _, ok := ar.archived[name]; ok {
		return nil
	}
	defer func() {
		if err == nil {
			ar.archived[name] = struct{}{}
		}
	}()

	// Get file info.
	fi, err := fs.Stat(ar.fs, name)
	if err != nil {
		return err
	}

	// Populate TAR header based on file info, and normalize name.
	h, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return err
	}
	h.Name = filepath.ToSlash(name)

	// Check that we're writing a supported type, and make any necessary adjustments.
	switch h.Typeflag {
	case tar.TypeReg:
		// Nothing to do.

	case tar.TypeSymlink:
		// Always follow symbolic links.
		h.Typeflag = tar.TypeReg
		h.Linkname = ""
		h.Size = fi.Size()

	case tar.TypeDir:
		// Normalize name.
		if !strings.HasSuffix(h.Name, "/") {
			h.Name += "/"
		}

	default:
		return fmt.Errorf("%v: %w (%v)", name, errUnsupportedType, h.Typeflag)
	}

	// Write TAR header.
	if err := ar.w.WriteHeader(h); err != nil {
		return err
	}

	// Write file contents, if applicable.
	if h.Typeflag == tar.TypeReg && h.Size > 0 {
		f, err := ar.fs.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(ar.w, f); err != nil {
			return err
		}
	}

	return nil
}

// walkDirFunc returns a WalkDirFunc that writes each path to ar.
func (ar *archiver) walkDirFunc() fs.WalkDirFunc {
	return func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == "." {
			return nil
		}

		return ar.writeEntry(path)
	}
}

// writeDirAll writes an entry for directory name to the archive, along with any necessary parents.
func (ar *archiver) writeDirAll(name string) error {
	if name == "." {
		return nil
	}

	if _, ok := ar.archived[name]; ok {
		return nil // Path already archived, no need to recurse.
	}

	// Recursively create parent(s).
	if err := ar.writeDirAll(path.Dir(name)); err != nil {
		return err
	}

	return ar.writeEntry(name)
}

// WriteFiles writes all files matching pattern from the file system to the archive. If the named
// path is a directory, its contents are recursively added using fs.WalkDir.
func (ar *archiver) WriteFiles(pattern string) error {
	names, err := fs.Glob(ar.fs, pattern)
	if err != nil {
		return err
	}

	if len(names) == 0 {
		return fmt.Errorf("%v: %w", pattern, fs.ErrNotExist)
	}

	for _, name := range names {
		// Ensure parent directory exists in archive.
		if err := ar.writeDirAll(path.Dir(name)); err != nil {
			return err
		}

		fi, err := fs.Stat(ar.fs, name)
		if err != nil {
			return err
		}

		// If name refers to a directory, walk it, adding entries. Otherwise, add a single entry.
		if fi.IsDir() {
			if err := fs.WalkDir(ar.fs, name, ar.walkDirFunc()); err != nil {
				return err
			}
		} else if err := ar.writeEntry(name); err != nil {
			return err
		}
	}

	return nil
}

// Close closes the archive.
func (ar *archiver) Close() error {
	return ar.w.Close()
}
