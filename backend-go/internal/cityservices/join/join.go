// Package join is the business application of the mother & child directory — the 4-step «ثبت مجموعه» join of the
// canvas boards nbl_Dir_Join / JoinForm / JoinDocs / JoinDone (canvas CB-DIR-04, docs/canvas-build/directory.md §2).
// It sits next to bloom's internal/cityservices (B-N10-05) and only adds: its rows are city_service_applications
// (goose 00081_directory), its queries live in db/queries/cityservices_join (generated package ./store).
//
// Flow: an account opens a draft (one open draft at a time), auto-saves each step (PUT …/steps/{1..4} — lenient:
// types and lengths are checked, completeness only on submit), uploads place photos and private documents on the
// application's own routes, and submits → a random tracking code, `submitted`, and an SMS through the Notifier
// (fake provider: ids only in the log). The Ritme team reviews in admin (CB-DIR-05); nothing is published here and
// no payment exists (MVP, DECISIONS #13/#14).
//
// Security / privacy:
//   - Owner-only: every query carries user_id; another account's id is the same 404 as a missing one.
//   - Files go through internal/files (CB-CORE-05): photos are purpose place_photo_pending (private, re-encoded WebP
//     without EXIF / GPS, 5-minute signed links for the owner and the review team) until the team approves the
//     application, which promotes them to public place_photo files (a rejection deletes them; decision 2026-10-12);
//     documents purpose place_licence (private, AES-GCM at rest, 5-minute signed links for the owner; the review team
//     reads them in admin). Types are sniffed from the bytes, sizes and per-owner quotas are the
//     purpose registry's. A file id is only ever accepted from the application's own upload, never from the body.
//   - Tracking codes are 60 random bits (crypto/rand); they identify, they do not authorize (the status route is
//     owner-scoped too).
//   - No health / cycle / pregnancy field. Logs carry ids only — never a name, phone, address or tracking code.
package join

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"slices"
	"strings"
)

// Application statuses (the admin team moves submitted → in_review → approved | rejected | needs_info).
const (
	StatusDraft     = "draft"
	StatusSubmitted = "submitted"
	StatusInReview  = "in_review"
	StatusNeedsInfo = "needs_info"
	StatusApproved  = "approved"
	StatusRejected  = "rejected"
)

// Editable reports whether the applicant may still change the steps and documents.
func Editable(status string) bool { return status == StatusDraft || status == StatusNeedsInfo }

// PhotosOpen reports whether the applicant may still add photos (JoinDone «عکس‌های بیشتر اضافه کن»: also while the
// team reviews).
func PhotosOpen(status string) bool {
	return Editable(status) || status == StatusSubmitted || status == StatusInReview
}

// Document slots of step 4 (keys of the `documents` JSON).
const (
	SlotLicence        = "licence"
	SlotInstructorCert = "instructor_cert"
	SlotHealthCert     = "health_cert"
)

// Slots lists the document slots in board order.
var Slots = []string{SlotLicence, SlotInstructorCert, SlotHealthCert}

// BookingModes are the step-4 choices (`online` = the Ritme calendar; bookings are still confirmed by the team).
var BookingModes = []string{"online", "request", "phone"}

// ServiceKinds / ServiceFormats of a step-3 service.
var (
	ServiceKinds   = []string{"session", "package"}
	ServiceFormats = []string{"group", "private"}
)

// Catalog groups of the step-2 chips (seeded by 00081, admin-editable).
const (
	GroupAgeRanges = "dir_age_ranges"
	GroupAmenities = "dir_amenities"
)

// HealthCertCategory is the category whose places must upload the health certificate («برای استخر و فضاهای آبی»).
const HealthCertCategory = "pool"

// Limits.
const (
	MaxApplications = 10 // rows one account may hold (drafts + history)
	MinPhotos       = 4  // to submit («دست‌کم ۴ عکس واقعی»)
	MaxPhotos       = 20
	MaxCategories   = 8
	MaxServices     = 30
	MaxAbout        = 2000
	MaxPrice        = 1_000_000_000 // toman, display only
)

// Errors.
var (
	ErrNotFound   = errors.New("join: application not found")
	ErrLocked     = errors.New("join: application not editable")
	ErrTooMany    = errors.New("join: too many applications")
	ErrPhotoLimit = errors.New("join: photo limit")
	ErrStorage    = errors.New("join: storage unavailable")
)

