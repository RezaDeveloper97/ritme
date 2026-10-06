// Package labs is the lab analysis «تحلیل آزمایش» (bloom B-N6-06; artboards nbl_Lab_* in c-health-record and
// nbl_An_Labs): upload a lab sheet, read its markers with the AI extractor, let the user verify / edit / add them,
// explain the result in plain language and follow each marker across labs.
//
// Flow (one lab_reports row):
//
//	POST /labs (multipart, Plus plus.lab_ai via the AI gate) → files encrypted at rest (internal/labs/files)
//	  → job extract (lab_jobs, worker.go) → status extracting → needs_review (markers with confidence)
//	PUT/POST/DELETE /labs/{id}/markers… → the user fixes what was misread
//	POST /labs/{id}/verify → job interpret → interpreting → ready (summary from the AI client, else rules)
//	POST /labs/manual → typed-in markers, verified at once, rules interpretation (no AI, not Plus)
//
// Interpretation (interpret.go): per-marker status against the sheet's own reference range (the catalog's
// typical range only when the sheet has none and the unit matches), red flags from the catalog's thresholds,
// doctor questions and «when to see a doctor» from the catalog, and a short plain-language summary written by the
// AI client with the user's context (age band, life mode, approximate cycle phase, medication names) under
// non-diagnostic guardrails; any AI failure falls back to a rules summary, so a lab never stays stuck.
//
// Marker catalog: catalog_items group lab_markers (admin-editable, CB-CORE-03), seeded by migration 00034 with
// common markers, needs_review. Trends: every verified value of a marker (catalog code, else the normalised
// printed name) across the user's labs, oldest first.
//
// Privacy: the sheet files are encrypted (AES-256-GCM, LAB_FILE_KEY), private, deletable, removed with the lab and
// the account; only the file, the schema and a language hint go to the extractor; the interpretation prompt
// carries no name or id (the AI client redacts names / numbers on top). Nothing health-related is logged.
package labs

import (
	"embed"
	"errors"
	"io/fs"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Lab sources.
const (
	SourceUpload = "upload"
	SourceManual = "manual"
)

// Lab statuses.
const (
	StatusQueued       = "queued"
	StatusExtracting   = "extracting"
	StatusNeedsReview  = "needs_review"
	StatusInterpreting = "interpreting"
	StatusReady        = "ready"
	StatusFailed       = "failed"
)

// Marker sources.
const (
	MarkerExtracted = "extracted"
	MarkerEdited    = "edited"
	MarkerManual    = "manual"
)

// Categories are the lab types of the upload form (nbl_Lab_Upload «نوع آزمایش»).
var Categories = []string{"blood", "hormone", "thyroid", "urine", "other"}

// Limits.
const (
	MaxFiles = 5 // pages / files per upload
	// MaxImageBytes / MaxPDFBytes are the per-file limits (the AI platform's: ai.MaxImageBytes, ai.MaxDocumentBytes).
	MaxImageBytes = ai.MaxImageBytes
	MaxPDFBytes   = ai.MaxDocumentBytes
	// MaxUploadBytes bounds the whole multipart body (inside the API's 25 MB limit).
	MaxUploadBytes = 20 << 20
	// MaxImagePixels refuses decompression bombs before decoding.
	MaxImagePixels = 40_000_000
	// MaxMarkers per lab.
	MaxMarkers = 80
	// MaxInterpretations per lab (the first after verify + re-interpretations after edits).
	MaxInterpretations = 3
	// MaxLabsListed caps GET /labs.
	MaxLabsListed = 100
	// UploadsPerDay is a per-user cap on top of the monthly Plus quota (abuse / cost guard).
	UploadsPerDay = 10
	// LowConfidence marks a value the extractor was unsure about (nbl_Lab_Verify «با اطمینان کم»).
	LowConfidence = 0.7
	// MaxNameLen etc. are the column sizes.
	MaxNameLen     = 120
	MaxUnitLen     = 24
	MaxRefTextLen  = 64
	MaxValueText   = 32
	MaxTitleLen    = 120
	MaxFeedbackLen = 500
	// MaxValue bounds numeric values (decimal(14,4)).
	MaxValue = 9_999_999_999.0
)

// Domain errors (mapped by handlers.go).
var (
	ErrNotFound       = errors.New("labs: lab not found")
	ErrMarkerNotFound = errors.New("labs: marker not found")
	ErrFileNotFound   = errors.New("labs: file not found")
	ErrBusy           = errors.New("labs: the lab is being processed")
	ErrNotReviewable  = errors.New("labs: nothing to verify")
	ErrNoMarkers      = errors.New("labs: no markers")
	ErrInterpretLimit = errors.New("labs: interpretation limit reached")
	ErrTooManyMarkers = errors.New("labs: too many markers")
	ErrStorage        = errors.New("labs: file storage unavailable")
	ErrDailyLimit     = errors.New("labs: daily upload limit reached")
)

// The messages and sentences are data: lang/<code>/labs.json (English fallback).
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

// T is the labs line for key ("messages.deleted") in locale, with :params.
func T(key, locale string, params map[string]string) string {
	return translator().Trans("labs."+key, params, locale)
}

// num writes n in the locale's digit set (lang "digits"; «۵» in Persian sentences).
func num(n int, locale string) string {
	digits := []rune(T("digits", locale, nil))
	s := strconv.Itoa(n)
	if len(digits) != 10 {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		b.WriteRune(digits[r-'0'])
	}
	return b.String()
}

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	line, ok := translator().Get("labs.attributes", locale)
	m, isMap := line.(phpval.Map)
	if !ok || !isMap {
		return nil
	}
	var kv []string
	for _, k := range m.Keys() {
		if v, ok := m.Get(k); ok {
			if s, ok := v.(string); ok {
				kv = append(kv, k, s)
			}
		}
	}
	return kv
}

// tehranSecond is t in Tehran truncated to the second (DB wall-clock).
func tehranSecond(t time.Time) time.Time { return t.In(civildate.Tehran).Truncate(time.Second) }
