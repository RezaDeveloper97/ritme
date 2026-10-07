package telemed

import (
	"database/sql"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/payments"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/telemed/store"
)

// BookingParts are the related rows a booking renders with.
type BookingParts struct {
	Doctor    store.TelemedDoctor
	ChildName string // "" when not a child booking (or the child was deleted)
	Scopes    []string
}

func isoOrNil(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return jsonx.ISO8601(wall(t.Time))
}

func nullString(s sql.NullString) any {
	if !s.Valid || s.String == "" {
		return nil
	}
	return s.String
}

// doctorBrief is the doctor block of a booking (name, specialty, photo).
func (v View) doctorBrief(d store.TelemedDoctor) *jsonx.OrderedMap {
	name := v.text(d.Name)
	nameStr, _ := name.(string)
	return jsonx.Obj(
		"id", d.ID,
		"kind", d.Kind,
		"name", name,
		"initial", Initial(nameStr),
		"specialty", v.code(GroupSpecialties, d.Specialty),
		"photo_url", v.photo(d.PhotoPath),
	)
}

// PriceJSON is a price block in rials.
func PriceJSON(price, discount uint64, source string, total uint64) *jsonx.OrderedMap {
	var src any
	if source != "" {
		src = source
	}
	return jsonx.Obj("price_rials", price, "discount_rials", discount, "discount_source", src, "total_rials", total,
		"currency", payments.Currency)
}

// ConsentJSON is {scope: bool} for every scope.
func ConsentJSON(active []string) *jsonx.OrderedMap {
	o := jsonx.Obj()
	for _, s := range Scopes {
		o.Set(s, slices.Contains(active, s))
	}
	return o
}

// BookingJSON is one booking (nbl_v17_ReviewBook summary / nbl_v17_Booked).
func (v View) BookingJSON(bk store.TelemedBooking, p BookingParts, now time.Time) *jsonx.OrderedMap {
	start, end := wall(bk.StartsAt), wall(bk.EndsAt)
	var child any
	if bk.ChildID.Valid && p.ChildName != "" {
		child = jsonx.Obj("id", bk.ChildID.Int64, "name", p.ChildName)
	}
	var reason any
	if bk.Reason.Valid && bk.Reason.String != "" {
		reason = v.code(GroupVisitReasons, bk.Reason.String)
	}
	var deadline any
	if bk.Status == StatusConfirmed {
		deadline = jsonx.ISO8601(CancelDeadline(bk))
	}
	var appt any
	if bk.AppointmentID.Valid {
		appt = bk.AppointmentID.Int64
	}
	var hold any
	if bk.Status == StatusHeld {
		hold = isoOrNil(bk.HoldExpiresAt)
	}
	return jsonx.Obj(
		"id", bk.ID,
		"reference", bk.Reference,
		"status", bk.Status,
		"mode", bk.Mode,
		"starts_at", jsonx.ISO8601(start),
		"ends_at", jsonx.ISO8601(end),
		"date", civildate.InTehran(start).String(),
		"time", start.Format("15:04"),
		"duration_minutes", bk.DurationMinutes,
		"hold_expires_at", hold,
		"doctor", v.doctorBrief(p.Doctor),
		"for_whom", bk.ForWhom,
		"child", child,
		"patient_name", nullString(bk.PatientName),
		"reason", reason,
		"note", nullString(bk.Note),
		"price", PriceJSON(bk.PriceRials, bk.DiscountRials, bk.DiscountSource.String, bk.TotalRials),
		"payment_status", bk.PaymentStatus,
		"cancel_deadline", deadline,
		"can_cancel", bk.Status == StatusHeld || Changeable(bk, now),
		"can_reschedule", Changeable(bk, now),
		"appointment_id", appt,
		"consent", ConsentJSON(p.Scopes),
		"href", BookingHref(bk.ID),
		"created_at", isoOrNil(bk.CreatedAt),
	)
}
