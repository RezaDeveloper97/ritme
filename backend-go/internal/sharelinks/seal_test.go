package sharelinks

import (
	"bytes"
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToken_FormatAndHash(t *testing.T) {
	tok, err := NewToken(rand.Reader)
	require.NoError(t, err)
	assert.Len(t, tok, 43)
	assert.True(t, ValidToken(tok))
	assert.False(t, ValidToken(tok[:42]))
	assert.False(t, ValidToken(strings.Repeat("a", 42)+"/"))
	assert.Len(t, HashToken(tok), 64)
	assert.NotContains(t, HashToken(tok), tok)
	other, err := NewToken(rand.Reader)
	require.NoError(t, err)
	assert.NotEqual(t, tok, other)
}

func TestSeal_RoundTripAndTamper(t *testing.T) {
	tok, err := NewToken(rand.Reader)
	require.NoError(t, err)
	plain := []byte(`{"record":{"person":{"name":"Maryam"}}}`)
	stored, err := Seal(tok, plain, rand.Reader)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(stored, "v1:"))
	assert.NotContains(t, stored, "Maryam")

	got, err := Open(tok, stored)
	require.NoError(t, err)
	assert.Equal(t, plain, got)

	other, _ := NewToken(rand.Reader)
	_, err = Open(other, stored)
	assert.ErrorIs(t, err, ErrUnreadable, "another token never opens it")

	b := []byte(stored)
	b[len(b)-3] ^= 1
	_, err = Open(tok, string(b))
	assert.ErrorIs(t, err, ErrUnreadable, "tampered ciphertext")
	_, err = Open(tok, "v2:"+stored[3:])
	assert.ErrorIs(t, err, ErrUnreadable, "unknown version")
	_, err = Open("not-a-token", stored)
	assert.ErrorIs(t, err, ErrUnreadable)
}

func TestSeal_FreshNonce(t *testing.T) {
	tok := strings.Repeat("A", 43)
	a, err := Seal(tok, []byte("x"), rand.Reader)
	require.NoError(t, err)
	b, err := Seal(tok, []byte("x"), rand.Reader)
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
	_, err = Seal(tok, []byte("x"), bytes.NewReader(nil))
	assert.Error(t, err, "a failing random source fails the seal")
}

func TestLink_Status(t *testing.T) {
	now := time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)
	l := Link{ExpiresAt: now.Add(time.Hour)}
	assert.Equal(t, StatusActive, l.Status(now))
	assert.Equal(t, StatusExpired, l.Status(now.Add(time.Hour)))
	l.RevokedAt = now
	assert.Equal(t, StatusRevoked, l.Status(now))
}

func TestLang_Lines(t *testing.T) {
	for _, loc := range []string{"en", "fa"} {
		for _, k := range []string{"created", "revoked", "not_found", "expired", "revoked_link", "limit"} {
			assert.NotEqual(t, "sharelinks.messages."+k, T("messages."+k, loc), loc+" "+k)
		}
	}
}
