package httpx

import "strings"

// sensitivePrefixes are route groups whose sub-paths say too much about a user's health to be written to the logs
// (CB-LOSS-01: /api/v1/loss/followup, /api/v1/loss/note …) or carry a bearer secret in the path (B-N6-04: the share
// token of /api/v1/shared-reports/{token} is also the report's decryption key). Their requests are logged as "<prefix>/*".
var sensitivePrefixes = []string{"/api/v1/loss", "/api/v1/shared-reports"}

// LogPath is the request path as the logs may record it: a sensitive group collapses to "<prefix>/*" (no sub-path,
// no ids), everything else is unchanged.
func LogPath(path string) string {
	for _, p := range sensitivePrefixes {
		if path == p || strings.HasPrefix(path, p+"/") {
			return p + "/*"
		}
	}
	return path
}
