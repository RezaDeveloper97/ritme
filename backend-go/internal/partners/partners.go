// Package partners is the home lab / screening booking request flow (bloom B-N10-04, D-94; docs/night-bloom/extras.md
// §3): an admin catalog of orderable services (home blood draw, annual checkup / Pap), the users' booking requests with
// a small status machine, the LabPartner adapter (Submit / Status / Cancel / FetchResult; `fake` by default) and the
// optional copy of a result into labs.
//
// Privacy. What the user types (name, phone, city, address, note) is sealed in one blob (PRIVATE_NOTE_KEY, bound to
// owner + reference) and wiped 30 days after the request closed. The partner receives the allow-listed Submission only
// — never a user id, account mobile, health log or profile data — after an explicit consent that names the partner.
// Results are not stored: they are fetched from the partner when the user imports them, and only when she ticked the
// import consent on the request AND confirms again at result time. Nothing here reads a health log, so the Today
// «پیشنهاد» cards are driven by the catalog's active flag alone.
package partners

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Request statuses.
const (
	StatusRequested   = "requested"
	StatusAccepted    = "accepted"
	StatusSampled     = "sampled"
	StatusResultReady = "result_ready"
	StatusRejected    = "rejected"
	StatusCancelled   = "cancelled"
)

// Service kinds (what the Today card says).
const (
	KindHomeDraw = "home_draw"
	KindCheckup  = "checkup"
)

// Preferred time windows (the client labels them).
const (
	WindowMorning = "morning"
	WindowNoon    = "noon"
	WindowEvening = "evening"
)

// Limits and policy.
const (
	MaxOpenRequests = 3                   // active requests per user
	MaxDaysAhead    = 60                  // preferred day: today … today+60
	PurgeAfter      = 30 * 24 * time.Hour // the personal blob is wiped this long after the request closed
	PurgeEvery      = time.Hour
	SyncEvery       = 10 * time.Minute
	MaxNote         = 300
	MaxAddress      = 300
	DefaultPartner  = "fake"
)

// Kinds, Windows and Statuses are the accepted values.
var (
	Kinds    = []string{KindHomeDraw, KindCheckup}
	Windows  = []string{WindowMorning, WindowNoon, WindowEvening}
	Statuses = []string{StatusRequested, StatusAccepted, StatusSampled, StatusResultReady, StatusRejected, StatusCancelled}
)

// Error codes (the `error_code` extra of a failed response).
const (
	CodeNotFound          = "lab_request_not_found"
	CodeServiceUnavail    = "lab_service_unavailable"
	CodeTooManyOpen       = "lab_too_many_open"
	CodePartnerUnavail    = "lab_partner_unavailable"
	CodeStorage           = "lab_storage_unavailable"
	CodeNotCancellable    = "lab_not_cancellable"
	CodeNoResult          = "lab_result_not_ready"
	CodeImportConsent     = "lab_import_consent_required"
	CodeAlreadyImported   = "lab_already_imported"
	CodeBadTransition     = "lab_bad_transition"
	CodeManualUnsupported = "lab_manual_status_unsupported"
)

// Errors of the service.
var (
	ErrNotFound          = errors.New("partners: request not found")
	ErrServiceUnavail    = errors.New("partners: service not available")
	ErrTooManyOpen       = errors.New("partners: too many open requests")
	ErrPartnerUnavail    = errors.New("partners: partner unavailable")
	ErrStorage           = errors.New("partners: sealed storage unavailable")
	ErrNotCancellable    = errors.New("partners: request cannot be cancelled")
	ErrNoResult          = errors.New("partners: no result to import")
	ErrImportConsent     = errors.New("partners: import consent missing")
	ErrAlreadyImported   = errors.New("partners: result already imported")
	ErrBadTransition     = errors.New("partners: bad status transition")
	ErrManualUnsupported = errors.New("partners: partner does not take manual statuses")
)

// Controller messages, validation lines and attribute names are data: lang/<code>/partners.json (English fallback).
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

// T is the partners line for key ("messages.requested") in locale.
func T(key, locale string) string { return translator().Trans("partners."+key, nil, locale) }

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	v, ok := translator().Get("partners.attributes", locale)
	m, isMap := v.(phpval.Map)
	if !ok || !isMap {
		return nil
	}
	var kv []string
	for _, k := range m.Keys() {
		if s, ok := m.Get(k); ok {
			if str, ok := s.(string); ok {
				kv = append(kv, k, str)
			}
		}
	}
	return kv
}

// Open reports whether a status is still being worked on.
func Open(status string) bool {
	return status == StatusRequested || status == StatusAccepted || status == StatusSampled
}

// CanCancel reports whether the user may still cancel (before the sample is taken).
func CanCancel(status string) bool { return status == StatusRequested || status == StatusAccepted }

var rank = map[string]int{StatusRequested: 0, StatusAccepted: 1, StatusSampled: 2, StatusResultReady: 3}

// CanMove reports whether the partner side may move a request from -> to: forward along requested → accepted →
// sampled → result_ready (steps may be skipped), or to rejected while nothing was sampled. Cancelled is the user's.
func CanMove(from, to string) bool {
	switch to {
	case StatusRejected:
		return from == StatusRequested || from == StatusAccepted
	case StatusAccepted, StatusSampled, StatusResultReady:
		rf, okf := rank[from]
		rt, okt := rank[to]
		return okf && okt && rt > rf
	}
	return false
}

func contains(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

// PickText picks the text of a localized JSON column {"fa":…,"en":…}: locale, then the default language, then fa,
// then any non-empty value. A plain (non-JSON) string is returned as is.
func PickText(raw, locale, def string) string {
	var m map[string]string
	if json.Unmarshal([]byte(raw), &m) != nil {
		return raw
	}
	for _, k := range []string{locale, def, "fa", "en"} {
		if v := m[k]; v != "" {
			return v
		}
	}
	for _, v := range m {
		if v != "" {
			return v
		}
	}
	return ""
}
