// Copyright (c) 2022-2025, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestValidateBuildSpec(t *testing.T) {
	tests := []struct {
		name        string
		buildSpec   string
		expectError bool
	}{
		{"DockerBuild", "docker://alpine:3", false},
		{"MalformedButValid", "docke//alpine:3", false},
		{"MalformedAgainButValidFilename", "docker:alpine:3", false},
		{"File", "alpine_3.def", false},
		{"FileScheme", "file://alpine_3.def", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseBuildSpec(tt.buildSpec)
			if (err != nil) != tt.expectError {
				t.Fatal(err)
			}
		})
	}
}

func withDefaults(t *testing.T) (*cobra.Command, *viper.Viper) {
	t.Helper()

	v := viper.New()

	cmd := &cobra.Command{
		Use: "testcmd",
		Run: func(*cobra.Command, []string) {},
	}

	addBuildCommandFlags(cmd)

	return cmd, v
}

func withUnrequiredPassphrase(t *testing.T) (*cobra.Command, *viper.Viper) {
	t.Helper()

	cmd, v := withDefaults(t)

	v.Set(keyPassphrase, "passphrase goes here")

	return cmd, v
}

func withKeyIdx(t *testing.T) (*cobra.Command, *viper.Viper) {
	t.Helper()

	cmd, v := withDefaults(t)

	v.Set(keyPassphrase, "passphrase goes here")

	v.Set(keySigningKeyIndex, 0)
	cmd.Flag(keySigningKeyIndex).Changed = true

	return cmd, v
}

func withFingerprint(t *testing.T) (*cobra.Command, *viper.Viper) {
	t.Helper()

	cmd, v := withDefaults(t)

	v.Set(keyPassphrase, "passphrase goes here")

	cmd.SetArgs([]string{"--fingerprint=xxx"})

	cmd.Flag(keyFingerprint).Changed = true

	return cmd, v
}

func TestValidateArgs(t *testing.T) {
	for _, tt := range []struct {
		name        string
		expectError error
		setupFunc   func(t *testing.T) (*cobra.Command, *viper.Viper)
	}{
		{"Defaults", nil, withDefaults},
		{"WithUnrequiredPassphrase", errPassphraseNotRequired, withUnrequiredPassphrase},
		{"ValidKeyIdx", nil, withKeyIdx},
		{"ValidFingerprint", nil, withFingerprint},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd, v := tt.setupFunc(t)

			if got, want := validateArgs(cmd, v), tt.expectError; got != want {
				t.Fatalf("Unexpected error: got %v, want %v", got, want)
			}
		})
	}
}
