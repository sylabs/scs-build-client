// Copyright (c) 2023, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package buildclient

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/spf13/viper"
	"github.com/sylabs/sif/v2/pkg/integrity"
	"golang.org/x/term"
)

type pgpSignerOpts struct {
	keyringFile        string
	passphraseFunc     func() ([]byte, error)
	entitySelectorFunc func(e openpgp.EntityList) (*openpgp.Entity, error)
}

var (
	errKeyringPath       = errors.New("unable to determine keyring path: neither XDG_CONFIG_HOME nor HOME set")
	errKeyNotFound       = errors.New("key not found")
	errNoPrivateKeyFound = errors.New("private key not found")
	errIndexOutOfRange   = errors.New("index out of range")
)

func parsePGPSignerOpts(v *viper.Viper) ([]pgpSignerOpt, error) {
	var so []pgpSignerOpt

	path, err := keyringPath(v.GetString(keyKeyring))
	if err != nil {
		return nil, err
	}
	so = append(so, signKeyringFile(path))

	if keyringFingerprint := v.GetString(keyFingerprint); keyringFingerprint != "" {
		so = append(so, signKeyringFingerprint(keyringFingerprint))
	} else if keyidx := v.GetInt(keySigningKeyIndex); keyidx != -1 {
		so = append(so, signKeyringKeyIdx(keyidx))
	} else {
		so = append(so, signEntitySelector(keyringEntitySelectorFunc))
	}

	if passphrase := v.GetString(keyPassphrase); passphrase != "" {
		so = append(so, signKeyringPassphrase(passphrase))
	} else {
		so = append(so, signKeyringPassphraseFunc(keyringPassphraseFunc))
	}

	return so, nil
}

func keyringPath(keyring string) (string, error) {
	if path := keyring; path != "" {
		return path, nil
	}

	if home := os.Getenv("XDG_CONFIG_HOME"); home != "" {
		return filepath.Join(home, ".gnupg", "secring.gpg"), nil
	}

	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".gnupg", "secring.gpg"), nil
	}

	return "", errKeyringPath
}

func keyringPassphraseFunc() ([]byte, error) {
	fmt.Print("Keyring passphrase: ")
	bytePassword, err := term.ReadPassword(0)

	// Add missing newline after passphrase prompt
	fmt.Println()

	if err != nil {
		return []byte(""), err
	}

	return bytePassword, nil
}

func keyringEntitySelectorFunc(e openpgp.EntityList) (*openpgp.Entity, error) {
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		var index int
		for i, entity := range e {
			for _, t := range entity.Identities {
				fmt.Printf("%d) U: %s (%s) <%s>\n", i, t.UserId.Name, t.UserId.Comment, t.UserId.Email)
			}
			fmt.Printf("   C: %s - %d\n", entity.PrimaryKey.CreationTime, i)
			fmt.Printf("   F: %0X\n", entity.PrimaryKey.Fingerprint)
			bits, _ := entity.PrimaryKey.BitLength()
			fmt.Printf("   L: %d\n", bits)
			fmt.Printf("   --------\n")
		}
		fmt.Printf("Key #: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		index, err := strconv.Atoi(strings.TrimSuffix(input, "\n"))
		if err != nil {
			return nil, err
		}
		if index < 0 || index >= len(e) {
			return nil, errIndexOutOfRange
		}
		return e[index], nil
	}
	return nil, nil //nolint:nilnil
}

// signKeyringFile Set the keyring to use for signing.
func signKeyringFile(keyringFile string) pgpSignerOpt {
	return func(s *pgpSignerOpts) error {
		s.keyringFile = keyringFile
		return nil
	}
}

// signKeyringFingerprint Sign using this Fingerprint.
func signKeyringFingerprint(keyringFingerprint string) pgpSignerOpt {
	return signEntitySelector(func(el openpgp.EntityList) (*openpgp.Entity, error) {
		for _, e := range el {
			fPrint := fmt.Sprintf("%0x", e.PrimaryKey.Fingerprint)
			if fPrint == strings.ToLower(keyringFingerprint) {
				return e, nil
			}
		}
		return nil, errKeyNotFound
	})
}

// signKeyringKeyIdx using key at index n.
func signKeyringKeyIdx(n int) pgpSignerOpt {
	return signEntitySelector(func(el openpgp.EntityList) (*openpgp.Entity, error) {
		if n >= len(el) {
			return nil, errKeyNotFound
		}
		return el[n], nil
	})
}

// signEntitySelector specifies fn as the entity selection function.
func signEntitySelector(fn func(e openpgp.EntityList) (*openpgp.Entity, error)) pgpSignerOpt {
	return func(s *pgpSignerOpts) error {
		s.entitySelectorFunc = fn
		return nil
	}
}

// signKeyringPassphraseFunc Passphrease prompt function.
func signKeyringPassphraseFunc(fn func() ([]byte, error)) pgpSignerOpt {
	return func(s *pgpSignerOpts) error {
		s.passphraseFunc = fn
		return nil
	}
}

// signKeyringPassphrase Passphrase for encrypted key.
func signKeyringPassphrase(s string) pgpSignerOpt {
	return signKeyringPassphraseFunc(func() ([]byte, error) {
		return []byte(s), nil
	})
}

type pgpSignerOpt func(*pgpSignerOpts) error

// stripPublicKeys returns an EntityList of PrivateKeys only.
func stripPublicKeys(e openpgp.EntityList) openpgp.EntityList {
	var el openpgp.EntityList
	for _, entity := range e {
		if entity.PrivateKey != nil {
			el = append(el, entity)
		}
	}
	return el
}

// getPGPSignerOpts returns a Signer that will Sign imgName.
func getPGPSignerOpts(opts ...pgpSignerOpt) ([]integrity.SignerOpt, error) {
	s := pgpSignerOpts{}

	// Apply options.
	for _, o := range opts {
		if err := o(&s); err != nil {
			return nil, err
		}
	}

	keyringFileBuffer, err := os.Open(s.keyringFile)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Using keyfile: %v\n", s.keyringFile)
	defer keyringFileBuffer.Close()

	e, err := openpgp.ReadKeyRing(keyringFileBuffer)
	if err != nil {
		return nil, fmt.Errorf("key read: %w", err)
	}
	e = stripPublicKeys(e)
	if len(e) == 0 {
		return nil, errNoPrivateKeyFound
	}

	entity, err := s.entitySelectorFunc(e)
	if err != nil {
		return nil, err
	}
	for _, i := range entity.Identities {
		fmt.Printf("Using Key: %s (%s) <%s>\n", i.UserId.Name, i.UserId.Comment, i.UserId.Email)
	}

	if entity.PrivateKey.Encrypted {
		b, err := s.passphraseFunc()
		if err != nil {
			return nil, err
		}
		if err = entity.PrivateKey.Decrypt(b); err != nil {
			return nil, fmt.Errorf("key decrypt: %w", err)
		}
	}

	return []integrity.SignerOpt{integrity.OptSignWithEntity(entity)}, nil
}
