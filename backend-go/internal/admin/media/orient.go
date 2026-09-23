package media

import (
	"encoding/binary"
	"image"
)

// jpegOrientation reads the EXIF Orientation tag (0x0112) of a JPEG's APP1 segment; 0 when
// absent or unreadable. Only the header segments are walked (up to the first scan).
func jpegOrientation(b []byte) int {
	if len(b) < 4 || b[0] != 0xFF || b[1] != 0xD8 {
		return 0
	}
	i := 2
	for i+4 <= len(b) {
		if b[i] != 0xFF {
			return 0
		}
		marker := b[i+1]
		if marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7) || marker == 0x01 || marker == 0xFF {
			i += 2
			if marker == 0xFF {
				i-- // fill byte
			}
			continue
		}
		if marker == 0xDA || marker == 0xD9 { // start of scan / end of image
			return 0
		}
		size := int(binary.BigEndian.Uint16(b[i+2 : i+4]))
		if size < 2 || i+2+size > len(b) {
			return 0
		}
		seg := b[i+4 : i+2+size]
		if marker == 0xE1 && len(seg) >= 6 && string(seg[:6]) == "Exif\x00\x00" {
			return tiffOrientation(seg[6:])
		}
		i += 2 + size
	}
	return 0
}

// tiffOrientation finds tag 0x0112 in IFD0 of a TIFF header.
func tiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 0
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 0
	}
	off := int(bo.Uint32(t[4:8]))
	if off < 8 || off+2 > len(t) {
		return 0
	}
	n := int(bo.Uint16(t[off : off+2]))
	for k := 0; k < n; k++ {
		e := off + 2 + 12*k
		if e+12 > len(t) {
			return 0
		}
		if bo.Uint16(t[e:e+2]) != 0x0112 {
			continue
		}
		if bo.Uint16(t[e+2:e+4]) != 3 { // SHORT
			return 0
		}
		return int(bo.Uint16(t[e+8 : e+10]))
	}
	return 0
}

// orient applies ImageOptimizer::autoRotate: orientation 3 → 180°, 6 → 90° clockwise,
// 8 → 90° counter-clockwise; everything else (incl. the mirrored variants) unchanged.
func orient(src image.Image, orientation int) image.Image {
	if orientation != 3 && orientation != 6 && orientation != 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	var dst *image.NRGBA
	if orientation == 3 {
		dst = image.NewNRGBA(image.Rect(0, 0, w, h))
	} else {
		dst = image.NewNRGBA(image.Rect(0, 0, h, w))
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := src.At(b.Min.X+x, b.Min.Y+y)
			switch orientation {
			case 3:
				dst.Set(w-1-x, h-1-y, c)
			case 6: // rotate 90° clockwise: (x, y) → (h-1-y, x)
				dst.Set(h-1-y, x, c)
			case 8: // rotate 90° counter-clockwise: (x, y) → (y, w-1-x)
				dst.Set(y, w-1-x, c)
			}
		}
	}
	return dst
}
