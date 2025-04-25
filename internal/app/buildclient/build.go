// Copyright (c) 2022-2025, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sylabs/scs-build-client/internal/pkg/useragent"
	"github.com/sylabs/sif/v2/pkg/integrity"
)

const (
	keyAccessToken       = "auth-token"
	keySkipTLSVerify     = "skip-verify"
	keyArch              = "arch"
	keyFrontendURL       = "url"
	keyForceOverwrite    = "force"
	keySign              = "sign"
	keySigningKeyIndex   = "keyidx"
	keyFingerprint       = "fingerprint"
	keyKeyring           = "keyring"
	keyPassphrase        = "passphrase"
	keyPrivateSigningKey = "key"
)

var buildCmd = &cobra.Command{
	Use:   "build [flags] <build spec> <image path>",
	Short: "Perform remote build on Singularity Container Services (https://cloud.sylabs.io) or Singularity Enterprise",
	Args:  cobra.MinimumNArgs(1),
	RunE:  executeBuildCmd,
	Example: `
  Build and push artifact to cloud library:

      scs-build build alpine.def library:user/project/image:tag

  Build and push artifact to Singularity Enterprise:

      scs-build build alpine.def library://cloud.enterprise.local/user/project/image:tag

  Build local artifact:

      scs-build build docker://alpine alpine_latest.sif

  Build local artifact on Singularity Enterprise:

      scs-build build --url https://cloud.enterprise.local --skip-verify docker://alpine alpine_latest.sif

  Build ephemeral artifact:

      scs-build build alpine.def

  Note: ephemeral artifacts are short-lived and are usually deleted within 24 hours.

  Using --sign will enable automatic PGP signing. Use '--sign --key FILE' to sign with private key.`,
}

var (
	errSigningNotSupported   = errors.New("build and sign ephemeral image is not supported")
	errPassphraseNotRequired = errors.New("--passphrase only effective when PGP signing enabled")
)

// addBuildCommandFlags configures flags for 'build' subcommand.
func addBuildCommandFlags(cmd *cobra.Command) {
	cmd.Flags().String(keyAccessToken, "", "Access token")
	cmd.Flags().Bool(keySkipTLSVerify, false, "Skip SSL/TLS certificate verification")
	cmd.Flags().StringSlice(keyArch, []string{runtime.GOARCH}, "Requested build architecture")
	cmd.Flags().String(keyFrontendURL, "", "Singularity Container Services or Singularity Enterprise URL")
	cmd.Flags().Bool(keyForceOverwrite, false, "Overwrite image file if it exists")
	cmd.Flags().Bool(keySign, false, "Automatically sign image after build")
	cmd.Flags().IntP(keySigningKeyIndex, "k", -1, "PGP private key to use")
	cmd.Flags().String(keyFingerprint, "", "Fingerprint for PGP key to sign with")
	cmd.Flags().String(keyKeyring, "", "Full path to PGP keyring")
	cmd.Flags().String(keyPassphrase, "", "Passphrase for PGP key")
	cmd.Flags().String(keyPrivateSigningKey, "", "Private key for signing")

	cmd.MarkFlagsMutuallyExclusive(keySigningKeyIndex, keyFingerprint, keyPrivateSigningKey)
	cmd.MarkFlagsMutuallyExclusive(keyKeyring, keyPrivateSigningKey)
	cmd.MarkFlagsMutuallyExclusive(keyPassphrase, keyPrivateSigningKey)
	cmd.MarkFlagsMutuallyExclusive(keyFingerprint, keyPrivateSigningKey)
}

func AddBuildCommand(rootCmd *cobra.Command) {
	addBuildCommandFlags(buildCmd)

	rootCmd.AddCommand(buildCmd)
}

func getConfig(cmd *cobra.Command) (*viper.Viper, error) {
	v := viper.New()

	v.SetEnvPrefix("sylabs")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	return v, v.BindPFlags(cmd.Flags())
}

func validateArgs(cmd *cobra.Command, v *viper.Viper) error {
	// Error if passphrase has been set and signing key index and fingerprint have NOT been set.
	if v.GetString(keyPassphrase) != "" &&
		!cmd.Flag(keySigningKeyIndex).Changed &&
		!cmd.Flag(keyFingerprint).Changed {
		return errPassphraseNotRequired
	}

	return nil
}

func executeBuildCmd(cmd *cobra.Command, args []string) error {
	// Get command-line/envvars
	v, err := getConfig(cmd)
	if err != nil {
		return fmt.Errorf("error getting config: %w", err)
	}

	if err := validateArgs(cmd, v); err != nil {
		return err
	}

	signing := v.GetString(keyPassphrase) != "" ||
		v.GetInt(keySigningKeyIndex) != -1 ||
		v.GetString(keyFingerprint) != "" ||
		v.GetBool(keySign)

	var signerOpts []integrity.SignerOpt

	if signing {
		fmt.Printf("Build artifacts will be automatically signed\n")

		signerOpts, err = parseSigningOpts(v)
		if err != nil {
			return fmt.Errorf("error parsing signing opts: %w", err)
		}
	}

	var libraryRef string

	if len(args) > 1 {
		libraryRef = args[1]
	} else {
		if len(args) == 1 && signing {
			return errSigningNotSupported
		}
	}

	buildSpec, err := parseBuildSpec(args[0])
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app, err := New(ctx, &Config{
		URL:           v.GetString(keyFrontendURL),
		AuthToken:     v.GetString(keyAccessToken),
		BuildSpec:     buildSpec,
		LibraryRef:    libraryRef,
		SkipTLSVerify: v.GetBool(keySkipTLSVerify),
		Force:         v.GetBool(keyForceOverwrite),
		UserAgent:     useragent.Value(),
		ArchsToBuild:  v.GetStringSlice(keyArch),
		SignerOpts:    signerOpts,
	})
	if err != nil {
		return fmt.Errorf("application init error: %w", err)
	}

	// set up signal handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Fprintf(os.Stderr, "Shutting down due to signal: %v\n", <-c)
		cancel()
	}()

	return app.Run(ctx)
}

var errInvalidBuildSpec = errors.New("invalid build spec")

// parseBuildSpec validates buildspec argument.
func parseBuildSpec(buildSpec string) (string, error) {
	u, err := url.Parse(buildSpec)
	if err != nil {
		return buildSpec, nil
	}

	if u.Scheme == "file" {
		return strings.TrimPrefix(buildSpec, "file://"), nil
	}

	if u.Scheme != "" && u.Scheme != "docker" {
		return "", errInvalidBuildSpec
	}

	return buildSpec, nil
}

func parseSigningOpts(v *viper.Viper) ([]integrity.SignerOpt, error) {
	// Parse flags to determine signing configuration
	opts := []integrity.SignerOpt{}

	if privateSigningKey := v.GetString(keyPrivateSigningKey); privateSigningKey != "" {
		// Use private key for signing
		ss, err := signature.LoadSignerFromPEMFile(privateSigningKey, crypto.SHA256, cryptoutils.GetPasswordFromStdIn)
		if err != nil {
			return nil, fmt.Errorf("error initializing private key signer: %w", err)
		}

		return append(opts, integrity.OptSignWithSigner(ss)), nil
	}

	// Fallback to PGP signing
	s, err := parsePGPSignerOpts(v)
	if err != nil {
		return nil, err
	}

	pgpSignerOpts, err := getPGPSignerOpts(s...)
	if err != nil {
		return nil, err
	}

	return append(opts, pgpSignerOpts...), nil
}
