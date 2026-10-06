// Package sharelinks is the doctor report's 7-day share link (bloom B-N6-04, D-65; artboards nbl_Record_Export /
// nbl_Record_Preview).
//
// The owner picks a window, sections and an optional question; the API builds the health record for the share
// audience (internal/healthrecord.Service.BuildReport: no row ids, no free-text notes, losses only as a count),
// freezes it with the question into a snapshot and stores it encrypted. The link carries a random 256-bit token:
//
//   - the token is never stored, only its SHA-256 (hex) — the public lookup key;
//   - the snapshot is sealed with AES-256-GCM under a key derived from the token (HKDF-SHA256, info
//     "ritme-share-link:v1"), additional data = the token hash, so a stored row cannot be read without the link and a
//     ciphertext moved to another row does not open;
//   - a link lives LinkTTL (7 days), the owner may revoke it any time (the ciphertext is wiped at once), expired
//     ciphertexts are wiped by the purge loop and rows are deleted RetainAfterExpiry later;
//   - creation is Plus-gated (plus.PDFShare) and capped at MaxActive live links per user;
//   - the public read (GET /api/v1/shared-reports/{token}) is unauthenticated and IP-throttled: 404 for an unknown
//     token, 410 for an expired or revoked one.
//
// The owner's list in «حریم خصوصی و امنیت» shows only metadata (sections, window, dates, opens). No route takes an
// id of another user, there is no companion access, and nothing here logs a token or a health value.
package sharelinks

import (
	"embed"
	"io/fs"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/i18n/lang"
)

// Limits.
const (
	LinkTTL           = 7 * 24 * time.Hour
	MaxActive         = 10
	ListWindow        = 30 * 24 * time.Hour // the owner's list: links created in the last 30 days
	RetainAfterExpiry = 30 * 24 * time.Hour // rows are deleted this long after they expired
	PurgeEvery        = time.Hour
)

// Link statuses.
const (
	StatusActive  = "active"
	StatusExpired = "expired"
	StatusRevoked = "revoked"
)

// Error codes.
const (
	CodeNotFound = "share_link_not_found"
	CodeExpired  = "share_link_expired"
	CodeRevoked  = "share_link_revoked"
	CodeLimit    = "share_link_limit"
)

// Controller messages and attribute names are data: lang/<code>/sharelinks.json (English fallback). The screen copy
// lives in the clients' `recordExport` translation namespace.
//
//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	return t
})

// T is the sharelinks line for key ("messages.created") in locale.
func T(key, locale string) string { return translator().Trans("sharelinks."+key, nil, locale) }

// Tp is T with :param replacements.
func Tp(key string, params map[string]string, locale string) string {
	return translator().Trans("sharelinks."+key, params, locale)
}
