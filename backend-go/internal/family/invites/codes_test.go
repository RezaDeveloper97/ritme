package invites

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/config"
)

func TestNewCode_AlphabetAndLength(t *testing.T) {
	code, err := NewCode(bytes.NewReader([]byte{0, 31, 32, 255, 8, 40}))
	require.NoError(t, err)
	assert.Equal(t, "A9A9JJ", code)
	_, err = NewCode(bytes.NewReader([]byte{1}))
	require.Error(t, err, "short entropy fails")
}

func TestNormalizeCode(t *testing.T) {
	for in, want := range map[string]string{"rt7k2m": "RT7K2M", "RT7-K2M": "RT7K2M", "rt7 k2m\t": "RT7K2M", "RT۷K۲M": "RT7K2M", "RT٧K٢M": "RT7K2M"} {
		got, ok := NormalizeCode(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, in := range []string{"", "RT7K2", "RT7K2MM", "RT0K2M", "RTIK2M", "RT7K2!"} {
		_, ok := NormalizeCode(in)
		assert.False(t, ok, in)
	}
}

func TestHashCode_PepperedAndStable(t *testing.T) {
	a := HashCode([]byte("p1"), "RT7K2M")
	assert.Len(t, a, 64)
	assert.Equal(t, a, HashCode([]byte("p1"), "RT7K2M"))
	assert.NotEqual(t, a, HashCode([]byte("p2"), "RT7K2M"), "the pepper keys the hash")
	assert.NotEqual(t, a, HashCode([]byte("p1"), "RT7K2N"))
}

func TestNormalizeAndMaskMobile(t *testing.T) {
	for in, want := range map[string]string{"09123456789": "09123456789", "+989123456789": "09123456789", "00989123456789": "09123456789",
		"9123456789": "09123456789", "۰۹۱۲۳۴۵۶۷۸۹": "09123456789", "0912 345 6789": "09123456789"} {
		got, ok := NormalizeMobile(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, in := range []string{"", "0812345678", "0912345678a", "021123456789"} {
		_, ok := NormalizeMobile(in)
		assert.False(t, ok, in)
	}
	assert.Equal(t, "0912****789", MaskMobile("09123456789"))
	assert.Equal(t, "***", MaskMobile("0912"))
}

func TestIsRefusal(t *testing.T) {
	for _, err := range []error{ErrInvalid, ErrExpired, ErrUsed, ErrLocked, ErrPhoneMismatch, ErrSelf, ErrAlreadyLinked, ErrNotFound} {
		assert.True(t, IsRefusal(err), err.Error())
	}
	assert.False(t, IsRefusal(ErrNotAllowed), "the mode refusal has its own message")
	assert.False(t, IsRefusal(errors.New("db down")))
}

// TestPepper_RequiredOutsideLocal: only local / testing / contract may hash codes with the public DevPepper; anywhere
// else the start-up check refuses a missing or short pepper (FAM-L1 / CMP-L3).
func TestPepper_RequiredOutsideLocal(t *testing.T) {
	strong := strings.Repeat("k", config.MinCompanionPepperLen)
	for _, env := range []string{"local", "testing", "contract"} {
		c := config.Companion{}
		require.NoError(t, c.CheckPepper(config.App{Env: env}), env)
		assert.False(t, c.PepperMissing(config.App{Env: env}), env)
	}
	for _, env := range []string{"production", "staging", "stage", "beta", ""} {
		require.Error(t, config.Companion{}.CheckPepper(config.App{Env: env}), env)
		assert.True(t, config.Companion{}.PepperMissing(config.App{Env: env}), env)
		require.Error(t, config.Companion{CodePepper: "short"}.CheckPepper(config.App{Env: env}), env)
		require.NoError(t, config.Companion{CodePepper: strong}.CheckPepper(config.App{Env: env}), env)
	}
	assert.Equal(t, DevPepper, NewService(nil, nil, Options{}).opt.Pepper, "dev default only when unset")
	assert.Equal(t, []byte(strong), NewService(nil, nil, Options{Pepper: []byte(strong)}).opt.Pepper)
}

func flatKeys(prefix string, v any, out map[string]bool) {
	m, ok := v.(map[string]any)
	if !ok {
		out[prefix] = true
		return
	}
	for k, sub := range m {
		flatKeys(prefix+"."+k, sub, out)
	}
}

// TestLang_KeyParity: every language file has the same keys, and every key the code asks for exists.
func TestLang_KeyParity(t *testing.T) {
	dirs, err := fs.ReadDir(langFS, "lang")
	require.NoError(t, err)
	var ref map[string]bool
	for _, d := range dirs {
		raw, err := fs.ReadFile(langFS, "lang/"+d.Name()+"/invites.json")
		require.NoError(t, err)
		var v any
		require.NoError(t, json.Unmarshal(raw, &v))
		keys := map[string]bool{}
		flatKeys("invites", v, keys)
		if ref == nil {
			ref = keys
			continue
		}
		assert.Equal(t, slices.Sorted(maps.Keys(ref)), slices.Sorted(maps.Keys(keys)), d.Name())
	}
	for _, n := range []string{NoticeAccepted, NoticeAwaitingOwner, NoticeInviteExpired, NoticeCarerExpired} {
		assert.True(t, ref["invites.notices."+n+"_title"], n)
		assert.True(t, ref["invites.notices."+n+"_body"], n)
	}
	assert.Equal(t, "Mina's access has ended", T("notices.carer_expired_title", "en", "name", "Mina"))
	assert.NotEqual(t, T("messages.invite_invalid", "en"), T("messages.invite_invalid", "fa"))
}
