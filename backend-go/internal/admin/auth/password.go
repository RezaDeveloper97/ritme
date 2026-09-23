// Package auth is the admin login: POST /auth/login, POST /auth/logout, GET /auth/me and
// PUT /auth/password (backend Admin\AuthController + AccountController), on the
// Redis sessions of httpadmin.
package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost is Laravel's BCRYPT_ROUNDS for admins (12).
const BcryptCost = 12

// dummyHash is compared against when the email is unknown, so a miss costs the same
// time as a wrong password (no account enumeration by timing).
var dummyHash = []byte("$2y$12$lo8Z3h/WEQp7b8AwCvwiTObVkv7/Xa68c/VnNTSszAOrR4kLj1HBi")

// HashPassword returns a bcrypt hash at BcryptCost. Go writes the $2a$ prefix; PHP's
// password_verify (Laravel Hash::check) accepts it, so Laravel can still read it.
func HashPassword(plain string) (string, error) {
	if len(plain) > 72 {
		return "", errors.New("auth: password longer than 72 bytes")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}
	return string(b), nil
}

// CheckPassword verifies plain against a Laravel ($2y$) or Go ($2a$/$2b$) bcrypt hash.
// Like PHP's crypt(), only the first 72 bytes of the password count.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), bcryptInput(plain)) == nil
}

// burnTime runs one bcrypt comparison whose result is ignored.
func burnTime(plain string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash, bcryptInput(plain))
}

func bcryptInput(plain string) []byte {
	b := []byte(plain)
	if len(b) > 72 {
		b = b[:72]
	}
	return b
}
