package ai

import "bytes"

// Central magic-byte sniffing of the files AI features send to a provider (B-N6-05b, L3). The type always comes
// from the bytes, never from the client's Content-Type or file name: handlers sniff an upload with SniffImage /
// SniffDocument before building the request, and the Client refuses (ErrInvalidRequest) an Image or Document
// whose bytes are not the MIME it claims — so a renamed HTML file, an SVG (script) or a zip never reaches a
// provider as «a photo». Audio has its own sniffing in internal/voicelog (container brands).
//
// Size limits line up with the API's 25 MB request body limit (RequestBodyLimit = cmd/api bodyLimit): one
// document ≤ MaxDocumentBytes, the photos of one chat request ≤ MaxChatImageBytes, both with room for multipart
// overhead and text.

// RequestBodyLimit mirrors the API's request body limit (cmd/api bodyLimit, 25 MB, the PHP stack's ceiling).
const RequestBodyLimit = 25 << 20

var (
	magicJPEG = []byte{0xFF, 0xD8, 0xFF}
	magicPNG  = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'}
	magicPDF  = []byte("%PDF-")
)

// SniffImage is image/jpeg | image/png | image/webp from the leading bytes, or "" for anything else.
func SniffImage(data []byte) string {
	switch {
	case bytes.HasPrefix(data, magicJPEG):
		return "image/jpeg"
	case bytes.HasPrefix(data, magicPNG):
		return "image/png"
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "image/webp"
	}
	return ""
}

// SniffDocument is SniffImage or application/pdf, or "" for anything else.
func SniffDocument(data []byte) string {
	if m := SniffImage(data); m != "" {
		return m
	}
	// A PDF may carry a few junk bytes before the header; readers accept it within the first KB.
	head := data[:min(len(data), 1024)]
	if bytes.Contains(head, magicPDF) {
		return "application/pdf"
	}
	return ""
}
