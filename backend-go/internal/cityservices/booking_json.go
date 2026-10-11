package cityservices

// CB-DIR-03 booking JSON: the directory fields appended after bloom's booking shape (Dir_Booked / Dir_MyBookings).

import (
	"database/sql"
	"time"

	"github.com/ritme/backend-go/internal/platform/jsonx"
)

func (l Labeler) now() time.Time {
	if l.Now.IsZero() {
		return time.Now()
	}
	return l.Now
}

func nullFloat(f sql.NullFloat64) any {
	if !f.Valid {
		return nil
	}
	return f.Float64
}

func nullWall(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return slotTime(t.Time)
}

// directoryBooking appends code, service, slot, child (first name + age in months), price snapshot, status history,
// the cancel window, the upcoming flag, reschedule / review switches and the review.
func (l Labeler) directoryBooking(o *jsonx.OrderedMap, b Booking, now time.Time) {
	var service, slot, child, review any
	if b.ServiceID.Valid {
		service = jsonx.Obj("id", b.ServiceID.Int64, "name", nullStr(b.ServiceName), "duration_minutes", nullInt(b.ServiceDuration))
	}
	if b.SlotID.Valid {
		slot = jsonx.Obj("id", b.SlotID.Int64, "starts_at", nullWall(b.SlotStartsAt), "ends_at", nullWall(b.SlotEndsAt))
	}
	if b.ChildID.Valid && b.ChildName.Valid {
		child = jsonx.Obj("id", b.ChildID.Int64, "name", b.ChildName.String, "age_months", b.ChildAgeMonths())
	}
	if b.ReviewID.Valid {
		review = jsonx.Obj("id", b.ReviewID.Int64, "rating", nullInt(b.ReviewRating), "status", nullStr(b.ReviewStatus))
	}
	var until any
	if t, ok := b.CancelFreeUntil(); ok {
		until = slotTime(t)
	}
	o.Set("code", nullStr(b.Code))
	o.Set("service", service)
	o.Set("slot", slot)
	o.Set("child", child)
	o.Set("price_toman", nullInt(b.PriceToman))
	o.Set("is_upcoming", b.IsUpcoming(now))
	o.Set("cancel_free_hours", nullInt(b.CancelFreeHours))
	o.Set("cancel_free_until", until)
	o.Set("cancel_policy", nullStr(b.CancelPolicy))
	o.Set("can_reschedule", b.CanReschedule(now))
	o.Set("can_review", b.CanReview())
	o.Set("review", review)
	o.Set("confirmed_at", timeJSON(b.ConfirmedAt))
	o.Set("cancelled_at", timeJSON(b.CancelledAt))
	o.Set("cancelled_by", nullStr(b.CancelledBy))
	o.Set("completed_at", timeJSON(b.CompletedAt))
	if lj, ok := o.Get("listing"); ok {
		if m, ok := lj.(*jsonx.OrderedMap); ok {
			m.Set("lat", nullFloat(b.PlaceLat))
			m.Set("lng", nullFloat(b.PlaceLng))
		}
	}
}
