// Copyright (c) 2022-2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	build "github.com/sylabs/scs-build-client/client"
	"github.com/sylabs/scs-build-client/internal/pkg/endpoints"
	library "github.com/sylabs/scs-library-client/client"
)

const defaultFrontendURL = "https://cloud.sylabs.io"

// Config contains set up for application
type Config struct {
	URL           string
	AuthToken     string
	DefFileName   string
	SkipTLSVerify bool
	LibraryRef    string
	Force         bool
	UserAgent     string
	ArchsToBuild  []string
	Key           string
	KeyIdx        int
	Fingerprint   string
	Keyring       string
	Signing       bool
	Passphrase    string
}

// App represents the application instance
type App struct {
	buildClient   *build.Client
	libraryClient *library.Client
	buildSpec     string
	LibraryRef    *library.Ref
	dstFileName   string
	force         bool
	buildURL      string
	skipTLSVerify bool
	archsToBuild  []string
	key           string
	keyIdx        int
	fingerprint   string
	keyring       string
	signing       bool
	passphrase    string
}

var errNoBuildContextFiles = errors.New("no files referenced in build definition")

// New creates new application instance
func New(ctx context.Context, cfg *Config) (*App, error) {
	app := &App{
		buildSpec:     cfg.DefFileName,
		force:         cfg.Force,
		skipTLSVerify: cfg.SkipTLSVerify,
		archsToBuild:  cfg.ArchsToBuild,
		key:           cfg.Key,
		keyIdx:        cfg.KeyIdx,
		fingerprint:   cfg.Fingerprint,
		keyring:       cfg.Keyring,
		signing:       cfg.Signing,
		passphrase:    cfg.Passphrase,
	}
	var libraryRefHost string

	// Parse/validate image spec (local file or library ref)
	if strings.HasPrefix(cfg.LibraryRef, library.Scheme+":") {
		ref, err := library.ParseAmbiguous(cfg.LibraryRef)
		if err != nil {
			return nil, fmt.Errorf("malformed library ref: %w", err)
		}

		if ref.Host != "" {
			// Ref contains a host. Note this to determine the front end URL, but don't include it
			// in the LibraryRef, since the Build Service expects a hostless format.
			libraryRefHost = ref.Host
			ref.Host = ""
		}

		app.LibraryRef = ref
	} else if cfg.LibraryRef != "" {
		// Parse as URL
		ref, err := url.Parse(cfg.LibraryRef)
		if err != nil {
			return nil, fmt.Errorf("error parsing %v as URL: %w", cfg.LibraryRef, err)
		}
		if ref.Scheme != "file" && ref.Scheme != "" {
			return nil, fmt.Errorf("unsupported library ref scheme %v", ref.Scheme)
		}
		app.dstFileName = ref.Path
	}

	// Determine frontend URL either from library ref, if provided or url, if provided, or default.
	feURL, err := getFrontendURL(cfg.URL, libraryRefHost)
	if err != nil {
		return nil, err
	}

	// Initialize build & library clients
	feCfg, err := endpoints.GetFrontendConfig(ctx, cfg.SkipTLSVerify, feURL)
	if err != nil {
		return nil, err
	}
	app.buildURL = feCfg.BuildAPI.URI

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.SkipTLSVerify}

	app.buildClient, err = build.NewClient(
		build.OptBaseURL(feCfg.BuildAPI.URI),
		build.OptBearerToken(cfg.AuthToken),
		build.OptUserAgent(cfg.UserAgent),
		build.OptHTTPClient(&http.Client{Transport: tr}),
	)
	if err != nil {
		return nil, fmt.Errorf("error initializing build client: %w", err)
	}

	app.libraryClient, err = library.NewClient(&library.Config{
		BaseURL:    feCfg.LibraryAPI.URI,
		AuthToken:  cfg.AuthToken,
		HTTPClient: &http.Client{Transport: tr},
		UserAgent:  cfg.UserAgent,
	})
	if err != nil {
		return nil, fmt.Errorf("error initializing library client: %w", err)
	}

	return app, nil
}

