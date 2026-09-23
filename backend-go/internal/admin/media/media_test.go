package media

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/gen2brain/webp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func solid(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x % 256), G: uint8(y % 256), B: 90, A: 255})
		}
	}
	return img
}

func encJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, jpeg.Encode(&b, img, &jpeg.Options{Quality: 90}))
	return b.Bytes()
}

func encPNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, img))
	return b.Bytes()
}

// withOrientation inserts an APP1 Exif segment carrying Orientation=o after SOI.
func withOrientation(jpg []byte, o uint16) []byte {
	tiff := []byte("MM\x00\x2a\x00\x00\x00\x08")
	ifd := make([]byte, 2+12+4)
	binary.BigEndian.PutUint16(ifd[0:], 1)
	binary.BigEndian.PutUint16(ifd[2:], 0x0112)
	binary.BigEndian.PutUint16(ifd[4:], 3)
	binary.BigEndian.PutUint32(ifd[6:], 1)
	binary.BigEndian.PutUint16(ifd[10:], o)
	payload := append([]byte("Exif\x00\x00"), append(tiff, ifd...)...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2)) //nolint:gosec // test data
	seg = append(seg, payload...)
	out := append([]byte{}, jpg[:2]...)
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

func TestInspect(t *testing.T) {
	img, err := Inspect(encJPEG(t, solid(40, 20)))
	require.NoError(t, err)
	assert.Equal(t, FormatJPEG, img.Format)
	assert.Equal(t, "jpg", img.Ext())
	assert.Equal(t, [2]int{40, 20}, [2]int{img.Width, img.Height})

	img, err = Inspect(encPNG(t, solid(8, 9)))
	require.NoError(t, err)
	assert.Equal(t, FormatPNG, img.Format)

	var wb bytes.Buffer
	require.NoError(t, webp.Encode(&wb, solid(30, 10), webp.Options{Quality: 80}))
	img, err = Inspect(wb.Bytes())
	require.NoError(t, err)
	assert.Equal(t, FormatWebP, img.Format)
	assert.Equal(t, 30, img.Width)

	_, err = Inspect([]byte("GIF89a\x01\x00\x01\x00"))
	assert.ErrorIs(t, err, ErrWrongType)
	for _, bad := range [][]byte{
		[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		[]byte("<?php echo 1; ?>"),
		[]byte("\xFF\xD8\xFF garbage that is not a jpeg"),
		[]byte("\x89PNG\r\n\x1a\n"),
		nil,
	} {
		_, err = Inspect(bad)
		assert.ErrorIs(t, err, ErrNotImage, "%q", bad)
	}
}

func TestOptimizeFitsBoxAndEncodesWebP(t *testing.T) {
	src, err := Inspect(encJPEG(t, solid(2000, 1000)))
	require.NoError(t, err)
	out, err := DefaultOptimizer.Optimize(src)
	require.NoError(t, err)
	got, err := Inspect(out)
	require.NoError(t, err)
	assert.Equal(t, FormatWebP, got.Format)
	assert.Equal(t, [2]int{1080, 540}, [2]int{got.Width, got.Height})

	small, err := Inspect(encPNG(t, solid(300, 200)))
	require.NoError(t, err)
	out, err = DefaultOptimizer.Optimize(small)
	require.NoError(t, err)
	got, err = Inspect(out)
	require.NoError(t, err)
	assert.Equal(t, [2]int{300, 200}, [2]int{got.Width, got.Height}, "never upscaled")
}

func TestOptimizeHonoursOrientation(t *testing.T) {
	src, err := Inspect(withOrientation(encJPEG(t, solid(60, 20)), 6))
	require.NoError(t, err)
	out, err := DefaultOptimizer.Optimize(src)
	require.NoError(t, err)
	got, err := Inspect(out)
	require.NoError(t, err)
	assert.Equal(t, [2]int{20, 60}, [2]int{got.Width, got.Height})
	assert.Equal(t, 0, jpegOrientation(encJPEG(t, solid(4, 4))))
	assert.Equal(t, 8, jpegOrientation(withOrientation(encJPEG(t, solid(4, 4)), 8)))
}

func TestOptimizeSkipsHugeImages(t *testing.T) {
	o := DefaultOptimizer
	o.MaxPixels = 100
	src, err := Inspect(encPNG(t, solid(20, 20)))
	require.NoError(t, err)
	_, err = o.Optimize(src)
	require.ErrorIs(t, err, errSkip)

	d := NewDisk(t.TempDir())
	rel, err := d.Store(o, "articles", src)
	require.NoError(t, err)
	assert.Regexp(t, `^articles/[A-Za-z0-9]{40}\.png$`, rel)
	b, err := os.ReadFile(filepath.Join(d.Root(), rel))
	require.NoError(t, err)
	assert.Equal(t, src.Data, b, "stored as uploaded")
}

func TestDiskPaths(t *testing.T) {
	root := t.TempDir()
	d := NewDisk(root)
	require.NoError(t, d.Put("banners/a.jpg", []byte("x")))
	fi, err := os.Stat(filepath.Join(root, "app", "public", "banners", "a.jpg"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), fi.Mode().Perm())
	assert.True(t, d.Exists("banners/a.jpg"))

	for _, bad := range []string{"", "/etc/passwd", "../x", "banners/../../x", "banners/./a", "a//b", ".env",
		"banners/.hidden", `banners\a`, "a\x00b", "banners/"} {
		assert.ErrorIs(t, d.Put(bad, []byte("x")), ErrBadPath, bad)
		assert.False(t, d.Exists(bad), bad)
		assert.NoError(t, d.Delete(bad), bad)
	}
	require.NoError(t, d.Delete("banners/a.jpg"))
	assert.False(t, d.Exists("banners/a.jpg"))
	require.NoError(t, d.Delete("banners/a.jpg"), "missing file is a no-op")

	assert.ErrorIs(t, NewDisk("").Put("a/b.jpg", nil), ErrBadPath)
}

func TestRandomName(t *testing.T) {
	re := regexp.MustCompile(`^[A-Za-z0-9]{40}$`)
	seen := map[string]bool{}
	for range 50 {
		n := RandomName(40)
		assert.Regexp(t, re, n)
		assert.False(t, seen[n])
		seen[n] = true
	}
	assert.Regexp(t, `^articles/[A-Za-z0-9]{40}\.webp$`, NewPath("/articles/", "webp"))
}
