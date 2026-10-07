package httpx

import "strings"

// sensitivePrefixes are route groups whose sub-paths say too much about a user's health to be written to the logs
// (CB-LOSS-01: /api/v1/loss/followup, /api/v1/loss/note …) or carry a bearer secret in the path (B-N6-04: the share
// token of /api/v1/shared-reports/{token} is also the report's decryption key; CB-REC-03: the public emergency card
// token of /api/v1/emergency-cards/{token}). Their requests are logged as "<prefix>/*".
var sensitivePrefixes = []string{"/api/v1/loss", "/api/v1/shared-reports", "/api/v1/emergency-cards"}

// LogPath is the request path as the logs may record it: a sensitive group collapses to "<prefix>/*" (no sub-path,
// no ids), everything else is unchanged.
// The match ignores case and repeated slashes (the router is case-insensitive, so /API/V1/Shared-Reports/x reaches
// the same handler).
func LogPath(path string) string {
	if p, ok := sensitivePrefix(path); ok {
		return p + "/*"
	}
	return path
}

// Sensitive reports whether path is in a sensitive group (CB-REC-03: the request log then drops the client IP too, so a
// public viewer's IP cannot be matched with the share access log).
func Sensitive(path string) bool {
	_, ok := sensitivePrefix(path)
	return ok
}

func sensitivePrefix(path string) (string, bool) {
	norm := strings.ToLower(path)
	for strings.Contains(norm, "//") {
		norm = strings.ReplaceAll(norm, "//", "/")
	}
	for _, p := range sensitivePrefixes {
		if norm == p || strings.HasPrefix(norm, p+"/") {
			return p, true
		}
	}
	return "", false
}
