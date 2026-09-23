package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// Made by Laravel 12 (Illuminate\Hashing\BcryptHasher, rounds 12) for "Secret-pass-123".
const laravelHash = "$2y$12$lo8Z3h/WEQp7b8AwCvwiTObVkv7/Xa68c/VnNTSszAOrR4kLj1HBi"

func TestCheckPassword_LaravelHash(t *testing.T) {
	assert.True(t, CheckPassword(laravelHash, "Secret-pass-123"))
	assert.False(t, CheckPassword(laravelHash, "secret-pass-123"))
	assert.False(t, CheckPassword(laravelHash, ""))
	assert.False(t, CheckPassword("not-a-hash", "Secret-pass-123"))
}

func TestHashPassword_Cost12AndRoundTrip(t *testing.T) {
	h, err := HashPassword("another-Pass-9")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(h, "$2a$12$"), h)
	cost, err := bcrypt.Cost([]byte(h))
	require.NoError(t, err)
	assert.Equal(t, BcryptCost, cost)
	assert.True(t, CheckPassword(h, "another-Pass-9"))

	_, err = HashPassword(strings.Repeat("x", 73))
	assert.Error(t, err)
}

func TestCheckPassword_72ByteLimitLikePHP(t *testing.T) {
	long := strings.Repeat("a", 72)
	h, err := HashPassword(long)
	require.NoError(t, err)
	assert.True(t, CheckPassword(h, long+"ignored-by-bcrypt"))
}
