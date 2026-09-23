// Package passport validates and issues Laravel Passport personal access tokens
// (Passport 13 / league/oauth2-server 9 / lcobucci/jwt 5) byte-compatibly, with the same
// RSA key pair and the same oauth_access_tokens / oauth_clients rows, so a token issued by
// either stack is accepted by the other (docs/go-migration/infra-auth-admin-inventory.md §1).
//
// The key pair lives on the backend-storage volume (STORAGE_PATH). Go never generates,
// copies or rewrites keys: a missing or unreadable key is a start-up error.
package passport

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Key file names inside STORAGE_PATH (Passport::keyPath()).
const (
	PrivateKeyFile = "oauth-private.key"
	PublicKeyFile  = "oauth-public.key"
)

// Keys is the Passport RSA key pair.
type Keys struct {
	Private *rsa.PrivateKey
	Public  *rsa.PublicKey
}

// LoadKeys reads storagePath/oauth-private.key (PKCS#8; PKCS#1 is accepted too) and
// storagePath/oauth-public.key (PKIX). Both must exist, parse, and belong together.
func LoadKeys(storagePath string) (*Keys, error) {
	if storagePath == "" {
		return nil, errors.New("passport: STORAGE_PATH is empty")
	}
	priv, err := loadPrivate(filepath.Join(storagePath, PrivateKeyFile))
	if err != nil {
		return nil, err
	}
	pub, err := loadPublic(filepath.Join(storagePath, PublicKeyFile))
	if err != nil {
		return nil, err
	}
	if !priv.PublicKey.Equal(pub) {
		return nil, errors.New("passport: oauth-public.key does not match oauth-private.key")
	}
	return &Keys{Private: priv, Public: pub}, nil
}

func readPEM(path string) (*pem.Block, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: fixed file name under STORAGE_PATH
	if err != nil {
		return nil, fmt.Errorf("passport: %w", err)
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("passport: %s: no PEM block", path)
	}
	return block, nil
}

func loadPrivate(path string) (*rsa.PrivateKey, error) {
	block, err := readPEM(path)
	if err != nil {
		return nil, err
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rk, ok := k.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("passport: %s: not an RSA key", path)
		}
		return rk, nil
	}
	k, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("passport: %s: %w", path, err)
	}
	return k, nil
}

func loadPublic(path string) (*rsa.PublicKey, error) {
	block, err := readPEM(path)
	if err != nil {
		return nil, err
	}
	k, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("passport: %s: %w", path, err)
	}
	rk, ok := k.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("passport: %s: not an RSA key", path)
	}
	return rk, nil
}
