// Package emergency is the emergency card «کارت اضطراری» (canvas-build CB-REC-03, D-72; artboard nbl_Rec_Emergency):
// «فقط همین اطلاعات روی کارت نمایش داده می‌شود؛ بقیه پرونده قفل می‌ماند».
//
// The card is read live from the health record (internal/healthrecord, share audience) and care: name, blood group,
// allergies (only while health_records.allergies_on_emergency_card is on, CB-REC-01), conditions, permanent
// medications (active care medications with duration «ongoing» plus the onboarding list), the pregnancy week only when
// the owner turns «نمایش وضعیت بارداری» on and a pregnancy is active (never a loss, a due date or anything else), a
// masked insurance placeholder (label + last 4 digits — a full number is never accepted) and one emergency contact.
// The owner stores only those settings (emergency_cards); show_on_lock_screen is a client flag («نمایش روی صفحه
// قفل — بدون نیاز به رمز»: the app keeps an offline copy for the lock screen).
//
// Public card: off by default. Only when the owner turns it on does she get a link token (256 random bits,
// base64url; only its SHA-256 is stored); GET /api/v1/emergency-cards/{token} is unauthenticated, IP-throttled,
// never cached or indexed, logged as "/api/v1/emergency-cards/*", and answers the minimal card — no insurance, no
// age, no ids. Turning it off or rotating kills the old token at once; an unknown or disabled token is the same 404.
package emergency

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Limits.
const (
	MaxContactName     = 60
	MaxContactRelation = 30
	MaxInsuranceLabel  = 60
)

// Error codes.
const (
	CodeNotFound = "emergency_card_not_found"
)

// Controller messages, validation lines and attribute names are data: lang/<code>/emergency.json (English fallback).
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

// T is the emergency line for key ("messages.saved") in locale.
func T(key, locale string) string { return translator().Trans("emergency."+key, nil, locale) }

func attributes(locale string) []string {
	v, ok := translator().Get("emergency.attributes", locale)
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
