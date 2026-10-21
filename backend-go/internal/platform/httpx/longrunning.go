package httpx

import (
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// Long-running routes (T-M9-10). They are exempt from the per-request DB deadline (RequestDeadline) because their
// work legitimately outlasts it: AI calls, uploads and exports.
var (
	// longPrefixes: whole groups (instructor media upload chunks, tus-style PATCH bodies that are not multipart).
	longPrefixes = []string{"/api/instructor/v1/media"}
	// longExact: single AI routes (voice log transcription, ai.ProcessTimeout 40 s).
	longExact = map[string]bool{"/api/v1/logs/voice": true}
	// longPatterns: AI routes with an id in the path (assistant chat stream 90 s + summary 60 s, health-record
	// document extraction).
	longPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^/api/v1/assistant/conversations/[^/]+/(messages|summary)$`),
		regexp.MustCompile(`^/api/v1/health-record/documents/[^/]+/extract$`),
	}
)

// mediaTransferPrefixes are the routes nginx proxies without buffering (deploy/vhost-*.inc: upload chunks with
// proxy_request_buffering off, playback with proxy_buffering off), so the client's own speed reaches Go: they get the
// media read/write timeout instead of the listener's.
var mediaTransferPrefixes = []string{"/api/instructor/v1/media", "/api/v1/media/"}

// LongRunning reports whether the request is exempt from the per-request deadline: any multipart upload
// (POST/PUT/PATCH multipart/form-data), any `…/export` route (profile export, admin CSV exports) and the AI / media
// routes above.
func LongRunning(c fiber.Ctx) bool {
	return IsLongRunning(c.Method(), c.Path(), string(c.Request().Header.ContentType()))
}

// IsLongRunning is LongRunning on plain values.
func IsLongRunning(method, path, contentType string) bool {
	switch method {
	case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch:
		if strings.HasPrefix(strings.ToLower(contentType), fiber.MIMEMultipartForm) {
			return true
		}
	}
	p := normPath(path)
	if strings.HasSuffix(p, "/export") || longExact[p] {
		return true
	}
	for _, pre := range longPrefixes {
		if p == pre || strings.HasPrefix(p, pre+"/") {
			return true
		}
	}
	for _, re := range longPatterns {
		if re.MatchString(p) {
			return true
		}
	}
	return false
}

// MediaTransfer reports whether the request URI (path, optional query) is a media upload chunk or playback stream.
func MediaTransfer(requestURI string) bool {
	if i := strings.IndexByte(requestURI, '?'); i >= 0 {
		requestURI = requestURI[:i]
	}
	p := normPath(requestURI)
	for _, pre := range mediaTransferPrefixes {
		if p == strings.TrimSuffix(pre, "/") || strings.HasPrefix(p, strings.TrimSuffix(pre, "/")+"/") {
			return true
		}
	}
	return false
}

// normPath matches the router: case-insensitive, repeated slashes collapsed, no trailing slash.
func normPath(path string) string {
	p := strings.ToLower(path)
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	if len(p) > 1 {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}