// getFrontendURL determines the front end value based on urlOverride and/or libraryRefHost.
func getFrontendURL(urlOverride, libraryRefHost string) (string, error) {
	if urlOverride != "" {
		if libraryRefHost == "" {
			return urlOverride, nil
		}

		u, err := url.Parse(urlOverride)
		if err != nil {
			return "", err
		}

		if u.Host != libraryRefHost {
			return "", errors.New("conflicting arguments")
		}

		return urlOverride, nil
	}

	if libraryRefHost != "" {
		return "https://" + libraryRefHost, nil
	}

	return defaultFrontendURL, nil
}

// uploadBuildContext parses definition file specified by 'rawDef' and uploads build context
// containing files referenced in '%files' section(s) to build server.
//
// Returns sha256 digest of uploaded build context if build context was uploaded successfully,
// otherwise returns errNoBuildContextFiles indicating no build context was uploaded/required.
func (app *App) uploadBuildContext(ctx context.Context, rawDef []byte) (string, error) {
	// Get list of files from def file '%files' section(s)
	files, err := app.getFiles(ctx, bytes.NewReader(rawDef))
	if err != nil {
		return "", fmt.Errorf("error getting build context files: %w", err)
	}
	if files == nil {
		return "", errNoBuildContextFiles
	}

	// Upload build context containing files referenced in def file to build server
	digest, err := app.buildClient.UploadBuildContext(ctx, files)
	if err != nil {
		return "", err
	}
	return digest, nil
}

func appendFileSuffix(name, suffix string, appendSuffix bool) string {
	if !appendSuffix {
		return name
	}
	return fmt.Sprintf("%v-%v", name, suffix)
}

// Run is the main application entrypoint
func (app *App) Run(ctx context.Context) error {
	if !app.force && app.dstFileName != "" {
		// Check for existence of dst files
		for _, arch := range app.archsToBuild {
			fn := appendFileSuffix(app.dstFileName, arch, len(app.archsToBuild) > 1)

			if _, err := os.Stat(fn); !os.IsNotExist(err) {
				return fmt.Errorf("destination file %q already exists", fn)
			}
		}
	}

	var err error
	buildDef, err := app.getBuildDef(app.buildSpec)
	if err != nil {
		return fmt.Errorf("unable to get build definition: %w", err)
	}

	// Upload build context, as necessary
	buildContext, err := app.uploadBuildContext(ctx, buildDef)
	if err != nil && !errors.Is(err, errNoBuildContextFiles) {
		return fmt.Errorf("error uploading build context: %w", err)
	}

	if len(app.archsToBuild) > 1 {
		fmt.Printf("Performing builds for following architectures: %v\n", strings.Join(app.archsToBuild, " "))
	}

	return app.build(ctx, buildDef, buildContext, app.archsToBuild)
}

func (app *App) initSigner(enable bool) (*signer, error) {
	if !enable {
		// Signing is disabled
		return nil, nil //nolint:nilnil
	}

	fmt.Printf("Build artifacts will be automatically signed\n")

	var so []signerOpt

	path, err := keyringPath(app.keyring)
	if err != nil {
		return nil, err
	}
	so = append(so, signKeyringFile(path))

	if keyringFingerprint := app.fingerprint; keyringFingerprint != "" {
		so = append(so, signKeyringFingerprint(keyringFingerprint))
	} else if app.keyIdx != -1 {
		so = append(so, signKeyringKeyIdx(app.keyIdx))
	} else {
		so = append(so, signEntitySelector(keyringEntitySelectorFunc))
	}

	if app.passphrase != "" {
		so = append(so, signKeyringPassphrase(app.passphrase))
	} else {
		so = append(so, signKeyringPassphraseFunc(keyringPassphraseFunc))
	}

	return newSigner(so...)
}

