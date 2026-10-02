package companion

import (
	"bytes"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCode_AlphabetAndLength(t *testing.T) {
	seen := map[string]bool{}
	for i := range 200 {
		src := rand.Reader
		if i == 0 {
			src = bytes.NewReader([]byte{0, 31, 32, 63, 255, 128}) // low 5 bits: 0, 31, 0, 31, 31, 0
		}
		code, err := newCode(src)
		require.NoError(t, err)
		require.Len(t, code, CodeLength)
		for _, r := range code {
			assert.True(t, strings.ContainsRune(CodeAlphabet, r), code)
		}
		norm, ok := NormalizeCode(code)
		assert.True(t, ok)
		assert.Equal(t, code, norm)
		seen[code] = true
	}
	assert.Greater(t, len(seen), 190)
	assert.True(t, seen["A9A99A"], "byte values map by their low 5 bits")

	_, err := newCode(bytes.NewReader([]byte{1, 2}))
	require.Error(t, err, "short entropy read fails")
}

func TestNormalizeCode(t *testing.T) {
	cases := map[string]string{
		"RT7K2M":     "RT7K2M",
		"rt7k2m":     "RT7K2M",
		" RT7-K2M ":  "RT7K2M",
		"RT۷K۲M":     "RT7K2M", // Persian digits
		"RT٧K٢M":     "RT7K2M", // Arabic-Indic digits
		"r t 7 k2 m": "RT7K2M",
	}
	for in, want := range cases {
		got, ok := NormalizeCode(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, bad := range []string{"", "RT7K2", "RT7K2MX", "RT0K2M", "RT1K2M", "RTOK2M", "RTIK2M", "RT7K2!", "رتکلمن"} {
		_, ok := NormalizeCode(bad)
		assert.False(t, ok, bad)
	}
}

func TestNormalizeMobile(t *testing.T) {
	for in, want := range map[string]string{
		"09121234567":     "09121234567",
		"۰۹۱۲۱۲۳۴۵۶۷":     "09121234567",
		"+989121234567":   "09121234567",
		"00989121234567":  "09121234567",
		"989121234567":    "09121234567",
		"9121234567":      "09121234567",
		"0912 123 4567":   "09121234567",
		"(0912) 123-4567": "09121234567",
	} {
		got, ok := NormalizeMobile(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, bad := range []string{"", "0912123456", "091212345678", "08121234567", "0912abc4567", "+1 555 123 4567"} {
		_, ok := NormalizeMobile(bad)
		assert.False(t, ok, bad)
	}
}

func TestHashCode_PepperedAndStable(t *testing.T) {
	a := NewService(nil, nil, Options{CodePepper: []byte("one")})
	b := NewService(nil, nil, Options{CodePepper: []byte("two")})
	assert.Len(t, a.hashCode("RT7K2M"), 64)
	assert.Equal(t, a.hashCode("RT7K2M"), a.hashCode("RT7K2M"))
	assert.NotEqual(t, a.hashCode("RT7K2M"), a.hashCode("RT7K2N"))
	assert.NotEqual(t, a.hashCode("RT7K2M"), b.hashCode("RT7K2M"), "the pepper keys the hash")
	assert.NotContains(t, a.hashCode("RT7K2M"), "RT7K2M")
}

func TestLevelsAndGrants(t *testing.T) {
	assert.False(t, LevelNone.CanRead())
	assert.False(t, LevelNone.CanWrite())
	assert.True(t, LevelView.CanRead())
	assert.False(t, LevelView.CanWrite())
	assert.True(t, LevelEdit.CanRead())
	assert.True(t, LevelEdit.CanWrite())
	assert.False(t, Level("admin").Valid())

	g := Grants{SectionMeds: LevelEdit}
	assert.Equal(t, LevelEdit, g.Of(SectionMeds))
	assert.Equal(t, LevelNone, g.Of(SectionCycle), "absent = none")
	assert.Equal(t, LevelNone, Grants(nil).Of(SectionCycle))
	require.NoError(t, g.Validate())
	require.ErrorIs(t, Grants{"diary": LevelView}.Validate(), ErrInvalidSection)
	require.ErrorIs(t, Grants{SectionCycle: "admin"}.Validate(), ErrInvalidLevel)

	assert.True(t, TypeSpouse.Valid())
	assert.False(t, Type("parent").Valid(), "parent arrives with CB-TEEN-01")
}

func TestValidateInput(t *testing.T) {
	in := InviteInput{Type: TypePartner, DisplayName: "  Ali ", Phone: "+989121234567"}
	require.NoError(t, validateInput(&in))
	assert.Equal(t, "Ali", in.DisplayName)
	assert.Equal(t, "09121234567", in.Phone)

	require.ErrorIs(t, validateInput(&InviteInput{Type: "friend"}), ErrInvalidType)
	require.ErrorIs(t, validateInput(&InviteInput{Type: TypePartner, Phone: "123"}), ErrInvalidPhone)
	require.ErrorIs(t, validateInput(&InviteInput{Type: TypePartner, DisplayName: strings.Repeat("ن", 101)}), ErrInvalidName)
	require.ErrorIs(t, validateInput(&InviteInput{Type: TypePartner, ChildIDs: []uint64{1}}), ErrChildrenNoSpouse)
	require.ErrorIs(t, validateInput(&InviteInput{Type: TypeSpouse, Grants: Grants{SectionCycle: "all"}}), ErrInvalidLevel)
}
