// Package telemed is the doctors directory «پزشکان و ماماها» (bloom B-N7-02, deviation D-67; artboards nbl_v17_Doctors
// and nbl_v17_DoctorProfile): admin-managed doctors and midwives with their visit types (video / phone / in person,
// duration and price in rials), weekly availability and time off, the free slots generated from them (slots.go), and
// reviews from users who completed a visit.
//
//	GET    /api/v1/telemed/doctors                         the directory (filters, paged)
//	GET    /api/v1/telemed/doctors/filters                 filter chips with counts
//	GET    /api/v1/telemed/doctors/{id}                    profile, visit types, stats, latest reviews
//	GET    /api/v1/telemed/doctors/{id}/slots              free slots of one visit mode, per day
//	GET    /api/v1/telemed/doctors/{id}/reviews            visible reviews (paged)
//	POST   /api/v1/telemed/doctors/{id}/reviews            review a completed visit (VisitChecker)
//	DELETE /api/v1/telemed/doctors/{id}/reviews/{review}   delete one's own review
//	/api/admin/v1/telemed/*                                admin CRUD (admin.go, docs/go-migration/admin-api.md §19)
//
// Specialties, cities and insurers are catalog_items codes (groups telemed_specialties / telemed_cities /
// telemed_insurers): their labels are admin content read through catalog.Reader. Booking (B-N7-03) plugs into the two
// hooks: Busy (booked or held intervals remove slots) and VisitChecker (a completed, not yet reviewed visit allows a
// review). Until then no slot is busy and nobody may review.
package telemed

import (
	"context"
	"errors"
	"slices"
	"time"
)

// Doctor kinds (telemed_doctors.kind). The honorific («دکتر» / «خانم», "Dr.") is client copy chosen by kind.
const (
	KindDoctor  = "doctor"
	KindMidwife = "midwife"
)

// Kinds in display order.
var Kinds = []string{KindDoctor, KindMidwife}

// Visit modes (telemed_visit_types.mode), in display order.
const (
	ModeVideo    = "video"
	ModePhone    = "phone"
	ModeInPerson = "in_person"
)

// Modes in display order.
var Modes = []string{ModeVideo, ModePhone, ModeInPerson}

// ValidMode reports whether m is a visit mode.
func ValidMode(m string) bool { return slices.Contains(Modes, m) }

// Catalog groups holding the labels of the codes stored on doctors.
const (
	GroupSpecialties = "telemed_specialties"
	GroupCities      = "telemed_cities"
	GroupInsurers    = "telemed_insurers"
)

// InsurerAny is the insurance filter value "accepts at least one insurer" (the «بیمه» chip).
const InsurerAny = "any"

// Limits.
const (
	// LeadMinutes: a slot must start at least this long after now to be offered.
	LeadMinutes = 30
	// HorizonDays: slots are generated up to this many days ahead (today included).
	HorizonDays = 60
	// NextSlotDays: how far ahead the directory looks for a doctor's first free slot.
	NextSlotDays = 30
	// MaxSlotDays: the largest `days` window of GET …/slots.
	MaxSlotDays = 31
	// DefaultSlotDays: the default `days` window (one week strip of the profile).
	DefaultSlotDays = 7
	// DirectoryCap: the most doctors one directory query considers (the "today" filter runs in Go over them).
	DirectoryCap = 1000
	// PerPage: directory page size; ReviewsPerPage: reviews page size; PreviewReviews on the profile.
	PerPage        = 20
	ReviewsPerPage = 20
	PreviewReviews = 3
	// MaxReviewLen: review body characters.
	MaxReviewLen = 1000
	// PositiveRating: ratings at or above this count towards the «رضایت» percent.
	PositiveRating = 4
)

// Errors mapped to 403 / 404 / 409 by the handlers.
var (
	ErrDoctorNotFound   = errors.New("telemed: doctor not found")
	ErrReviewNotFound   = errors.New("telemed: review not found")
	ErrReviewNotAllowed = errors.New("telemed: no completed visit to review")
	ErrReviewExists     = errors.New("telemed: visit already reviewed")
)

// Interval is a half-open [Start, End) span of Tehran wall-clock time.
type Interval struct {
	Start, End time.Time
}

// Overlaps reports whether i and o share any instant.
func (i Interval) Overlaps(o Interval) bool { return i.Start.Before(o.End) && o.Start.Before(i.End) }

// Busy reports the intervals in which doctors cannot take a new visit (booked or held slots). B-N7-03 implements it
// over its bookings; the result is keyed by doctor id and may omit doctors with nothing busy.
type Busy interface {
	Busy(ctx context.Context, doctorIDs []uint64, from, to time.Time) (map[uint64][]Interval, error)
}

// NoBusy is the Busy of a directory without bookings: nothing is busy.
type NoBusy struct{}

// Busy implements Busy.
func (NoBusy) Busy(context.Context, []uint64, time.Time, time.Time) (map[uint64][]Interval, error) {
	return nil, nil
}

// VisitChecker finds a completed visit of userID with doctorID that has no review yet. ok=false means the user may not
// review (now). B-N7-03 implements it over its bookings; bookingID must be > 0 (one review per booking).
type VisitChecker interface {
	ReviewableVisit(ctx context.Context, userID, doctorID uint64) (bookingID uint64, ok bool, err error)
}

// NoVisits is the VisitChecker of a directory without bookings: nobody may review.
type NoVisits struct{}

// ReviewableVisit implements VisitChecker.
func (NoVisits) ReviewableVisit(context.Context, uint64, uint64) (uint64, bool, error) {
	return 0, false, nil
}
