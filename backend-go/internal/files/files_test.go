package files

import (
	"bytes"
	"crypto/sha256"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func vault(t *testing.T, root string, key []byte, prev ...[]byte) *Vault {
	t.Helper()
	v, err := NewVault(root, key, prev, false)
	require.NoError(t, err)
	return v
}

func strconvI(n int64) string { return strconv.FormatInt(n, 10) }

func k(s string) []byte { x := sha256.Sum256([]byte(s)); return x[:] }

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for i := range 8 {
		img.Set(i, i, color.NRGBA{R: 200, A: 255})
	}
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, img))
	return b.Bytes()
}

func TestRegistry(t *testing.T) {
	for _, name := range []string{PurposeRecordDocument, PurposeClaimDocument, PurposePlacePhoto, PurposePlaceLicence, PurposeProductImage} {
		p, ok := Lookup(name)
		require.True(t, ok, name)
		assert.Equal(t, "app/private/files/"+name, p.Dir())
		assert.True(t, strings.HasPrefix(p.Dir(), "app/private/"), "never under app/public")
		assert.Positive(t, p.MaxBytes)
		assert.Positive(t, p.QuotaFiles)
		assert.Positive(t, p.QuotaBytes)
		if p.Public {
			assert.Equal(t, []string{KindImage}, p.Kinds, "%s: a public purpose only holds images", name)
		}
	}
	assert.True(t, Registry[PurposePlacePhoto].Public)
	assert.True(t, Registry[PurposeProductImage].Public)
	assert.False(t, Registry[PurposeRecordDocument].Public)
	assert.False(t, Registry[PurposeClaimDocument].Public)
	assert.False(t, Registry[PurposePlaceLicence].Public)
	assert.False(t, Registry[PurposeProductImage].UserUpload, "product images are written by the Ritme team only")
	for _, name := range []string{PurposePlacePhoto, PurposePlaceLicence} {
		assert.False(t, Registry[name].UserUpload, "%s: opened by CB-DIR with place-ownership checks", name)
	}
	assert.True(t, Registry[PurposeRecordDocument].UserUpload)
	assert.True(t, Registry[PurposeClaimDocument].UserUpload)
	assert.Equal(t, "app/private/labs", Lab.Dir())
	assert.True(t, Lab.blobOnly)
}

// Round trip per purpose: encrypted at rest, 0600 / 0700, bound to purpose + owner + path.
func TestVaultRoundTripPerPurpose(t *testing.T) {
	root := t.TempDir()
	v := vault(t, root, k("k1"))
	for name, p := range Registry {
		plain := []byte("%PDF-1.4 secret-" + name)
		rel, err := v.Put(p, 7, plain)
		require.NoError(t, err, name)
		assert.Regexp(t, `^`+p.Dir()+`/7/[a-f0-9]{40}\.bin$`, rel)
		abs := filepath.Join(root, filepath.FromSlash(rel))
		raw, err := os.ReadFile(abs)
		require.NoError(t, err)
		assert.False(t, bytes.Contains(raw, []byte("secret")), "%s readable at rest", name)
		assert.True(t, bytes.HasPrefix(raw, []byte(p.magic)))
		st, err := os.Stat(abs)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), st.Mode().Perm())
		dir, err := os.Stat(filepath.Dir(abs))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), dir.Mode().Perm())
		got, err := v.Get(p, 7, rel)
		require.NoError(t, err)
		assert.Equal(t, plain, got)

		// Another owner, another purpose, a copy into another owner's directory: unreadable.
		_, err = v.Get(p, 8, rel)
		require.ErrorIs(t, err, ErrUnreadable)
		other := filepath.Join(root, filepath.FromSlash(p.Dir()), "8")
		require.NoError(t, os.MkdirAll(other, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(other, filepath.Base(rel)), raw, 0o600))
		_, err = v.Get(p, 8, p.Dir()+"/8/"+filepath.Base(rel))
		require.ErrorIs(t, err, ErrUnreadable)
	}
	// Same bytes moved across purposes do not open (additional data binds the purpose).
	rec, cl := Registry[PurposeRecordDocument], Registry[PurposeClaimDocument]
	rel, err := v.Put(rec, 7, []byte("doc"))
	require.NoError(t, err)
	raw, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	moved := cl.Dir() + "/7/" + filepath.Base(rel)
	require.NoError(t, os.WriteFile(filepath.Join(root, filepath.FromSlash(moved)), raw, 0o600))
	_, err = v.Get(cl, 7, moved)
	require.ErrorIs(t, err, ErrUnreadable)
	_, err = v.Get(cl, 7, rel)
	require.ErrorIs(t, err, ErrUnreadable, "a path of another purpose is refused")
}

