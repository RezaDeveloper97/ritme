package cityservices

import (
	"database/sql"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// CardInfo is what CB-DIR-02 computes per card (directory.go Rank + Store.Extras).
type CardInfo struct {
	Distance          *float64
	OpenKnown, IsOpen bool
	AgeGiven          bool
	AgeKnown, AgeFits bool
	PriceFrom         *int64
	NextSlot          *time.Time
}

// InfoOf builds a card's CardInfo from a ranked hit and the page's extras.
func InfoOf(r Ranked, ageGiven bool, ex CardExtras) CardInfo {
	ci := CardInfo{Distance: r.Distance, OpenKnown: r.OpenKnown, IsOpen: r.IsOpen, AgeGiven: ageGiven,
		AgeKnown: r.AgeKnown, AgeFits: r.AgeFits}
	if p, ok := ex.PriceFrom[r.ID]; ok {
		ci.PriceFrom = &p
	}
	if t, ok := ex.NextSlot[r.ID]; ok {
		ci.NextSlot = &t
	}
	return ci
}

// PlaceDetail is the place page's extra blocks.
type PlaceDetail struct {
	Services  []Service
	NextSlots []Slot
	Reviews   ReviewSummary
}

// listingPhotos prefers the structured place photos (public files) and falls back to bloom's public-disk photos.
func (l Labeler) listingPhotos(x Listing) []string {
	if len(x.FilePhotoURLs) > 0 {
		return x.FilePhotoURLs
	}
	return l.photos(x.Photos)
}

// FilePhotoURLs resolves a listing's photo_file_ids through paths (Store.PhotoPaths), keeping the stored order and
// skipping ids that are not public place photos.
func FilePhotoURLs(appURL string, ids []uint64, paths map[uint64]string) []string {
	out := []string{}
	for _, id := range ids {
		if p, ok := paths[id]; ok {
			out = append(out, strings.TrimRight(appURL, "/")+"/api/v1/files/public/"+strconv.FormatUint(id, 10)+"/"+files.Name(p))
		}
	}
	return out
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }

func nullRound1(f sql.NullFloat64) any {
	if !f.Valid {
		return nil
	}
	return round1(f.Float64)
}

func nullInt(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

func slotTime(t time.Time) string {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, civildate.Tehran).
		Format("2006-01-02T15:04:05-07:00")
}

func list(ss []string) []string {
	if ss == nil {
		return []string{}
	}
	return ss
}

// safeURL keeps an http(s) URL only.
func safeURL(s sql.NullString) any {
	v := strings.TrimSpace(s.String)
	if !s.Valid || (!strings.HasPrefix(v, "https://") && !strings.HasPrefix(v, "http://")) {
		return nil
	}
	return v
}

func (l Labeler) directoryCard(o *jsonx.OrderedMap, x Listing, ci CardInfo) {
	var lat, lng, dist, rating, open, fit, price, next any
	if x.Lat.Valid && x.Lng.Valid {
		lat, lng = x.Lat.Float64, x.Lng.Float64
	}
	if ci.Distance != nil {
		dist = round1(*ci.Distance)
	}
	if x.RatingCount > 0 {
		rating = nullRound1(x.RatingAvg)
	}
	if ci.OpenKnown {
		open = ci.IsOpen
	}
	if ci.AgeGiven && ci.AgeKnown {
		fit = ci.AgeFits
	}
	if ci.PriceFrom != nil {
		price = *ci.PriceFrom
	}
	if ci.NextSlot != nil {
		next = slotTime(*ci.NextSlot)
	}
	o.Set("lat", lat)
	o.Set("lng", lng)
	o.Set("distance_km", dist)
	o.Set("verified", x.Verified)
	o.Set("rating_avg", rating)
	o.Set("rating_count", x.RatingCount)
	o.Set("age_ranges", list(x.AgeRanges))
	o.Set("amenities", list(x.Amenities))
	o.Set("price_from_toman", price)
	o.Set("open_now", open)
	o.Set("next_slot_at", next)
	o.Set("age_fit", fit)
}

func (l Labeler) directoryDetail(o *jsonx.OrderedMap, x Listing, d PlaceDetail) {
	hours := make([]*jsonx.OrderedMap, 0, len(x.WeeklyHours))
	for _, h := range x.WeeklyHours {
		hours = append(hours, jsonx.Obj("day", h.Day, "open", h.Open, "close", h.Close))
	}
	svc := make([]*jsonx.OrderedMap, 0, len(d.Services))
	for _, v := range d.Services {
		svc = append(svc, ServiceJSON(v))
	}
	slots := make([]*jsonx.OrderedMap, 0, len(d.NextSlots))
	for _, s := range d.NextSlots {
		slots = append(slots, SlotJSON(s))
	}
	latest := make([]*jsonx.OrderedMap, 0, len(d.Reviews.Latest))
	for _, r := range d.Reviews.Latest {
		latest = append(latest, jsonx.Obj("id", r.ID, "rating", r.Rating, "body", nullStr(r.Body),
			"author_first_name", nullStr(r.Author), "created_at", timeJSON(r.CreatedAt)))
	}
	var avg any
	if d.Reviews.Count > 0 {
		avg = nullRound1(d.Reviews.Avg)
	}
	o.Set("about", nullStr(x.About))
	o.Set("website_url", safeURL(x.WebsiteURL))
	o.Set("weekly_hours", hours)
	o.Set("rules", nullStr(x.Rules))
	o.Set("cancel_free_hours", nullInt(x.CancelFreeHours))
	o.Set("cancel_policy", nullStr(x.CancelPolicy))
	o.Set("services", svc)
	o.Set("next_slots", slots)
	o.Set("reviews", jsonx.Obj("rating_avg", avg, "rating_count", d.Reviews.Count,
		"scores", jsonx.Obj("cleanliness", nullRound1(d.Reviews.Cleanliness), "instructor", nullRound1(d.Reviews.Instructor),
			"amenities", nullRound1(d.Reviews.Amenities)),
		"latest", latest))
}

// ServiceJSON is one service / package of a place (prices as the venue declared them, display only).
func ServiceJSON(v Service) *jsonx.OrderedMap {
	return jsonx.Obj("id", v.ID, "kind", v.Kind, "name", v.Name, "description", nullStr(v.Description),
		"format", nullStr(v.Format), "duration_minutes", nullInt(v.DurationMinutes), "age_min_months", nullInt(v.AgeMinMonths),
		"age_max_months", nullInt(v.AgeMaxMonths), "capacity", nullInt(v.Capacity), "price_toman", nullInt(v.PriceToman),
		"package_sessions", nullInt(v.PackageSessions), "package_validity_days", nullInt(v.PackageValidityDays))
}

// SlotJSON is one open time; is_full = «پر شد».
func SlotJSON(s Slot) *jsonx.OrderedMap {
	var ends any
	if s.EndsAt.Valid {
		ends = slotTime(s.EndsAt.Time)
	}
	left := max(s.Capacity-s.BookedCount, 0)
	return jsonx.Obj("id", s.ID, "service_id", nullInt(s.ServiceID), "starts_at", slotTime(s.StartsAt), "ends_at", ends,
		"capacity", s.Capacity, "seats_left", left, "is_full", left == 0)
}

// CatalogOptionJSON is a dir_amenities / dir_age_ranges item for the filter sheet.
func CatalogOptionJSON(loc catalog.Localizer, it catalog.Item) *jsonx.OrderedMap {
	return jsonx.Obj("code", it.Code, "name", loc.Text(it.Title), "meta", loc.Meta(it.Meta))
}