func (app *App) build(ctx context.Context, Def []byte, Context string, Archs []string) error {
	errs := make(map[string]error)

	s, err := app.initSigner(app.signing)
	if err != nil {
		return fmt.Errorf("error initializing signer: %w", err)
	}

	for _, arch := range Archs {
		dstFileName := appendFileSuffix(app.dstFileName, arch, len(Archs) > 1)

		var libraryRef string
		if app.LibraryRef != nil {
			libraryRef = app.LibraryRef.String()
		}

		fmt.Printf("Building for %v...\n", arch)

		bi, err := app.buildArch(ctx, arch, Def, Context, libraryRef, dstFileName, s)
		if err != nil {
			errs[arch] = err
			continue
		}

		signed := s != nil

		if !signed && dstFileName == "" {
			// Library ref specified; image pushed to library automatically
			if app.LibraryRef == nil {
				fmt.Printf("Build artifact %v is available for 24 hours or less\n", bi.LibraryRef())
			}
			continue
		}

		if signed && dstFileName == "" {
			// Do not display image stats
			continue
		}

		// Display file stats for locally downloaded image
		fi, err := os.Lstat(dstFileName)
		if err != nil {
			return fmt.Errorf("error opening file %v for reading: %w", dstFileName, err)
		}
		fmt.Fprintf(os.Stderr, "Wrote %v (%d bytes)\n", dstFileName, fi.Size())
	}

	return app.reportErrs(errs)
}

func (app *App) directLibraryUpload(filename string) bool {
	return app.LibraryRef != nil || filename == ""
}

func (app *App) buildArch(ctx context.Context, arch string, def []byte, buildContext string, libraryRef string, dstFileName string, s *signer) (*build.BuildInfo, error) {
	signed := s != nil

	var tmpFileName string
	var tmpLibraryRef string

	if !signed {
		if libraryRef != "" && dstFileName == "" {
			tmpLibraryRef = libraryRef
		} else if libraryRef == "" && dstFileName != "" {
			tmpFileName = dstFileName
		}
	}

	// Submit build request
	bi, err := app.buildArtifact(ctx, arch, def, buildContext, tmpLibraryRef)
	if err != nil {
		return nil, err
	}

	// Build completed successfully
	if !signed {
		if tmpFileName == "" {
			// Build image uploaded directly to library
			return bi, nil
		}

		// Build image will be written directly to 'tmpFileName'
	} else {
		if dstFileName != "" || libraryRef != "" {
			// Create (local) temporary file for images being pushed directly to library
			f, err := os.CreateTemp("", "scs-build-")
			if err != nil {
				return nil, err
			}
			f.Close()
			tmpFileName = f.Name()
		}
	}

	// Download file locally
	if err := app.retrieveArtifact(ctx, bi, tmpFileName, arch); err != nil {
		return nil, fmt.Errorf("error retrieving build artifact: %w", err)
	}

	if signed {
		// Sign local file
		if err := app.sign(ctx, s, tmpFileName); err != nil {
			return nil, err
		}

		if app.directLibraryUpload(dstFileName) {
			// Upload temporary (local) image file to library
			if err := app.uploadImage(ctx, tmpFileName, arch); err != nil {
				return nil, err
			}
		} else {
			// Rename temporary local file to specified destination
			if err := os.Rename(tmpFileName, dstFileName); err != nil {
				return nil, fmt.Errorf("file rename error: %w", err)
			}
		}
	}

	return bi, nil
}

func (app *App) sign(ctx context.Context, s *signer, fileName string) error {
	fmt.Printf("Signing...\n")

	if err := s.Sign(ctx, fileName); err != nil {
		return fmt.Errorf("signing error: %w", err)
	}
	return nil
}

func (app *App) uploadImage(ctx context.Context, tmpFileName, arch string) error {
	fp, err := os.Open(tmpFileName)
	if err != nil {
		return fmt.Errorf("uploading file: %w", err)
	}
	defer func() {
		_ = fp.Close()
	}()

	if _, err := app.libraryClient.UploadImage(ctx, fp, app.LibraryRef.Path, arch, app.LibraryRef.Tags, "", nil); err != nil {
		return fmt.Errorf("error uploading image %v to %v: %w", tmpFileName, app.LibraryRef.String(), err)
	}

	// Remove temporary file
	_ = os.Remove(tmpFileName)

	return nil
}

// reportErrs iterates over arch/error map and outputs error(s) to console
func (app *App) reportErrs(errs map[string]error) error {
	// Report any build errors

	if len(errs) == 0 {
		return nil
	}

	if len(errs) == 1 {
		// Return first (and only) error
		for _, err := range errs {
			return err
		}
	}

	fmt.Fprintf(os.Stderr, "\nBuild error(s):\n")

	for arch, err := range errs {
		fmt.Fprintf(os.Stderr, "  - %v: %v\n", arch, err)
	}

	fmt.Fprintln(os.Stderr)

	return errors.New("failed to build images")
}
