// Package loss is the pregnancy loss path (CB-LOSS-01, boards nbl_Loss_Start / _Care / _Next): recording a loss
// durably stops pregnancy content (bloom's pregnancy mode goes off, open pregnancy alerts close, pregnancy-only
// reminders pause, a forced pregnancy message mode is ignored), the follow-up (bleeding until it stops, beta hCG until
// negative, a visit about 2 weeks later — dated ones are M3 care appointments), daily mood check-ins, an encrypted
// private note and the next step (cycle / ttc / nothing → life-stage mode). Go only, under /api/v1/loss.
//
// Privacy: everything is scoped to the signed-in owner. A companion learns only one line, and only when she asks for
// it and the companion holds the pregnancy grant. Nothing about a loss is logged or counted: errors carry no health
// detail and there are no analytics events.
package loss

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Loss types (catalog loss_types codes; «ترجیح می‌دهم نگویم» = unspecified, also the default).
const (
	TypeEarlyMiscarriage = "early_miscarriage"
	TypeLateMiscarriage  = "late_miscarriage"
	TypeEctopic          = "ectopic"
	TypeChemical         = "chemical"
	TypeUnspecified      = "unspecified"
)

// Types are the loss types in board order.
var Types = []string{TypeEarlyMiscarriage, TypeLateMiscarriage, TypeEctopic, TypeChemical, TypeUnspecified}

// Moods are the «امروز چطوری؟» chips (catalog loss_moods codes).
var Moods = []string{"sad", "numb", "angry", "a_bit_better"}

// Next steps (catalog loss_next_steps codes).
const (
	NextCycle   = "cycle"
	NextTTC     = "ttc"
	NextNothing = "nothing"
)

// NextSteps are the «از اینجا به بعد» options in board order.
var NextSteps = []string{NextCycle, NextTTC, NextNothing}

// Catalog groups of the loss copy (seeded by goose 00028, admin-editable, needs clinical review).
const (
	GroupSupport          = "loss_support"
	GroupFollowups        = "loss_followups"
	ItemCompanionNotice   = "companion_notice"
	ItemFollowupBeta      = "beta"
	ItemFollowupVisit     = "visit"
	CompanionNoticeAction = "/companion"
)

// Clinical constants [needs clinical review].
const (
	// VisitAfterDays is when the follow-up visit is suggested («حدود ۲ هفته بعد»), from the loss date.
	VisitAfterDays = 14
	// RecurrentLosses is how many recorded losses show the «سقط دوم یا سوم» hint.
	RecurrentLosses = 2
	// MoodDays is how many days of mood check-ins the state returns (today included).
	MoodDays = 14
	// BetaHour is the time of day a beta-test reminder appointment is set for (the user picks the day).
	BetaHour = 9
	// MaxNoteLen caps the private note (characters).
	MaxNoteLen = 5000
	// MaxPastDays bounds how far back a loss date may be entered.
	MaxPastDays = 365
)

// The loss copy (controller messages, validation lines and attribute names) is data: lang/<code>/loss.json, one file
// per language, read through the platform translator (missing language/key → English). Lists and clinical copy are
// catalog content, not here.
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

// T is the loss line for key ("messages.saved") in locale.
func T(key, locale string) string { return translator().Trans("loss."+key, nil, locale) }

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	line, ok := translator().Get("loss.attributes", locale)
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
