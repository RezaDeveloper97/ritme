package files

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func box(t *testing.T, root string, key []byte, prev ...[]byte) *Box {
	t.Helper()
	b, err := New(root, key, prev, false)
	require.NoError(t, err)
	return b
}

func TestRoundTripAndAtRest(t *testing.T) {
	root := t.TempDir()
	k := sha256.Sum256([]byte("k1"))
	b := box(t, root, k[:])
	plain := []byte("%PDF-1.4 Ferritin 9 ng/mL")
	rel, err := b.Put(7, plain)
	require.NoError(t, err)
	assert.Regexp(t, `^app/private/labs/7/[a-f0-9]{40}\.bin$`, rel)

	abs := filepath.Join(root, filepath.FromSlash(rel))
	raw, err := os.ReadFile(abs)
	require.NoError(t, err)
	assert.False(t, bytes.Contains(raw, []byte("Ferritin")), "the sheet must not be readable at rest")
	assert.True(t, bytes.HasPrefix(raw, []byte(magic)))
	st, err := os.Stat(abs)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), st.Mode().Perm())
	dir, err := os.Stat(filepath.Dir(abs))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dir.Mode().Perm())

	got, err := b.Get(7, rel)
	require.NoError(t, err)
	assert.Equal(t, plain, got)
}

func TestForeignTamperedAndRotated(t *testing.T) {
	root := t.TempDir()
	k1, k2 := sha256.Sum256([]byte("k1")), sha256.Sum256([]byte("k2"))
	b := box(t, root, k1[:])
	rel, err := b.Put(7, []byte("sheet"))
	require.NoError(t, err)

	// Another user's id: the path does not belong to them.
	_, err = b.Get(8, rel)
	require.ErrorIs(t, err, ErrUnreadable)
	// Copied into user 8's directory: the additional data binds the owner.
	other := filepath.Join(root, "app/private/labs/8")
	require.NoError(t, os.MkdirAll(other, 0o700))
	raw, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	name := filepath.Base(rel)
	require.NoError(t, os.WriteFile(filepath.Join(other, name), raw, 0o600))
	_, err = b.Get(8, "app/private/labs/8/"+name)
	require.ErrorIs(t, err, ErrUnreadable)
	// Traversal and foreign names are refused before touching the disk.
	for _, bad := range []string{"app/private/labs/7/../8/" + name, "../" + rel, "app/private/support-reports/x.webp", ""} {
		_, err = b.Get(7, bad)
		require.ErrorIs(t, err, ErrUnreadable, bad)
	}
	// Tampering breaks the tag.
	raw[len(raw)-1] ^= 1
	require.NoError(t, os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), raw, 0o600))
	_, err = b.Get(7, rel)
	require.ErrorIs(t, err, ErrUnreadable)

	// Rotation: a file sealed with k1 opens with k1 as a previous key; new files use k2.
	rel2, err := b.Put(7, []byte("second"))
	require.NoError(t, err)
	rotated := box(t, root, k2[:], k1[:])
	got, err := rotated.Get(7, rel2)
	require.NoError(t, err)
	assert.Equal(t, []byte("second"), got)
	_, err = box(t, root, k2[:]).Get(7, rel2)
	require.ErrorIs(t, err, ErrUnreadable, "without the old key the file stays closed")
}

func TestDisabledRemoveAndUsers(t *testing.T) {
	root := t.TempDir()
	off, err := New(root, nil, nil, true)
	require.NoError(t, err)
	assert.True(t, off.Disabled())
	_, err = off.Put(1, []byte("x"))
	require.ErrorIs(t, err, ErrDisabled)
	_, err = off.Get(1, "app/private/labs/1/"+string(bytes.Repeat([]byte("a"), 40))+".bin")
	require.ErrorIs(t, err, ErrDisabled)

	dev := box(t, root, nil) // development key
	rel, err := dev.Put(3, []byte("x"))
	require.NoError(t, err)
	_, err = dev.Put(4, []byte("y"))
	require.NoError(t, err)
	ids, err := Users(root)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint64{3, 4}, ids)

	require.NoError(t, dev.Remove(3, rel))
	require.NoError(t, dev.Remove(3, rel), "a missing file is fine")
	_, err = dev.Get(3, rel)
	require.ErrorIs(t, err, ErrUnreadable)

	require.NoError(t, RemoveUser(root, 4))
	_, err = os.Stat(filepath.Join(root, "app/private/labs/4"))
	assert.True(t, os.IsNotExist(err))
	require.NoError(t, RemoveUser(root, 0), "no user: no-op")
	require.NoError(t, RemoveUser("", 4), "no storage: no-op")
}
