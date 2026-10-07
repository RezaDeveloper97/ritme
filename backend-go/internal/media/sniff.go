package media

import "bytes"

// SniffBytes is how much of the first chunk Sniff looks at.
const SniffBytes = 64

var (
	ebmlMagic = []byte{0x1A, 0x45, 0xDF, 0xA3}
	pdfMagic  = []byte("%PDF-")
)

// mp4 brands (ISO BMFF `ftyp` major brand) that are video files, and the audio-only ones.
var (
	videoBrands = map[string]bool{"isom": true, "iso2": true, "iso4": true, "iso5": true, "iso6": true, "mp41": true,
		"mp42": true, "avc1": true, "M4V ": true, "M4VH": true, "M4VP": true, "dash": true, "MSNV": true, "mmp4": true,
		"f4v ": true}
	audioBrands = map[string]bool{"M4A ": true, "M4B ": true, "mp42": true, "isom": true, "dash": true, "iso5": true,
		"iso6": true}
)

// Sniff returns the content type of a kind's file from its first bytes ("" = not an accepted file of that kind).
// It looks at the magic bytes only — the client's file name and Content-Type are never trusted.
func Sniff(kind string, head []byte) string {
	if len(head) > SniffBytes {
		head = head[:SniffBytes]
	}
	switch kind {
	case KindVideo:
		return sniffVideo(head)
	case KindAudio:
		return sniffAudio(head)
	case KindPDF:
		if bytes.HasPrefix(head, pdfMagic) {
			return "application/pdf"
		}
	}
	return ""
}

// ftypBrand is the major brand of an ISO BMFF file ("" when head is not one).
func ftypBrand(head []byte) string {
	if len(head) < 12 || string(head[4:8]) != "ftyp" {
		return ""
	}
	return string(head[8:12])
}

// webm reports whether head is an EBML file whose DocType is webm (Matroska .mkv is refused: browsers do not play it).
func webm(head []byte) bool {
	return bytes.HasPrefix(head, ebmlMagic) && bytes.Contains(head, []byte("webm"))
}

func sniffVideo(head []byte) string {
	switch b := ftypBrand(head); {
	case b == "qt  ":
		return "video/quicktime"
	case videoBrands[b]:
		return "video/mp4"
	case webm(head):
		return "video/webm"
	}
	return ""
}

func sniffAudio(head []byte) string {
	if b := ftypBrand(head); b != "" {
		if audioBrands[b] {
			return "audio/mp4"
		}
		return ""
	}
	switch {
	case bytes.HasPrefix(head, []byte("ID3")):
		return "audio/mpeg"
	case len(head) >= 2 && head[0] == 0xFF && head[1]&0xE0 == 0xE0:
		if head[1]&0x06 == 0 { // layer bits 00: an ADTS (AAC) header, not an MPEG audio frame
			return "audio/aac"
		}
		return "audio/mpeg"
	case bytes.HasPrefix(head, []byte("OggS")):
		return "audio/ogg"
	case len(head) >= 12 && string(head[:4]) == "RIFF" && string(head[8:12]) == "WAVE":
		return "audio/wav"
	case bytes.HasPrefix(head, []byte("fLaC")):
		return "audio/flac"
	case webm(head):
		return "audio/webm"
	}
	return ""
}
