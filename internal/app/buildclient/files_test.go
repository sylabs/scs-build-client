// Copyright (c) 2018-2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var defFileData = []byte(`{"data":{"header":{"bootstrap":"docker","from":"alpine"},"imageData":{"metadata":null,"labels":{},"imageScripts":{"help":{"args":"","script":""},"environment":{"args":"","script":""},"runScript":{"args":"","script":""},"test":{"args":"","script":""},"startScript":{"args":"","script":""}}},"buildData":{"files":[{"args":"","files":[{"source":"./file.txt","destination":"/testfile.txt"},{"source":"anotherfile.txt","destination":"/anotherfile.txt"},{"source":"/a/b/c/d/*.txt","destination":"/e/"},{"source":"../z","destination":"/z/"}]}],"buildScripts":{"pre":{"args":"","script":""},"setup":{"args":"","script":""},"post":{"args":"","script":""},"test":{"args":"","script":""}}},"customData":null,"raw":"Qm9vdHN0cmFwOiBkb2NrZXIKRnJvbTogYWxwaW5lCgolZmlsZXMKICAuL2ZpbGUudHh0IC90ZXN0ZmlsZS50eHQKICBhbm90aGVyZmlsZS50eHQgL2Fub3RoZXJmaWxlLnR4dAogIC9hL2IvYy9kLyoudHh0IC9lLwogIC4uL3ogL3ovCg==","appOrder":[]}}`)

func Test_Stage(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{"basic", "test", ""},
		{"escapeArgs", "\nfirst\nsecond", "second"},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			f := files{Args: tt.args}

			if got, want := f.Stage(), tt.want; got != want {
				t.Fatalf("got: %v, want: %v", got, want)
			}
		})
	}
}

func Test_SourcePath(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"basic", "test"},
		{"slash", "/"},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			ft := FileTransport{Src: tt.source}
			got, err := ft.SourcePath()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			log.Println(got)

			if tt.source == "/" && got != "." {
				t.Fatalf("Unexpected results: %v", got)
			}
		})
	}
}

func TestExtractFiles(t *testing.T) {
	// Create test build server
	r := http.NewServeMux()
	r.HandleFunc("/v1/convert-def-file", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if _, err := w.Write(defFileData); err != nil {
			t.Fatalf("HTTP write error: %v", err)
		}
	})
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Create test frontend server
	feRouter := http.NewServeMux()
	feRouter.HandleFunc("/assets/config/config.prod.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		res := fmt.Sprintf("{\"builderAPI\": {\"uri\": \"%v\"}, \"libraryAPI\": {\"uri\": \"http://invalidserver\"}}}", ts.URL)

		if _, err := w.Write([]byte(res)); err != nil {
			t.Fatalf("error writing HTTP response: %v", err)
		}
	})
	tsFE := httptest.NewServer(feRouter)
	defer tsFE.Close()

	app, err := New(context.Background(), &Config{
		URL: tsFE.URL,
	})
	if err != nil {
		t.Fatalf("error initializing app: %v", err)
	}

	// Extract files referenced by def file; rewrite all paths to be relative to current working directory
	files, err := app.getFiles(context.Background(), nil)
	if err != nil {
		t.Fatalf("%v", err)
	}

	if got, want := len(files), 4; got != want {
		t.Fatalf("unexpected number of files: got %v, want %v", got, want)
	}

	curwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Build expected results based on current working directory and expected results
	expectedFiles := []string{
		strings.TrimPrefix(filepath.Join(curwd, "file.txt"), "/"),
		strings.TrimPrefix(filepath.Join(curwd, "anotherfile.txt"), "/"),
		"a/b/c/d/*.txt",
		strings.TrimPrefix(filepath.Clean(filepath.Join(curwd, "../z")), "/"),
	}

	if !reflect.DeepEqual(files, expectedFiles) {
		t.Fatalf("unexpected results: got %v, want %v", files, expectedFiles)
	}
}
