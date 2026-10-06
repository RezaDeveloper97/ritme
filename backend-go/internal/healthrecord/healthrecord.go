// Package healthrecord is «پرونده سلامت من» (bloom B-N6-03, D-64; artboard nbl_Record_Summary): one owner-only summary
// that aggregates what other domains already store, plus the few user-owned facts nobody else keeps.
//
// Sources (read-only unless noted):
//
//   - basics: user_profiles height / weight (+ BMI), health_records.blood_type (user-owned; falls back to the pregnancy
//     profile's blood_type + rh_factor);
//   - conditions: user_life_profiles chronic_illnesses / gyn_conditions (edited through PUT /onboarding/steps/conditions);
//   - medications: active care medications (internal/care) + the life profile's medications list;
//   - allergies: health_records.allergies (user-owned);
//   - cycle: the analysis engine (internal/analysis BuildCycle / BuildSymptoms) over the last 6 months;
//   - vitals: vital_readings merged with the log sheet's day values (internal/vitals merge rule, D-63), last 30 days;
//   - pregnancies: the active pregnancy (ongoing), the postpartum birth, a count of pregnancy_losses (each only an
//     «ended» pregnancy — never a loss type, date, mood or the encrypted note, CB-LOSS privacy) and the entries the user
//     added by hand (health_record_pregnancies, user-owned);
//   - checkups: the latest checkup_records; labs: the latest ready lab_reports (internal/labs.Service.RecordLabs).
//
// Care appointments are not part of the record. If a later section adds them, rows whose care.Appointment.HiddenFrom
// is true (private loss follow-ups) must be dropped for AudienceShare and shown under neutral titles for the owner.
//
// Every section is built for an Audience: AudienceOwner (GET /health-record, the owner's own screen) or AudienceShare
// (the doctor report / share link of B-N6-04, which reuses Build): share drops free-text notes and every `editable`.
// There is no id parameter and no companion access: a request always reads the authenticated user's own rows.
package healthrecord

import (
	"embed"
	"io/fs"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Audience is who a record is built for.
type Audience string

// Audiences.
const (
	// AudienceOwner is the owner's own screen: everything she logged, user-owned sections editable.
	AudienceOwner Audience = "owner"
	// AudienceShare is a third party (doctor report, share link): no free-text notes, nothing editable.
	AudienceShare Audience = "share"
)

// Section keys, in screen order (nbl_Record_Summary).
const (
	SectionBasics      = "basics"
	SectionConditions  = "conditions"
	SectionMedications = "medications"
	SectionAllergies   = "allergies"
	SectionCycle       = "cycle"
	SectionVitals      = "vitals"
	SectionPregnancies = "pregnancies"
	SectionCheckups    = "checkups"
	SectionLabs        = "labs"
)

// Sections is every built-in section, in screen order.
var Sections = []string{
	SectionBasics, SectionConditions, SectionMedications, SectionAllergies, SectionCycle, SectionVitals,
	SectionPregnancies, SectionCheckups, SectionLabs,
}

// BloodTypes are the accepted health_records.blood_type values.
var BloodTypes = []string{"A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-"}

// Manual pregnancy outcomes: a birth (vaginal / cesarean) or a pregnancy that ended without one («ended», no detail).
const (
	OutcomeVaginal  = "vaginal"
	OutcomeCesarean = "cesarean"
	OutcomeEnded    = "ended"
)

// Outcomes are the manual entry outcomes.
var Outcomes = []string{OutcomeVaginal, OutcomeCesarean, OutcomeEnded}

// Pregnancy entry outcomes the record emits besides Outcomes: an active pregnancy, and a tracked birth whose delivery
// type was not told.
const (
	OutcomeOngoing = "ongoing"
	OutcomeBirth   = "birth"
)

// Entry sources of the pregnancies section.
const (
	SourceTracked = "tracked" // pregnancy / postpartum / loss rows of the app
	SourceManual  = "manual"  // health_record_pregnancies
)

// Limits.
const (
	MaxAllergies        = 20
	MaxAllergyLen       = 60
	MaxManualPregnancy  = 20
	MaxBabyCount        = 4
	DefaultVitalsDays   = 30
	DefaultCycleRange   = "6m"
	DefaultCheckupsList = 5
	DefaultLabsList     = 5
	TopSymptoms         = 3
)

// Controller messages, validation lines and attribute names are data: lang/<code>/healthrecord.json (English
// fallback). The screen copy lives in the clients' `health-record` translation namespace.
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

// T is the healthrecord line for key ("messages.saved") in locale.
func T(key, locale string) string { return translator().Trans("healthrecord."+key, nil, locale) }

// Tp is T with :param replacements.
func Tp(key string, params map[string]string, locale string) string {
	return translator().Trans("healthrecord."+key, params, locale)
}

// attributes are the validation attribute names for locale, as flat "key", "name" pairs.
func attributes(locale string) []string {
	v, ok := translator().Get("healthrecord.attributes", locale)
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
