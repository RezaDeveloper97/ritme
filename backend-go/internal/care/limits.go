package care

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Per-user row caps (security audit M3-M7 #5). Without them one account could script
// unlimited rows, and /care/today and the lists compute over all of a user's rows. The caps
// sit far above real use; the per-user write throttle (internal/http) bounds the rate.
const (
	// MaxMedications caps a user's medication reminders, active or paused (a paused one is
	// still listed and can be switched back on).
	MaxMedications = 100
	// MaxUpcomingAppointments caps the scheduled (not cancelled) appointments still ahead.
	MaxUpcomingAppointments = 100
	// MaxAppointments is the hard ceiling on all appointment rows, past visits included
	// (scheduled_at may be in the past, so the upcoming cap alone doesn't bound the table).
	MaxAppointments = 1000
)

// ErrorCodeLimitReached is the 422's error_code when a cap is hit.
const ErrorCodeLimitReached = "limit_reached"

// limitReached is the controller-style 422 {success:false, message, errors:{field:[msg]},
// error_code:"limit_reached"}; message and the field error carry the same localized line.
func limitReached(locale, key string) error {
	msg := T(key, locale)
	return httpx.Fail(fiber.StatusUnprocessableEntity, msg,
		"errors", jsonx.Obj("limit", []string{msg}), "error_code", ErrorCodeLimitReached)
}

// checkMedicationCap refuses a new medication once the user has MaxMedications.
func (h *Handlers) checkMedicationCap(c fiber.Ctx, userID uint64, locale string) error {
	rows, err := h.q.ListMedications(c, userID)
	if err != nil {
		return fmt.Errorf("care: count medications: %w", err)
	}
	if len(rows) >= MaxMedications {
		return limitReached(locale, "messages.medication_limit")
	}
	return nil
}

// checkAppointmentCap refuses a new appointment once the user has MaxAppointments rows, or
// MaxUpcomingAppointments still ahead when the new one is itself ahead.
func (h *Handlers) checkAppointmentCap(c fiber.Ctx, userID uint64, at, now time.Time, locale string) error {
	rows, err := h.q.ListAppointments(c, userID)
	if err != nil {
		return fmt.Errorf("care: count appointments: %w", err)
	}
	if len(rows) >= MaxAppointments {
		return limitReached(locale, "messages.appointment_limit")
	}
	if at.Before(now) {
		return nil
	}
	if upcomingCount(rows, now) >= MaxUpcomingAppointments {
		return limitReached(locale, "messages.appointment_limit")
	}
	return nil
}

func upcomingCount(rows []store.Reminder, now time.Time) int {
	n := 0
	for _, r := range rows {
		if ParseAppointment(r).Upcoming(now) {
			n++
		}
	}
	return n
}