func TestVaultOwnersAndTraversal(t *testing.T) {
	root := t.TempDir()
	v := vault(t, root, k("k1"))
	rec, photo := Registry[PurposeRecordDocument], Registry[PurposeProductImage]
	_, err := v.Put(rec, 0, []byte("x"))
	require.Error(t, err, "a private purpose needs a user")
	_, err = v.Put(Lab, 0, []byte("x"))
	require.Error(t, err)
	rel, err := v.Put(photo, 0, []byte("img"))
	require.NoError(t, err, "the platform may own public files")
	assert.True(t, strings.HasPrefix(rel, photo.Dir()+"/0/"))

	good, err := v.Put(rec, 7, []byte("x"))
	require.NoError(t, err)
	name := filepath.Base(good)
	for _, bad := range []string{rec.Dir() + "/7/../8/" + name, "../" + good, rec.Dir() + "/7/" + name + "/", "", "app/public/x.bin", rec.Dir() + "/7/x.bin"} {
		_, err = v.Get(rec, 7, bad)
		require.ErrorIs(t, err, ErrUnreadable, bad)
	}
	// Tampering breaks the tag.
	abs := filepath.Join(root, filepath.FromSlash(good))
	raw, _ := os.ReadFile(abs)
	raw[len(raw)-1] ^= 1
	require.NoError(t, os.WriteFile(abs, raw, 0o600))
	_, err = v.Get(rec, 7, good)
	require.ErrorIs(t, err, ErrUnreadable)
}

func TestVaultRotationDisabledAndRemoveUser(t *testing.T) {
	root := t.TempDir()
	rec := Registry[PurposeRecordDocument]
	old := vault(t, root, k("k1"))
	rel, err := old.Put(rec, 3, []byte("first"))
	require.NoError(t, err)
	got, err := vault(t, root, k("k2"), k("k1")).Get(rec, 3, rel)
	require.NoError(t, err)
	assert.Equal(t, []byte("first"), got)
	_, err = vault(t, root, k("k2")).Get(rec, 3, rel)
	require.ErrorIs(t, err, ErrUnreadable, "without the old key the file stays closed")

	off, err := NewVault(root, nil, nil, false)
	require.NoError(t, err)
	assert.True(t, off.Disabled(), "no key = disabled (fail closed)")
	_, err = off.Put(rec, 3, []byte("x"))
	require.ErrorIs(t, err, ErrDisabled)
	_, err = off.Get(rec, 3, rel)
	require.ErrorIs(t, err, ErrDisabled)
	require.NoError(t, off.Remove(rec, 3, rel), "removal never needs a key")
	_, err = os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	assert.True(t, os.IsNotExist(err))

	// RemoveUser: every purpose, lab included, never the platform or another user.
	for _, p := range Registry {
		_, err := old.Put(p, 4, []byte("x"))
		require.NoError(t, err)
		_, err = old.Put(p, 5, []byte("y"))
		require.NoError(t, err)
	}
	_, err = old.Put(Registry[PurposeProductImage], 0, []byte("z"))
	require.NoError(t, err)
	require.NoError(t, RemoveUser(root, 4))
	for _, p := range Registry {
		owners, err := Owners(root, p)
		require.NoError(t, err)
		assert.NotContains(t, owners, uint64(4), p.Name)
		assert.Contains(t, owners, uint64(5), p.Name)
	}
	_, err = os.Stat(filepath.Join(root, filepath.FromSlash(Registry[PurposeProductImage].Dir()), "0"))
	require.NoError(t, err, "platform files stay")
	require.NoError(t, RemoveUser(root, 0))
	require.NoError(t, RemoveUser("", 5))
}

func TestSigner(t *testing.T) {
	v := vault(t, t.TempDir(), k("k1"))
	s := NewSigner(v)
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	exp := now.Add(DefaultLinkTTL)
	f := File{ID: 10, Owner: 7, Purpose: PurposeRecordDocument, Path: "app/private/files/record_document/7/" + strings.Repeat("a", 40) + ".bin"}
	sig := s.Sign(f, exp)
	es := strconvI(exp.Unix())
	require.NoError(t, s.Verify(f, es, sig, now))

	// The same link on another file / another owner's file / another path / another purpose: invalid.
	for _, g := range []File{
		{ID: 11, Owner: 7, Purpose: f.Purpose, Path: f.Path},
		{ID: 10, Owner: 8, Purpose: f.Purpose, Path: f.Path},
		{ID: 10, Owner: 7, Purpose: f.Purpose, Path: strings.Replace(f.Path, "aaaa", "bbbb", 1)},
		{ID: 10, Owner: 7, Purpose: PurposeClaimDocument, Path: f.Path},
	} {
		require.ErrorIs(t, s.Verify(g, es, sig, now), ErrLinkInvalid, "%+v", g)
	}
	require.ErrorIs(t, s.Verify(f, strconvI(exp.Unix()+1), sig, now), ErrLinkInvalid, "a changed expiry")
	require.ErrorIs(t, s.Verify(f, "0"+es, sig, now), ErrLinkInvalid, "non-canonical expiry")
	require.ErrorIs(t, s.Verify(f, es, sig+"x", now), ErrLinkInvalid)
	require.ErrorIs(t, s.Verify(f, es, "", now), ErrLinkInvalid)
	require.ErrorIs(t, s.Verify(f, es, sig, exp.Add(time.Second)), ErrLinkExpired)
	require.ErrorIs(t, NewSigner(vault(t, t.TempDir(), k("k2"))).Verify(f, es, sig, now), ErrLinkInvalid, "another key")
	far := now.Add(MaxLinkTTL + time.Minute)
	require.ErrorIs(t, s.Verify(f, strconvI(far.Unix()), s.Sign(f, far), now), ErrLinkInvalid, "longer than MaxLinkTTL")
	var nilSigner *Signer
	require.ErrorIs(t, nilSigner.Verify(f, es, sig, now), ErrDisabled)
	off, _ := NewVault(t.TempDir(), nil, nil, true)
	assert.Nil(t, NewSigner(off))
}

