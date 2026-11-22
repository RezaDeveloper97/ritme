package passport

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ContractKeyFingerprint is the Fingerprint of the committed test key pair in backend-go/contract/fixtures/keys
// (T-M12-07, S-13). That key is public — anyone can mint tokens with it — so production refuses to start with it.
// Only the fingerprint is embedded here, never the key.
const ContractKeyFingerprint = "1a141d8e85c85ec28a98708906a14672422a8915244f9e342525a1f6d5ff7a08"

// ErrContractKey is returned when production would run on the committed contract key.
var ErrContractKey = errors.New("passport: oauth-public.key is the committed contract test key; " +
	"refusing to start with APP_ENV=production (put the real Passport keys in STORAGE_PATH)")

// Fingerprint is the lower-case hex SHA-256 of the key's PKIX (SubjectPublicKeyInfo) DER encoding — the same value as
// `openssl pkey -pubin -in oauth-public.key -outform DER | sha256sum`.
func Fingerprint(pub *rsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("passport: fingerprint: %w", err)
	}
	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:]), nil
}

// CheckProductionKey refuses the committed contract key when appEnv is "production". Every other environment
// (local, testing, contract, staging …) may use it.
func CheckProductionKey(pub *rsa.PublicKey, appEnv string) error {
	if appEnv != "production" {
		return nil
	}
	fp, err := Fingerprint(pub)
	if err != nil {
		return err
	}
	if fp == ContractKeyFingerprint {
		return ErrContractKey
	}
	return nil
}

// CheckStorageKey is the start-up form of CheckProductionKey (config.LoadFrom calls it): it reads
// storagePath/oauth-public.key and refuses the contract key in production. A missing or unreadable key is not its
// business — LoadKeys reports that when the guard is built — so only a readable contract key fails here.
func CheckStorageKey(storagePath, appEnv string) error {
	if appEnv != "production" || storagePath == "" {
		return nil
	}
	pub, err := loadPublic(filepath.Join(storagePath, PublicKeyFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return CheckProductionKey(pub, appEnv)
}
