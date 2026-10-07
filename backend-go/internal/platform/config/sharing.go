package config

import "strings"

// MinSharePepperLen is the shortest SHARE_CODE_PEPPER accepted in production (bytes).
const MinSharePepperLen = 32

// Sharing holds the record-sharing settings (canvas-build CB-REC-03, internal/sharelinks).
type Sharing struct {
	// CodePepper (SHARE_CODE_PEPPER, secret, server .env only) keys the HMAC of the 24h doctor codes and the key that
	// wraps a link token under its code, so a stored row alone never opens a report even though a code is short.
	// Required in production: when it is empty there, creating and opening codes answers 503 (fail closed) instead of
	// using the public development default. Changing it invalidates the live (≤ 24 h) codes.
	CodePepper string
	// WebURL (SHARE_WEB_URL, optional) is the public origin of the web app (https://web.example) the QR code of a 24h
	// link points at: {WebURL}/{locale}/shared/report/{token}. Empty → the API returns only the path and the client
	// prefixes its own origin.
	WebURL string
}

// PepperMissing reports whether production runs without a pepper (doctor codes are then disabled).
func (s Sharing) PepperMissing(app App) bool { return app.IsProduction() && s.CodePepper == "" }

func loadSharing(e *env, app App) Sharing {
	s := Sharing{
		CodePepper: e.str("SHARE_CODE_PEPPER", ""),
		WebURL:     strings.TrimRight(e.str("SHARE_WEB_URL", ""), "/"),
	}
	if s.CodePepper != "" && app.IsProduction() && len(s.CodePepper) < MinSharePepperLen {
		e.fail("SHARE_CODE_PEPPER: must be at least %d bytes in production", MinSharePepperLen)
	}
	if s.WebURL != "" {
		if err := checkPublicURL(s.WebURL, app.IsProduction()); err != nil {
			e.fail("SHARE_WEB_URL: %v", err)
		}
	}
	return s
}
