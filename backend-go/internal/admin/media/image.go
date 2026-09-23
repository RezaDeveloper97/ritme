package media

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"mime/multipart"

	"github.com/gen2brain/webp"
	xdraw "golang.org/x/image/draw"
)

// Formats the admin accepts (Laravel `mimes:jpeg,jpg,png,webp`).
const (
	FormatJPEG = "jpeg"
	FormatPNG  = "png"
	FormatWebP = "webp"
)

// Errors of the upload inspection.
var (
	// ErrNotImage: the bytes are not an image the `image` rule accepts (SVG included:
	// Laravel 12's `image` rule rejects SVG, and SVG can carry script).
	ErrNotImage = errors.New("media: not an image")
	// ErrWrongType: an image, but not one of jpeg/png/webp (`mimes`).
	ErrWrongType = errors.New("media: unsupported image type")
	// ErrTooLarge: the upload exceeds the size limit (`max:<KB>`).
	ErrTooLarge = errors.New("media: file too large")
)

// Image is an inspected upload.
type Image struct {
	Data   []byte
	Format string // FormatJPEG | FormatPNG | FormatWebP
	Width  int
	Height int
}

// Ext is the file extension Laravel's guessExtension() gives the format.
func (i Image) Ext() string {
	if i.Format == FormatJPEG {
		return "jpg"
	}
	return i.Format
}

// ReadUpload reads a multipart file, refusing more than maxBytes (ErrTooLarge).
func ReadUpload(fh *multipart.FileHeader, maxBytes int64) ([]byte, error) {
	if fh.Size > maxBytes {
		return nil, ErrTooLarge
	}
	f, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("media: open upload: %w", err)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("media: read upload: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, ErrTooLarge
	}
	return data, nil
}

// sniff identifies the format from the magic bytes. other=true for images the
// `image` rule knows but `mimes` refuses (gif, bmp).
func sniff(b []byte) (format string, other bool) {
	switch {
	case len(b) >= 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF:
		return FormatJPEG, false
	case bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		return FormatPNG, false
	case len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		return FormatWebP, false
	case bytes.HasPrefix(b, []byte("GIF87a")), bytes.HasPrefix(b, []byte("GIF89a")), bytes.HasPrefix(b, []byte("BM")):
		return "", true
	}
	return "", false
}

// Inspect identifies an upload by its bytes and reads its dimensions without decoding
// the pixels. It returns ErrNotImage or ErrWrongType for anything but a well-formed
// JPEG, PNG or WebP header.
func Inspect(data []byte) (Image, error) {
	format, other := sniff(data)
	if format == "" {
		if other {
			return Image{}, ErrWrongType
		}
		return Image{}, ErrNotImage
	}
	var (
		cfg image.Config
		err error
	)
	switch format {
	case FormatJPEG:
		cfg, err = jpeg.DecodeConfig(bytes.NewReader(data))
	case FormatPNG:
		cfg, err = png.DecodeConfig(bytes.NewReader(data))
	default:
		cfg, err = webp.DecodeConfig(bytes.NewReader(data))
	}
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return Image{}, ErrNotImage
	}
	return Image{Data: data, Format: format, Width: cfg.Width, Height: cfg.Height}, nil
}

// Optimizer is ImageOptimizer: scale into a MaxWidth×MaxHeight box (never up), apply the
// JPEG EXIF orientation, re-encode as WebP at Quality. Images above MaxPixels are not
// decoded (decompression-bomb guard) and are stored as uploaded.
type Optimizer struct {
	MaxWidth, MaxHeight, Quality, MaxPixels int
}

// DefaultOptimizer is ImageOptimizer's configuration.
var DefaultOptimizer = Optimizer{MaxWidth: 1080, MaxHeight: 1080, Quality: 82, MaxPixels: 30_000_000}

// errSkip means "store the original" (ImageOptimizer::optimize returning null).
var errSkip = errors.New("media: optimisation skipped")

// Optimize returns the WebP bytes of img, or errSkip when the image is too large to
// decode or cannot be decoded.
func (o Optimizer) Optimize(img Image) ([]byte, error) {
	if img.Width*img.Height > o.MaxPixels {
		return nil, errSkip
	}
	var (
		src image.Image
		err error
	)
	switch img.Format {
	case FormatJPEG:
		src, err = jpeg.Decode(bytes.NewReader(img.Data))
		if err == nil {
			src = orient(src, jpegOrientation(img.Data))
		}
	case FormatPNG:
		src, err = png.Decode(bytes.NewReader(img.Data))
	default:
		src, err = webp.Decode(bytes.NewReader(img.Data))
	}
	if err != nil {
		return nil, errSkip
	}
	scaled := o.scaleToFit(src)
	var buf bytes.Buffer
	if err := webp.Encode(&buf, scaled, webp.Options{Quality: o.Quality, Method: webp.DefaultMethod}); err != nil {
		return nil, fmt.Errorf("%w: webp encode: %w", errSkip, err) // PHP logs and keeps the original
	}
	return buf.Bytes(), nil
}

// scaleToFit is ImageOptimizer::scaleToFit: ratio = min(maxW/w, maxH/h, 1); target
// sides round(side·ratio) (PHP round, at least 1); alpha is kept.
func (o Optimizer) scaleToFit(src image.Image) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	ratio := math.Min(math.Min(float64(o.MaxWidth)/float64(w), float64(o.MaxHeight)/float64(h)), 1)
	if ratio >= 1 {
		return toNRGBA(src)
	}
	tw := max(1, int(math.Round(float64(w)*ratio)))
	th := max(1, int(math.Round(float64(h)*ratio)))
	dst := image.NewNRGBA(image.Rect(0, 0, tw, th))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Src, nil)
	return dst
}

func toNRGBA(src image.Image) *image.NRGBA {
	if n, ok := src.(*image.NRGBA); ok {
		return n
	}
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

// Store writes img to dir on the disk as ImageOptimizer::store does: the optimised WebP
// under a random name, or — when optimisation is skipped — the original bytes under a
// random name with the format's extension. It returns the relative path.
func (d *Disk) Store(o Optimizer, dir string, img Image) (string, error) {
	out, err := o.Optimize(img)
	if err != nil {
		return d.StoreOriginal(dir, img)
	}
	rel := NewPath(dir, "webp")
	return rel, d.Put(rel, out)
}

// StoreOriginal writes the upload untouched ($file->store($dir, 'public')).
func (d *Disk) StoreOriginal(dir string, img Image) (string, error) {
	rel := NewPath(dir, img.Ext())
	return rel, d.Put(rel, img.Data)
}
