// Package services is the «خدمات» hub (bloom B-N7-01, artboards nbl_/nbd_v17_Main): one composed read model for the
// services tab — which sections show and in which order, the health-care tiles, the care programs, the upcoming
// booking and the emergency card.
//
//	GET /api/v1/services   the hub for the current user (auth:api, Accept-Language)
//
// Everything editable lives in the content catalog (docs/canvas-build/catalog.md), three groups seeded by goose 00047:
//
//	services_sections   one item per section (codes in SectionCodes); is_active hides it, sort_order orders it; title /
//	                    body are the heading / sub-heading; meta {href?, caption?, phone?, categories?}
//	services_care       the «مراقبت سلامت» tiles; meta {icon, tone, href?, counter?}
//	services_programs   the «برنامه‌های مراقبتی» cards; meta {icon, tone, href?}; audiences = life modes that see it
//
// A tile, card or section with no valid internal href is «به‌زودی» (status soon): screens that do not exist yet are
// never linked and never faked. An admin flips one live by setting meta.href once its screen ships. Go only — no
// Laravel counterpart (deviations.md D-69). No health payload: the only user data read is the life mode (audience
// filter), the record's document count and the next booking (none until the telemedicine tasks, B-N7-03).
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Catalog groups of the hub.
const (
	GroupSections = "services_sections"
	GroupCare     = "services_care"
	GroupPrograms = "services_programs"
)

// Section codes the clients know how to render. Unknown codes in the catalog are skipped.
const (
	SectionSearch      = "search"
	SectionBooking     = "booking"
	SectionCare        = "care"
	SectionCheckups    = "checkups"
	SectionPrograms    = "programs"
	SectionMotherChild = "mother_child"
	SectionLearning    = "learning"
	SectionShop        = "shop"
	SectionEmergency   = "emergency"
)

// SectionCodes is the board order, also the fallback order when the sections group has no items at all.
var SectionCodes = []string{
	SectionSearch, SectionBooking, SectionCare, SectionCheckups, SectionPrograms,
	SectionMotherChild, SectionLearning, SectionShop, SectionEmergency,
}

// Statuses of a destination.
const (
	StatusLive = "live"
	StatusSoon = "soon"
)

// CounterRecordDocuments is the care-tile counter that shows the health record's document count.
const CounterRecordDocuments = "record_documents"

// DefaultEmergencyPhone is the emergency number when the catalog has none (or an invalid one).
const DefaultEmergencyPhone = "115"

// Tones the clients map to tokens (frontend shared/ui Tone). Anything else becomes "neutral".
var tones = map[string]bool{
	"brand": true, "data": true, "warm": true, "period": true, "bloom": true, "success": true, "danger": true, "neutral": true,
}