func TestPrepare(t *testing.T) {
	pdf := []byte("%PDF-1.4 report\n%%EOF\n")
	rec, photo := Registry[PurposeRecordDocument], Registry[PurposePlacePhoto]
	out, mime, err := prepare(rec, pdf)
	require.NoError(t, err)
	assert.Equal(t, "application/pdf", mime)
	assert.Equal(t, pdf, out)

	_, _, err = prepare(photo, pdf)
	require.ErrorIs(t, err, ErrType, "a public purpose never stores a PDF")
	_, _, err = prepare(rec, []byte("hello"))
	require.ErrorIs(t, err, ErrType)
	_, _, err = prepare(rec, nil)
	require.ErrorIs(t, err, ErrType)
	_, _, err = prepare(Purpose{Kinds: []string{KindPDF}, MaxBytes: 4}, pdf)
	require.ErrorIs(t, err, ErrTooLarge)

	img, mime, err := prepare(photo, pngBytes(t))
	require.NoError(t, err)
	assert.Equal(t, "image/webp", mime, "photos are re-encoded")
	assert.True(t, bytes.HasPrefix(img, []byte("RIFF")))
	_, _, err = prepare(photo, append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, make([]byte, 32)...))
	require.ErrorIs(t, err, ErrType, "an unreadable image")
}

func TestTranslations(t *testing.T) {
	entries, err := langFS.ReadDir("lang")
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	for _, e := range entries {
		for _, key := range []string{"messages.not_found", "messages.link_invalid", "messages.link_expired", "validation.quota", "validation.file_type"} {
			assert.NotEqual(t, "files."+key, T(key, e.Name(), nil), "%s: %s", e.Name(), key)
		}
	}
}

func TestStrictPDF(t *testing.T) {
	for _, ok := range []string{"%PDF-1.7\nx\n%%EOF", "\xef\xbb\xbf%PDF-1.4 x %%EOF\n", "\r\n %PDF-1.4 x\n%%EOF\r\n"} {
		assert.True(t, strictPDF([]byte(ok)), "%q", ok)
	}
	long := append([]byte("%PDF-1.4 "), bytes.Repeat([]byte("x"), 2000)...)
	for _, bad := range [][]byte{
		[]byte("%PDF-1.4 no trailer"),
		[]byte("junk%PDF-1.4 x %%EOF"),
		append([]byte("%%EOF "), long...),
		[]byte("GIF89a %PDF-1.4 %%EOF"),
	} {
		assert.False(t, strictPDF(bad), "%q", bad[:min(len(bad), 30)])
	}
	_, _, err := prepare(Registry[PurposeRecordDocument], []byte("junk before %PDF-1.4 x\n%%EOF"))
	require.ErrorIs(t, err, ErrType, "junk before the header (the shared sniffer accepts it)")
}

func TestImageLimits(t *testing.T) {
	img := image.NewNRGBA64(image.Rect(0, 0, 4, 4))
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, img))
	assert.True(t, pngTooDeep(b.Bytes()))
	assert.False(t, pngTooDeep(pngBytes(t)))
	_, _, err := prepare(Registry[PurposeRecordDocument], b.Bytes())
	require.ErrorIs(t, err, ErrType, "16-bit PNG refused before decoding")
	assert.Equal(t, 20_000_000, MaxImagePixels)
}

func TestPathlessErrors(t *testing.T) {
	root := t.TempDir()
	v := vault(t, root, k("k1"))
	blocker := filepath.Join(root, "app")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600)) // "app" is a file: MkdirAll fails
	_, err := v.Put(Registry[PurposeRecordDocument], 7, []byte("x"))
	require.Error(t, err)
	assert.NotContains(t, err.Error(), root)
	assert.NotContains(t, err.Error(), "/7")
	_, err = Owners(filepath.Join(blocker, "nope"), Registry[PurposeRecordDocument])
	if err != nil {
		assert.NotContains(t, err.Error(), root)
	}
	assert.ErrorIs(t, pathless(&os.PathError{Op: "open", Path: "/secret/7/a.bin", Err: os.ErrNotExist}), os.ErrNotExist)
	assert.NotContains(t, pathless(&os.PathError{Op: "open", Path: "/secret/7/a.bin", Err: os.ErrNotExist}).Error(), "secret")
}
