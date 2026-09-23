// Package care is the care-reminders API (docs/care-reminders/README.md): medications with
// per-dose intake tracking, doctor appointments and today's aggregate, under /api/v1/care.
// Go only — there is no Laravel counterpart. Rows live in `reminders` (type + meta JSON), so
// legacy /reminders readers keep seeing them; doses taken live in `reminder_intakes`.
package care

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Enum values (the labels are data: lang/<code>/care.json).
var (
	// Forms are the medication forms.
	Forms = []string{"tablet", "capsule", "syrup", "injection", "drops"}
	// Units are the suggested dose units; `unit` also accepts free text.
	Units = []string{"mg", "mcg", "ml", "iu", "drop"}
	// Durations are the medication durations.
	Durations = []string{DurationOngoing, DurationUntilDate, DurationPregnancyEnd}
	// AppointmentKinds are the appointment kinds (T-M3-02).
	AppointmentKinds = []string{"in_person", "phone", "online"}
	// AppointmentTopics are the appointment topics (T-M3-02).
	AppointmentTopics = []string{"ultrasound", "checkup", "lab", "consult", "vaccine", "other"}
	// RemindBefore are the appointment reminder offsets (T-M3-02).
	RemindBefore = []string{"1h", "3h", "1d", "2d"}
)

// Medication durations.
const (
	DurationOngoing      = "ongoing"
	DurationUntilDate    = "until_date"
	DurationPregnancyEnd = "pregnancy_end"
)

// Enums is GET /care/enums: localized {value, label} lists for the care forms.
func Enums(c fiber.Ctx) error {
	locale := i18n.Locale(c)
	return httpx.OK(c, jsonx.Obj(
		"forms", labeled("forms", locale, Forms),
		"units", labeled("units", locale, Units),
		"durations", labeled("durations", locale, Durations),
		"kinds", labeled("kinds", locale, AppointmentKinds),
		"topics", labeled("topics", locale, AppointmentTopics),
		"remind_before", labeled("remind_before", locale, RemindBefore),
	))
}
