package sharelinks

import (
	"bytes"
	"context"
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

func TestCodeLangFilesLoad(t *testing.T) {
	assert.Equal(t, "This code or link is not valid", T("messages.code_not_found", "en"))
	assert.Equal(t, "این کد یا لینک معتبر نیست", T("messages.code_not_found", "fa"))
	assert.Equal(t, "You can have at most 5 active doctor codes. Revoke one first.",
		Tp("messages.code_limit", map[string]string{"max": "5"}, "en"))
	assert.NotEmpty(t, attributes("fa"))
}

func TestNewCode_AlphabetLengthAndSpread(t *testing.T) {
	seen := map[rune]int{}
	codes := map[string]bool{}
	for range 2000 {
		c, err := NewCode(rand.Reader)
		require.NoError(t, err)
		require.Len(t, c, CodeLength)
		for _, r := range c {
			require.True(t, strings.ContainsRune(CodeAlphabet, r), c)
			seen[r]++
		}
		codes[c] = true
	}
	assert.Len(t, seen, len(CodeAlphabet), "every symbol shows up")
	assert.Len(t, codes, 2000, "no repeats in a small sample")
	for _, bad := range "01ILOU" {
		assert.NotContains(t, CodeAlphabet, string(bad))
	}
	_, err := NewCode(bytes.NewReader(nil))
	assert.Error(t, err, "a failing entropy source is an error, not a weak code")
}

func TestFormatAndNormalizeCode(t *testing.T) {
	assert.Equal(t, "K7QM-3XRA", FormatCode("K7QM3XRA"))
	for in, want := range map[string]string{
		"K7QM-3XRA":   "K7QM3XRA",
		"k7qm 3xra":   "K7QM3XRA",
		"K۷QM-۳XRA":   "K7QM3XRA",
		"K٧QM٣XRA":    "K7QM3XRA",
		" K7QM-3XRA ": "K7QM3XRA",
	} {
		got, ok := NormalizeCode(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, bad := range []string{"", "K7QM3XR", "K7QM3XRAA", "O0OO-0000", "K7QM-3XR!", "ک۷QM۳XRA", strings.Repeat("A", 100)} {
		_, ok := NormalizeCode(bad)
		assert.False(t, ok, bad)
	}
}

func TestCoder_HashAndWrap(t *testing.T) {
	a, b := NewCoder([]byte("pepper-a-0123456789012345678901234")), NewCoder([]byte("pepper-b-0123456789012345678901234"))
	assert.Equal(t, a.Hash("K7QM3XRA"), a.Hash("K7QM3XRA"))
	assert.NotEqual(t, a.Hash("K7QM3XRA"), b.Hash("K7QM3XRA"), "the hash depends on the pepper")
	assert.NotEqual(t, a.Hash("K7QM3XRA"), a.Hash("K7QM3XRB"))
	assert.Len(t, a.Hash("K7QM3XRA"), 64)
	assert.Equal(t, NewCoder(nil).Hash("X"), NewCoder(devCodePepper).Hash("X"), "empty = the development pepper")

	token, err := NewToken(rand.Reader)
	require.NoError(t, err)
	wrapped, err := a.Wrap("K7QM3XRA", token, rand.Reader)
	require.NoError(t, err)
	assert.NotContains(t, wrapped, token)
	got, err := a.Unwrap("K7QM3XRA", wrapped)
	require.NoError(t, err)
	assert.Equal(t, token, got)

	for name, try := range map[string]func() (string, error){
		"wrong code":   func() (string, error) { return a.Unwrap("K7QM3XRB", wrapped) },
		"wrong pepper": func() (string, error) { return b.Unwrap("K7QM3XRA", wrapped) },
		"tampered":     func() (string, error) { return a.Unwrap("K7QM3XRA", wrapped[:len(wrapped)-4]+"AAAA") },
		"bad version":  func() (string, error) { return a.Unwrap("K7QM3XRA", "v2:"+wrapped[3:]) },
	} {
		_, err := try()
		assert.ErrorIs(t, err, ErrUnreadable, name)
	}
}

func TestViewerOf(t *testing.T) {
	for ua, want := range map[string][2]string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1": {DeviceMobile, BrowserSafari},
		"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Mobile Safari/537.36":                       {DeviceMobile, BrowserChrome},
		"Mozilla/5.0 (Linux; Android 13; SM-X700) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/24.0 Chrome/117.0 Safari/537.36":          {DeviceTablet, BrowserSamsung},
		"Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/126.0 Mobile/15E148 Safari/604.1":           {DeviceTablet, BrowserChrome},
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36 Edg/126.0":                   {DeviceDesktop, BrowserEdge},
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 14.5; rv:127.0) Gecko/20100101 Firefox/127.0":                                                     {DeviceDesktop, BrowserFirefox},
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36 OPR/111.0":                             {DeviceDesktop, BrowserOpera},
		"curl/8.6.0": {DeviceUnknown, BrowserOther},
		"":           {DeviceUnknown, BrowserOther},
	} {
		v := ViewerOf(ViaLink, ua)
		assert.Equal(t, ViaLink, v.Via)
		assert.Equal(t, want, [2]string{v.Device, v.Browser}, ua)
	}
}

type fakeFiles struct{ owner uint64 }

func (f fakeFiles) Get(_ context.Context, owner, id uint64) (files.File, error) {
	if owner != f.owner || id == 99 {
		return files.File{}, files.ErrNotFound
	}
	return files.File{ID: id, MIME: "image/png", Size: 120}, nil
}

func (fakeFiles) Link(f files.File, now time.Time, ttl time.Duration) (files.Link, error) {
	exp := now.Add(ttl)
	return files.Link{URL: "https://api.test/api/v1/files/1/download?sig=x", ExpiresAt: &exp}, nil
}

func TestPublicDocuments_SignsOwnFilesAndDropsIDs(t *testing.T) {
	snap := snapshotDocuments([]SharedDocument{{Kind: "imaging", Title: "Scan", Date: civildate.MustParse("2026-09-20"),
		FileIDs: []uint64{5, 99}}})
	raw, err := jsonx.Marshal(jsonx.Obj("documents", snap), 0)
	require.NoError(t, err)
	decoded, err := phpval.Decode(raw)
	require.NoError(t, err)
	docs, _ := decoded.(phpval.Map).Get("documents")

	now := time.Date(2026, 10, 6, 9, 0, 0, 0, civildate.Tehran)
	s := (&Service{}).WithDocuments(nil, fakeFiles{owner: 7})
	out, err := s.publicDocuments(context.Background(), 7, docs, now)
	require.NoError(t, err)
	require.Len(t, out, 1)
	body, err := jsonx.Marshal(out[0], jsonx.UnescapedSlashes)
	require.NoError(t, err)
	assert.JSONEq(t, `{"kind":"imaging","title":"Scan","date":"2026-09-20","ended_on":null,"centre":null,"doctor":null,
		"files":[{"mime":"image/png","size_bytes":120,"url":"https://api.test/api/v1/files/1/download?sig=x",
		"url_expires_at":"2026-10-06T09:05:00+03:30"}]}`, string(body), "file 99 is gone; no file ids")

	out, err = s.publicDocuments(context.Background(), 8, docs, now)
	require.NoError(t, err)
	f, _ := out[0].Get("files")
	assert.Empty(t, f, "files of another owner never sign")
	out, err = (&Service{}).publicDocuments(context.Background(), 7, docs, now)
	require.NoError(t, err)
	f, _ = out[0].Get("files")
	assert.Empty(t, f, "no file service: listed without files")
}