// Hours is one open day: day 0 = Saturday (Jalali week), times HH:MM, close after open. A missing day is closed.
type Hours struct {
	Day   int    `json:"day"`
	Open  string `json:"open"`
	Close string `json:"close"`
}

// Offer is one step-3 service row (copied into city_service_services on approval). Optional fields are null.
type Offer struct {
	Kind                string  `json:"kind"`
	Name                string  `json:"name"`
	Description         *string `json:"description"`
	Format              *string `json:"format"`
	DurationMinutes     *int64  `json:"duration_minutes"`
	AgeMinMonths        *int64  `json:"age_min_months"`
	AgeMaxMonths        *int64  `json:"age_max_months"`
	Capacity            *int64  `json:"capacity"`
	PriceToman          *int64  `json:"price_toman"`
	PackageSessions     *int64  `json:"package_sessions"`
	PackageValidityDays *int64  `json:"package_validity_days"`
}

// Step1 is «معرفی».
type Step1 struct {
	BusinessName   string
	CategorySlugs  []string
	CityID         uint64 // 0 = none
	District       string
	ContactName    string
	ContactPhone   string // normalised 09xxxxxxxxx
	OwnerConfirmed bool
}

// Step2 is «مکان و تصاویر» (photos have their own routes).
type Step2 struct {
	Lat, Lng    string // decimal text, "" = none
	Address     string
	About       string
	AgeRanges   []string
	Amenities   []string
	WeeklyHours []Hours
}

// Step3 is «خدمات».
type Step3 struct{ Services []Offer }

// Step4 is «نحوه رزرو» + terms (documents have their own routes).
type Step4 struct {
	BookingMode   string // "" = none
	TermsAccepted bool
}

// trackingAlphabet has no 0/O/1/I so a code read over the phone is unambiguous; 32 symbols = 5 bits each.
const trackingAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

// NewTrackingCode returns 12 random symbols (60 bits) as XXXX-XXXX-XXXX (14 chars, fits varchar(16)).
func NewTrackingCode() (string, error) {
	var b strings.Builder
	n := big.NewInt(int64(len(trackingAlphabet)))
	for i := range 12 {
		if i > 0 && i%4 == 0 {
			b.WriteByte('-')
		}
		x, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", err
		}
		b.WriteByte(trackingAlphabet[x.Int64()])
	}
	return b.String(), nil
}

// Missing lists what a submit still needs, as field → validation key (lang …validation.<key>). photos is the number
// of live photo files; docs the live document slots. Pure, so it is unit-tested without a database.
func Missing(a Draft, photos int, docs map[string]bool) map[string]string {
	out := map[string]string{}
	need := func(ok bool, field, key string) {
		if !ok {
			out[field] = key
		}
	}
	need(a.BusinessName != "", "business_name", "required")
	need(len(a.CategorySlugs) > 0, "category_slugs", "required")
	need(a.CityID > 0, "city_id", "required")
	need(a.ContactName != "", "contact_name", "required")
	need(a.ContactPhone != "", "contact_phone", "required")
	need(a.OwnerConfirmed, "owner_confirmed", "owner_confirmed")
	need(a.Lat != "" && a.Lng != "", "location", "location")
	need(a.Address != "", "address", "required")
	need(len(a.AgeRanges) > 0, "age_ranges", "required")
	need(len(a.WeeklyHours) > 0, "weekly_hours", "required")
	need(photos >= MinPhotos, "photos", "min_photos")
	need(len(a.Services) > 0, "services", "required")
	need(a.BookingMode != "", "booking_mode", "required")
	need(a.TermsAccepted, "terms", "terms")
	need(docs[SlotLicence], "documents."+SlotLicence, "document_required")
	if slices.Contains(a.CategorySlugs, HealthCertCategory) {
		need(docs[SlotHealthCert], "documents."+SlotHealthCert, "document_required")
	}
	return out
}

// Draft is the decoded application (JSON columns parsed; a malformed column reads as empty).
type Draft struct {
	Step1
	Step2
	Step3
	Step4
	PhotoIDs  []uint64
	Documents map[string]uint64
}

func decode[T any](s string, v *T) {
	if s != "" {
		_ = json.Unmarshal([]byte(s), v) // a malformed column reads as empty
	}
}

func encode(v any) string {
	b, _ := json.Marshal(v) // plain slices / maps of strings and numbers
	return string(b)
}
