package media

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ftyp(brand string) []byte {
	return append([]byte{0, 0, 0, 0x20, 'f', 't', 'y', 'p'}, []byte(brand+"\x00\x00\x02\x00isomiso2avc1mp41")...)
}

func TestSniff(t *testing.T) {
	webmHead := append([]byte{0x1A, 0x45, 0xDF, 0xA3, 0x9F, 0x42, 0x86, 0x81, 0x01, 0x42, 0x82, 0x84}, []byte("webm")...)
	mkvHead := append([]byte{0x1A, 0x45, 0xDF, 0xA3, 0x9F, 0x42, 0x86, 0x81, 0x01, 0x42, 0x82, 0x88}, []byte("matroska")...)
	cases := []struct {
		kind string
		head []byte
		want string
	}{
		{KindVideo, ftyp("isom"), "video/mp4"},
		{KindVideo, ftyp("mp42"), "video/mp4"},
		{KindVideo, ftyp("qt  "), "video/quicktime"},
		{KindVideo, webmHead, "video/webm"},
		{KindVideo, mkvHead, ""},
		{KindVideo, ftyp("M4A "), ""},
		{KindVideo, []byte("%PDF-1.7\n"), ""},
		{KindVideo, []byte("MZ\x90\x00 an exe"), ""},
		{KindAudio, ftyp("M4A "), "audio/mp4"},
		{KindAudio, ftyp("qt  "), ""},
		{KindAudio, []byte("ID3\x04\x00\x00"), "audio/mpeg"},
		{KindAudio, []byte{0xFF, 0xFB, 0x90, 0x64}, "audio/mpeg"},
		{KindAudio, []byte{0xFF, 0xF1, 0x50, 0x80}, "audio/aac"},
		{KindAudio, []byte("OggS\x00\x02"), "audio/ogg"},
		{KindAudio, []byte("RIFF\x24\x08\x00\x00WAVEfmt "), "audio/wav"},
		{KindAudio, []byte("fLaC\x00\x00"), "audio/flac"},
		{KindAudio, webmHead, "audio/webm"},
		{KindAudio, []byte("<html>"), ""},
		{KindPDF, []byte("%PDF-1.4\n%âãÏÓ"), "application/pdf"},
		{KindPDF, []byte(" %PDF-1.4"), ""},
		{KindPDF, ftyp("isom"), ""},
		{"image", []byte("%PDF-1.4"), ""},
		{KindVideo, nil, ""},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, Sniff(c.kind, c.head), "%s %q", c.kind, c.head)
		if c.want != "" {
			assert.True(t, Accepts(c.kind, c.want), c.want)
		}
	}
}

func TestSigner(t *testing.T) {
	now := time.Unix(1_790_000_000, 0)
	s := NewSigner(DevKey())
	exp := now.Add(DefaultURLTTL)
	sig := s.Sign(7, 42, exp)
	es := "1790001800"

	viewer, err := s.Verify(7, "42", es, sig, now)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), viewer)

	for name, tc := range map[string]struct {
		id              uint64
		viewer, exp, sg string
	}{
		"other media":   {8, "42", es, sig},
		"other viewer":  {7, "43", es, sig},
		"other expiry":  {7, "42", "1790001801", sig},
		"padded expiry": {7, "42", "01790001800", sig},
		"padded viewer": {7, "042", es, sig},
		"zero viewer":   {7, "0", es, sig},
		"bad base64":    {7, "42", es, "***"},
		"empty":         {7, "", "", ""},
	} {
		_, err := s.Verify(tc.id, tc.viewer, tc.exp, tc.sg, now)
		assert.ErrorIs(t, err, ErrLinkInvalid, name)
	}

	// Past expiry: 410 material, only after the signature checks out.
	_, err = s.Verify(7, "42", es, sig, exp.Add(time.Second))
	assert.ErrorIs(t, err, ErrLinkExpired)

	// A validly signed link further out than MaxURLTTL is refused.
	far := now.Add(MaxURLTTL + time.Minute)
	_, err = s.Verify(7, "42", "1790003660", s.Sign(7, 42, far), now)
	assert.ErrorIs(t, err, ErrLinkInvalid)

	// Another key does not verify.
	other := NewSigner([]byte(strings.Repeat("k", 32)))
	_, err = other.Verify(7, "42", es, sig, now)
	assert.ErrorIs(t, err, ErrLinkInvalid)

	var disabled *Signer
	_, err = disabled.Verify(7, "42", es, sig, now)
	assert.ErrorIs(t, err, ErrDisabled)
	assert.Nil(t, NewSigner(nil))
}