var (
	// hrefRe: an in-app path only ("/labs", "/children/new") — never a scheme, host or "//".
	hrefRe  = regexp.MustCompile(`^/[a-z0-9][a-z0-9/_-]{0,127}$`)
	iconRe  = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]{0,31}$`)
	phoneRe = regexp.MustCompile(`^[0-9]{3,6}$`)
)

// CatalogReader reads one catalog group's active items (catalog.Reader).
type CatalogReader interface {
	Items(ctx context.Context, group string) ([]catalog.Item, error)
}

// DocumentCounter counts the user's health-record documents (healthrecord store).
type DocumentCounter interface {
	CountRecordDocuments(ctx context.Context, userID uint64) (int64, error)
}

// ModeReader is the user's effective life mode (checkups.UserLifeMode).
type ModeReader func(ctx context.Context, userID uint64) (enums.LifeMode, error)

// Booking is the user's next booked visit (the telemedicine domain fills it, B-N7-03).
type Booking struct {
	ID              uint64
	Kind            string // video | phone | in_person
	ProviderName    string
	StartsAt        time.Time
	DurationMinutes int
	Href            string // in-app path of the booking's detail screen
}

// BookingSource returns the user's next upcoming booking after now, nil when there is none.
type BookingSource interface {
	NextBooking(ctx context.Context, userID uint64, now time.Time) (*Booking, error)
}

// Service builds the hub.
type Service struct {
	cat      CatalogReader
	docs     DocumentCounter
	mode     ModeReader
	bookings BookingSource
}

// NewService wires the hub. bookings may be nil (no telemedicine yet): upcoming_booking is then always null.
func NewService(cat CatalogReader, docs DocumentCounter, mode ModeReader, bookings BookingSource) *Service {
	return &Service{cat: cat, docs: docs, mode: mode, bookings: bookings}
}

// meta is the union of the documented meta keys of the three groups. Translatable values stay raw.
type meta struct {
	Href       string          `json:"href"`
	Icon       string          `json:"icon"`
	Tone       string          `json:"tone"`
	Counter    string          `json:"counter"`
	Caption    json.RawMessage `json:"caption"`
	Phone      string          `json:"phone"`
	Categories []struct {
		Code  string          `json:"code"`
		Icon  string          `json:"icon"`
		Title json.RawMessage `json:"title"`
	} `json:"categories"`
}

func metaOf(it catalog.Item) meta {
	var m meta
	if len(it.Meta) > 0 {
		_ = json.Unmarshal(it.Meta, &m) // admin JSON; a malformed meta just reads as empty (soon, defaults)
	}
	return m
}

// href is the validated in-app path, or nil.
func href(s string) any {
	if hrefRe.MatchString(s) {
		return s
	}
	return nil
}

func status(h any) string {
	if h != nil {
		return StatusLive
	}
	return StatusSoon
}

func icon(s, fallback string) string {
	if iconRe.MatchString(s) {
		return s
	}
	return fallback
}

func tone(s string) string {
	if tones[s] {
		return s
	}
	return "neutral"
}

// Hub is GET /services for one user at now.
func (s *Service) Hub(ctx context.Context, userID uint64, now time.Time, loc catalog.Localizer) (*jsonx.OrderedMap, error) {
	mode, err := s.mode(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("services: life mode: %w", err)
	}
	sections, err := s.sections(ctx, mode, loc)
	if err != nil {
		return nil, err
	}
	care, err := s.care(ctx, userID, mode, loc)
	if err != nil {
		return nil, err
	}
	programs, err := s.programs(ctx, mode, loc)
	if err != nil {
		return nil, err
	}
	var booking any
	if s.bookings != nil {
		b, err := s.bookings.NextBooking(ctx, userID, now)
		if err != nil {
			return nil, fmt.Errorf("services: next booking: %w", err)
		}
		if b != nil {
			booking = BookingJSON(b)
		}
	}
	return jsonx.Obj(
		"sections", sections,
		"upcoming_booking", booking,
		"care", care,
		"programs", programs,
	), nil
}

// BookingJSON is the upcoming-booking card.
func BookingJSON(b *Booking) *jsonx.OrderedMap {
	return jsonx.Obj(
		"id", b.ID,
		"kind", b.Kind,
		"provider_name", b.ProviderName,
		"starts_at", b.StartsAt.Format(time.RFC3339),
		"duration_minutes", b.DurationMinutes,
		"href", href(b.Href),
	)
}

// sections: the active known sections in catalog order (board order when the group is empty). The emergency card is
// a safety surface and always shows — appended at the end when an admin deactivated or deleted it. A teen never
// sees the shop (teen mode has no commercial surface, CB-TEEN-01).
func (s *Service) sections(ctx context.Context, mode enums.LifeMode, loc catalog.Localizer) ([]*jsonx.OrderedMap, error) {
	items, err := s.cat.Items(ctx, GroupSections)
	if err != nil {
		return nil, fmt.Errorf("services: sections: %w", err)
	}
	known := map[string]bool{}
	for _, c := range SectionCodes {
		known[c] = true
	}
	if len(items) == 0 {
		for _, c := range SectionCodes {
			items = append(items, catalog.Item{Code: c})
		}
	}
	out := make([]*jsonx.OrderedMap, 0, len(items)+1)
	seen := map[string]bool{}
	for _, it := range items {
		if !known[it.Code] || seen[it.Code] || !it.For(string(mode)) {
			continue
		}
		if it.Code == SectionShop && mode == enums.LifeModeTeen {
			continue
		}
		seen[it.Code] = true
		out = append(out, sectionJSON(it, loc))
	}
	if !seen[SectionEmergency] {
		out = append(out, sectionJSON(catalog.Item{Code: SectionEmergency}, loc))
	}
	return out, nil
}

func sectionJSON(it catalog.Item, loc catalog.Localizer) *jsonx.OrderedMap {
	m := metaOf(it)
	h := href(m.Href)
	var phone any
	if it.Code == SectionEmergency {
		phone = DefaultEmergencyPhone
		if phoneRe.MatchString(m.Phone) {
			phone = m.Phone
		}
	}
	cats := make([]*jsonx.OrderedMap, 0, len(m.Categories))
	for _, c := range m.Categories {
		if !catalog.ValidCode(c.Code, catalog.MaxCodeLen) {
			continue
		}
		cats = append(cats, jsonx.Obj("code", c.Code, "title", loc.Text(c.Title), "icon", icon(c.Icon, "box")))
	}
	return jsonx.Obj(
		"code", it.Code,
		"title", loc.Text(it.Title),
		"subtitle", loc.Text(it.Body),
		"caption", loc.Text(m.Caption),
		"href", h,
		"status", status(h),
		"phone", phone,
		"categories", cats,
	)
}

// care: the «مراقبت سلامت» tiles for the user's mode; the record tile carries the document count.
func (s *Service) care(ctx context.Context, userID uint64, mode enums.LifeMode, loc catalog.Localizer) ([]*jsonx.OrderedMap, error) {
	items, err := s.cat.Items(ctx, GroupCare)
	if err != nil {
		return nil, fmt.Errorf("services: care tiles: %w", err)
	}
	out := make([]*jsonx.OrderedMap, 0, len(items))
	var docs *int64
	for _, it := range items {
		if !it.For(string(mode)) {
			continue
		}
		m := metaOf(it)
		var count any
		if m.Counter == CounterRecordDocuments {
			if docs == nil {
				n, err := s.docs.CountRecordDocuments(ctx, userID)
				if err != nil {
					return nil, fmt.Errorf("services: record documents: %w", err)
				}
				docs = &n
			}
			count = *docs
		}
		out = append(out, tileJSON(it, m, loc, "count", count))
	}
	return out, nil
}

// programs: the care-program cards whose audiences include the user's mode.
func (s *Service) programs(ctx context.Context, mode enums.LifeMode, loc catalog.Localizer) ([]*jsonx.OrderedMap, error) {
	items, err := s.cat.Items(ctx, GroupPrograms)
	if err != nil {
		return nil, fmt.Errorf("services: programs: %w", err)
	}
	out := make([]*jsonx.OrderedMap, 0, len(items))
	for _, it := range items {
		if it.For(string(mode)) {
			out = append(out, tileJSON(it, metaOf(it), loc))
		}
	}
	return out, nil
}

func tileJSON(it catalog.Item, m meta, loc catalog.Localizer, extra ...any) *jsonx.OrderedMap {
	h := href(m.Href)
	o := jsonx.Obj(
		"code", it.Code,
		"title", loc.Text(it.Title),
		"subtitle", loc.Text(it.Body),
		"icon", icon(m.Icon, "grid"),
		"tone", tone(m.Tone),
		"href", h,
		"status", status(h),
	)
	for i := 0; i+1 < len(extra); i += 2 {
		o.Set(extra[i].(string), extra[i+1])
	}
	return o
}