func TestParseRange(t *testing.T) {
	const size = 1000
	cases := []struct {
		header  string
		want    byteRange
		satisfy bool
	}{
		{"", byteRange{0, 999, false}, true},
		{"bytes=0-99", byteRange{0, 99, true}, true},
		{"bytes=500-", byteRange{500, 999, true}, true},
		{"bytes=900-5000", byteRange{900, 999, true}, true},
		{"bytes=-100", byteRange{900, 999, true}, true},
		{"bytes=-5000", byteRange{0, 999, true}, true},
		{"bytes=999-999", byteRange{999, 999, true}, true},
		{"bytes=1000-", byteRange{}, false},
		{"bytes=-0", byteRange{}, false},
		{"bytes=5-1", byteRange{0, 999, false}, true},     // invalid spec: ignored
		{"bytes=0-1,5-6", byteRange{0, 999, false}, true}, // multi-range: ignored
		{"items=0-1", byteRange{0, 999, false}, true},     // other unit: ignored
		{"bytes=abc-", byteRange{0, 999, false}, true},    // malformed: ignored
		{"bytes=0", byteRange{0, 999, false}, true},       // no dash: ignored
		{" bytes=10-19 ", byteRange{10, 19, true}, true},  // whitespace
		{"bytes=-9223372036854775807", byteRange{0, 999, true}, true},
	}
	for _, c := range cases {
		got, ok := parseRange(c.header, size)
		assert.Equal(t, c.satisfy, ok, c.header)
		if ok {
			assert.Equal(t, c.want, got, c.header)
		}
	}
}

func TestDisk(t *testing.T) {
	root := t.TempDir()
	d := NewDisk(root)
	rel, err := d.Create(12)
	require.NoError(t, err)
	assert.Regexp(t, `^12/[a-f0-9]{40}\.part$`, rel)

	st, err := os.Stat(filepath.Join(root, "12"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), st.Mode().Perm())

	require.NoError(t, d.WriteAt(12, rel, 0, []byte("hello ")))
	require.NoError(t, d.WriteAt(12, rel, 6, []byte("world, and a stale tail")))
	// A retried chunk at the committed offset overwrites and cuts the stale tail.
	require.NoError(t, d.WriteAt(12, rel, 6, []byte("world")))
	fin, err := d.Finalize(12, rel, 11)
	require.NoError(t, err)
	assert.Regexp(t, `^12/[a-f0-9]{40}\.bin$`, fin)
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(fin)))
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(b))
	st, err = os.Stat(filepath.Join(root, filepath.FromSlash(fin)))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), st.Mode().Perm())

	// Size mismatch refuses to finalize.
	rel2, err := d.Create(12)
	require.NoError(t, err)
	_, err = d.Finalize(12, rel2, 5)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), root, "no path in errors")

	// Another instructor's path or a traversal is never touched.
	for _, bad := range []string{"13/" + strings.Repeat("a", 40) + ".bin", "12/../13/x.bin", "../etc/passwd", fin + "x"} {
		_, err := d.Open(12, bad)
		require.ErrorIs(t, err, ErrNotFound, bad)
		require.NoError(t, d.Remove(12, bad))
	}
	_, err = d.Open(13, fin)
	require.ErrorIs(t, err, ErrNotFound)

	f, err := d.Open(12, fin)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, d.Remove(12, fin))
	require.NoError(t, d.Remove(12, fin), "missing is fine")

	_, err = NewDisk("").Create(1)
	require.ErrorIs(t, err, ErrDisabled)
}

func TestLangFilesLoad(t *testing.T) {
	assert.Equal(t, "Upload complete", T("messages.ready", "en"))
	assert.NotEqual(t, "media.messages.ready", T("messages.ready", "fa"))
	assert.NotEmpty(t, attributes("fa"))
}
