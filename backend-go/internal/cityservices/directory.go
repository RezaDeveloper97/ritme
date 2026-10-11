package cityservices

// Directory reads of CB-DIR-02 (canvas boards nbl_Dir_Home / List / Map / Place, docs/canvas-build/directory.md):
// search with q, category, child age, open-now, amenities, a lat/lng radius with distance, a map bbox, plus the place
// detail (services, next slots, reviews summary) and its open slots. Built ON bloom's B-N10-05 public listing routes —
// the same /api/v1/city-services/listings[/{id}] grow optional params and additive fields; there is no /places family.
//
// Ranking (sort=rank, the default) mixes distance, age fit and the review rating — never money: no column a venue can
// pay for takes part (the board promises «هیچ مجموعه‌ای برای بالاتر آمدن پول نمی‌دهد»). bloom's admin sort_order only
// breaks exact ties, so an unrated directory without a position keeps bloom's order.
//
// Privacy: only `published` + active listings of an active city and category are ever read here; owner_user_id,
// applications, licence files and booking rows are never selected. Review authors appear as a first name only.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/catalog"
)

// Listing statuses (CB-DIR-01). Only published listings are public.
const (
	ListingDraft     = "draft"
	ListingPending   = "pending"
	ListingPublished = "published"
	ListingSuspended = "suspended"
)

// ModeOnline is CB-DIR-01's extra booking mode: pick a slot; still a REQUEST the Ritme team confirms.
const ModeOnline = "online"

// Sort orders of the public list.
const (
	SortRank     = "rank"
	SortDistance = "distance"
	SortRating   = "rating"
)

// Sorts lists the valid sort orders.
var Sorts = []string{SortRank, SortDistance, SortRating}

// Directory limits.
const (
	MaxCandidates   = 1000 // rows ranked in memory per search (the directory is admin-curated, small)
	MaxRadiusKm     = 100
	MaxAgeMonths    = 216 // 18 years
	MaxQueryLen     = 80
	MaxAmenities    = 10
	NextSlotsShown  = 8
	ReviewsShown    = 3
	DefaultSlotDays = 14
	MaxSlotDays     = 31
	MaxSlotsShown   = 200
	ratingPrior     = 4.0 // Bayesian prior: an unrated place counts as 4.0 over ratingPriorN votes
	ratingPriorN    = 5
	publicWhere     = " WHERE l.is_active = 1 AND l.status = 'published' AND c.is_active = 1 AND t.is_active = 1"
	catalogAmenity  = "dir_amenities"
	catalogAgeRange = "dir_age_ranges"
)

// DayHours is one opening window of weekly_hours; Day 0 = Saturday (Jalali week).
type DayHours struct {
	Day   int    `json:"day"`
	Open  string `json:"open"`
	Close string `json:"close"`
}

// Dir holds a listing's directory columns (CB-DIR-01). Zero values = not filled (legacy bloom row).
type Dir struct {
	Status          string
	Verified        bool
	Lat, Lng        sql.NullFloat64
	About           sql.NullString
	AgeRanges       []string
	Amenities       []string
	WeeklyHours     []DayHours
	PhotoFileIDs    []uint64
	WebsiteURL      sql.NullString
	CancelFreeHours sql.NullInt64
	CancelPolicy    sql.NullString
	Rules           sql.NullString
	RatingAvg       sql.NullFloat64
	RatingCount     int
	// FilePhotoURLs are the resolved public URLs of PhotoFileIDs (filled by the handler); they win over bloom's photos.
	FilePhotoURLs []string
}

const dirColumns = `, l.status, l.verified_at IS NOT NULL, l.lat, l.lng, l.about, l.age_ranges, l.amenities,
	l.weekly_hours, l.photo_file_ids, l.website_url, l.cancel_free_hours, l.cancel_policy, l.rules, l.rating_avg,
	l.rating_count`

// dirScan is the scan target of dirColumns; apply() decodes the JSON lists (a malformed column reads as empty).
type dirScan struct {
	ages, amenities, hours, photos sql.NullString
}

func (d *Dir) targets(s *dirScan) []any {
	return []any{&d.Status, &d.Verified, &d.Lat, &d.Lng, &d.About, &s.ages, &s.amenities, &s.hours, &s.photos,
		&d.WebsiteURL, &d.CancelFreeHours, &d.CancelPolicy, &d.Rules, &d.RatingAvg, &d.RatingCount}
}

func (d *Dir) apply(s dirScan) {
	d.AgeRanges, d.Amenities, d.WeeklyHours, d.PhotoFileIDs = nil, nil, nil, nil
	decode := func(n sql.NullString, v any) {
		if n.Valid && n.String != "" {
			_ = json.Unmarshal([]byte(n.String), v)
		}
	}
	decode(s.ages, &d.AgeRanges)
	decode(s.amenities, &d.Amenities)
	decode(s.hours, &d.WeeklyHours)
	decode(s.photos, &d.PhotoFileIDs)
}

// ---------------------------------------------------------------------------
// Search

// BBox is a map viewport (south-west / north-east corners).
type BBox struct{ MinLat, MinLng, MaxLat, MaxLng float64 }

// Point is a user position.
type Point struct{ Lat, Lng float64 }

// Search is the full public list query; the zero value is bloom's plain list.
type Search struct {
	ListFilter
	Q         string
	AgeMonths int // -1 = not given
	OpenNow   bool
	Amenities []string
	At        *Point
	RadiusKm  float64 // 0 = no radius (needs At)
	BBox      *BBox
	Sort      string
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(s)
}

// SearchCandidates returns up to MaxCandidates public listings matching the SQL-side filters (city, category, q,
// amenities, bbox, the radius' bounding box). Age fit, open-now, the exact radius and the ranking run in Go.
func (s *Store) SearchCandidates(ctx context.Context, q Search) ([]Listing, error) {
	where := publicWhere
	var args []any
	if q.CityID > 0 {
		where += " AND l.city_id = ?"
		args = append(args, q.CityID)
	}
	if q.Category != "" {
		where += " AND c.slug = ?"
		args = append(args, q.Category)
	}
	if q.Q != "" {
		like := "%" + escapeLike(q.Q) + "%"
		where += " AND (l.name LIKE ? OR l.district LIKE ? OR l.description LIKE ? OR l.about LIKE ?)"
		args = append(args, like, like, like, like)
	}
	for _, a := range q.Amenities {
		where += " AND JSON_CONTAINS(COALESCE(l.amenities, '[]'), JSON_QUOTE(?))"
		args = append(args, a)
	}
	if q.BBox != nil {
		where += " AND l.lat BETWEEN ? AND ? AND l.lng BETWEEN ? AND ?"
		args = append(args, q.BBox.MinLat, q.BBox.MaxLat, q.BBox.MinLng, q.BBox.MaxLng)
	}
	if q.At != nil && q.RadiusKm > 0 {
		b := around(*q.At, q.RadiusKm)
		where += " AND l.lat BETWEEN ? AND ? AND l.lng BETWEEN ? AND ?"
		args = append(args, b.MinLat, b.MaxLat, b.MinLng, b.MaxLng)
	}
	rows, err := s.db.QueryContext(ctx, listingSelect+where+" ORDER BY l.sort_order, l.id LIMIT ?", append(args, MaxCandidates)...)
	if err != nil {
		return nil, fmt.Errorf("cityservices: search: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []Listing{}
	for rows.Next() {
		var l Listing
		if err := scanListing(rows, &l); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// around is the bounding box of a circle (a cheap index prefilter; the exact distance is checked in Go).
func around(p Point, km float64) BBox {
	dLat := km / 111.32
	cos := math.Cos(p.Lat * math.Pi / 180)
	dLng := 180.0
	if cos > 0.01 {
		dLng = km / (111.32 * cos)
	}
	return BBox{MinLat: p.Lat - dLat, MaxLat: p.Lat + dLat, MinLng: p.Lng - dLng, MaxLng: p.Lng + dLng}
}

// DistanceKm is the haversine distance.
func DistanceKm(a, b Point) float64 {
	const r = 6371.0
	rad := math.Pi / 180
	dLat, dLng := (b.Lat-a.Lat)*rad, (b.Lng-a.Lng)*rad
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(a.Lat*rad)*math.Cos(b.Lat*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * r * math.Asin(math.Min(1, math.Sqrt(h)))
}

// ServiceAgeFit reports, per listing that has active services with an age bound, whether one of them fits ageMonths.
func (s *Store) ServiceAgeFit(ctx context.Context, ids []uint64, ageMonths int) (map[uint64]bool, error) {
	out := map[uint64]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	in, args := inList(ids)
	//nolint:gosec // G202: inList only adds "?" placeholders
	rows, err := s.db.QueryContext(ctx, `SELECT listing_id,
		MAX(COALESCE(age_min_months, 0) <= ? AND COALESCE(age_max_months, 65535) >= ?)
		FROM city_service_services WHERE is_active = 1 AND (age_min_months IS NOT NULL OR age_max_months IS NOT NULL)
		AND listing_id IN (`+in+`) GROUP BY listing_id`, append([]any{ageMonths, ageMonths}, args...)...)
	if err != nil {
		return nil, fmt.Errorf("cityservices: age fit: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id uint64
		var fit bool
		if err := rows.Scan(&id, &fit); err != nil {
			return nil, err
		}
		out[id] = fit
	}
	return out, rows.Err()
}

// CardExtras are the per-card aggregates: the lowest active service price and the next slot with a free seat.
type CardExtras struct {
	PriceFrom map[uint64]int64
	NextSlot  map[uint64]time.Time
}

// Extras loads CardExtras for ids (slots after now).
func (s *Store) Extras(ctx context.Context, ids []uint64, now time.Time) (CardExtras, error) {
	ex := CardExtras{PriceFrom: map[uint64]int64{}, NextSlot: map[uint64]time.Time{}}
	if len(ids) == 0 {
		return ex, nil
	}
	in, args := inList(ids)
	//nolint:gosec // G202: inList only adds "?" placeholders
	rows, err := s.db.QueryContext(ctx, `SELECT listing_id, MIN(price_toman) FROM city_service_services
		WHERE is_active = 1 AND price_toman IS NOT NULL AND listing_id IN (`+in+`) GROUP BY listing_id`, args...)
	if err != nil {
		return ex, fmt.Errorf("cityservices: price from: %w", err)
	}
	for rows.Next() {
		var id uint64
		var p int64
		if err := rows.Scan(&id, &p); err != nil {
			_ = rows.Close()
			return ex, err
		}
		ex.PriceFrom[id] = p
	}
	_ = rows.Close()
	//nolint:gosec // G202: inList only adds "?" placeholders
	rows, err = s.db.QueryContext(ctx, `SELECT s.listing_id, MIN(s.starts_at) FROM city_service_slots s
		LEFT JOIN city_service_services v ON v.id = s.service_id
		WHERE s.status = 'open' AND s.booked_count < s.capacity AND s.starts_at > ? AND (s.service_id IS NULL OR v.is_active = 1)
		AND s.listing_id IN (`+in+`) GROUP BY s.listing_id`, append([]any{now}, args...)...)
	if err != nil {
		return ex, fmt.Errorf("cityservices: next slot: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id uint64
		var t time.Time
		if err := rows.Scan(&id, &t); err != nil {
			return ex, err
		}
		ex.NextSlot[id] = t
	}
	return ex, rows.Err()
}

// PhotoPaths maps public place_photo file ids to their stored path (the URL is /api/v1/files/public/{id}/{name}).
func (s *Store) PhotoPaths(ctx context.Context, ids []uint64) (map[uint64]string, error) {
	out := map[uint64]string{}
	if len(ids) == 0 {
		return out, nil
	}
	in, args := inList(ids)
	//nolint:gosec // G202: inList only adds "?" placeholders
	rows, err := s.db.QueryContext(ctx, `SELECT id, path FROM files WHERE purpose = 'place_photo' AND visibility = 'public'
		AND id IN (`+in+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("cityservices: photos: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id uint64
		var p string
		if err := rows.Scan(&id, &p); err != nil {
			return nil, err
		}
		out[id] = p
	}
	return out, rows.Err()
}

func inList(ids []uint64) (string, []any) {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return strings.TrimSuffix(strings.Repeat("?,", len(ids)), ","), args
}

// ---------------------------------------------------------------------------
// Place detail

// Service is one active service / package of a place.
type Service struct {
	ID                  uint64
	Kind                string
	Name                string
	Description         sql.NullString
	Format              sql.NullString
	DurationMinutes     sql.NullInt64
	AgeMinMonths        sql.NullInt64
	AgeMaxMonths        sql.NullInt64
	Capacity            sql.NullInt64
	PriceToman          sql.NullInt64
	PackageSessions     sql.NullInt64
	PackageValidityDays sql.NullInt64
}

// Slot is one open time of a place.
type Slot struct {
	ID          uint64
	ServiceID   sql.NullInt64
	StartsAt    time.Time
	EndsAt      sql.NullTime
	Capacity    int
	BookedCount int
}

// Review is one published review (the author as a first name only).
type Review struct {
	ID        uint64
	Rating    int
	Body      sql.NullString
	Author    sql.NullString
	CreatedAt sql.NullTime
}

// ReviewSummary is the place's «نظر مادرها» block.
type ReviewSummary struct {
	Count                              int
	Avg                                sql.NullFloat64
	Cleanliness, Instructor, Amenities sql.NullFloat64
	Latest                             []Review
}

// Services lists a place's active services in display order.
func (s *Store) Services(ctx context.Context, listingID uint64) ([]Service, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, kind, name, description, format, duration_minutes, age_min_months,
		age_max_months, capacity, price_toman, package_sessions, package_validity_days FROM city_service_services
		WHERE listing_id = ? AND is_active = 1 ORDER BY sort_order, id`, listingID)
	if err != nil {
		return nil, fmt.Errorf("cityservices: services: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []Service{}
	for rows.Next() {
		var v Service
		if err := rows.Scan(&v.ID, &v.Kind, &v.Name, &v.Description, &v.Format, &v.DurationMinutes, &v.AgeMinMonths,
			&v.AgeMaxMonths, &v.Capacity, &v.PriceToman, &v.PackageSessions, &v.PackageValidityDays); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ActiveService reports whether serviceID is an active service of listingID.
func (s *Store) ActiveService(ctx context.Context, listingID, serviceID uint64) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM city_service_services WHERE id = ? AND listing_id = ? AND is_active = 1`,
		serviceID, listingID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// Slots lists a place's open slots in [from, to) after now, earliest first; serviceID 0 = every service (slots of an
// inactive service are skipped), else that service's slots plus the place-wide ones (service_id NULL).
func (s *Store) Slots(ctx context.Context, listingID, serviceID uint64, from, to, now time.Time, limit int) ([]Slot, error) {
	q := `SELECT s.id, s.service_id, s.starts_at, s.ends_at, s.capacity, s.booked_count FROM city_service_slots s
		LEFT JOIN city_service_services v ON v.id = s.service_id
		WHERE s.listing_id = ? AND s.status = 'open' AND s.starts_at > ? AND s.starts_at >= ? AND s.starts_at < ?
		AND (s.service_id IS NULL OR v.is_active = 1)`
	args := []any{listingID, now, from, to}
	if serviceID > 0 {
		q += " AND (s.service_id IS NULL OR s.service_id = ?)"
		args = append(args, serviceID)
	}
	rows, err := s.db.QueryContext(ctx, q+" ORDER BY s.starts_at, s.id LIMIT ?", append(args, limit)...)
	if err != nil {
		return nil, fmt.Errorf("cityservices: slots: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []Slot{}
	for rows.Next() {
		var v Slot
		if err := rows.Scan(&v.ID, &v.ServiceID, &v.StartsAt, &v.EndsAt, &v.Capacity, &v.BookedCount); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// Reviews is the summary of a place's published reviews plus the latest few.
func (s *Store) Reviews(ctx context.Context, listingID uint64) (ReviewSummary, error) {
	var r ReviewSummary
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), AVG(rating), AVG(score_cleanliness), AVG(score_instructor),
		AVG(score_amenities) FROM city_service_reviews WHERE listing_id = ? AND status = 'published'`, listingID).
		Scan(&r.Count, &r.Avg, &r.Cleanliness, &r.Instructor, &r.Amenities); err != nil {
		return r, fmt.Errorf("cityservices: review summary: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT r.id, r.rating, r.body, u.name, r.created_at FROM city_service_reviews r
		JOIN users u ON u.id = r.user_id WHERE r.listing_id = ? AND r.status = 'published'
		ORDER BY r.created_at DESC, r.id DESC LIMIT ?`, listingID, ReviewsShown)
	if err != nil {
		return r, fmt.Errorf("cityservices: reviews: %w", err)
	}
	defer func() { _ = rows.Close() }()
	r.Latest = []Review{}
	for rows.Next() {
		var v Review
		if err := rows.Scan(&v.ID, &v.Rating, &v.Body, &v.Author, &v.CreatedAt); err != nil {
			return r, err
		}
		v.Author = firstName(v.Author)
		r.Latest = append(r.Latest, v)
	}
	return r, rows.Err()
}

// firstName keeps the first word of a display name (nothing else about an author is ever shown).
func firstName(n sql.NullString) sql.NullString {
	if f := strings.Fields(n.String); n.Valid && len(f) > 0 {
		return sql.NullString{String: f[0], Valid: true}
	}
	return sql.NullString{}
}

// ---------------------------------------------------------------------------
// Pure helpers (unit-tested)

// AgeRange is a dir_age_ranges catalog item's month bounds.
type AgeRange struct {
	Min, Max  int
	Pregnancy bool
}

// AgeRangesFrom reads the month bounds out of the catalog items' meta.
func AgeRangesFrom(items []catalog.Item) map[string]AgeRange {
	out := map[string]AgeRange{}
	for _, it := range items {
		var m struct {
			Min       *int `json:"min_months"`
			Max       *int `json:"max_months"`
			Pregnancy bool `json:"pregnancy"`
		}
		if json.Unmarshal(it.Meta, &m) != nil {
			continue
		}
		r := AgeRange{Pregnancy: m.Pregnancy, Min: 0, Max: math.MaxInt32}
		if m.Min != nil {
			r.Min = *m.Min
		}
		if m.Max != nil {
			r.Max = *m.Max
		}
		out[it.Code] = r
	}
	return out
}

// AgeFit reports whether a place's age data is known and whether it fits ageMonths: one of its age-range codes covers
// the age, or (serviceKnown) one of its age-bounded services does.
func AgeFit(codes []string, ranges map[string]AgeRange, ageMonths int, serviceFit, serviceKnown bool) (known, fit bool) {
	for _, code := range codes {
		r, ok := ranges[code]
		if !ok {
			continue
		}
		known = true // a pregnancy-only place is known not to fit a child's age
		if r.Pregnancy {
			continue
		}
		if ageMonths >= r.Min && ageMonths <= r.Max {
			fit = true
		}
	}
	if serviceKnown {
		known = true
		fit = fit || serviceFit
	}
	return known, fit
}

// JalaliWeekday is t's weekday with Saturday = 0 (weekly_hours' day).
func JalaliWeekday(t time.Time) int { return (int(t.Weekday()) + 1) % 7 }

func hhmm(s string) (int, bool) {
	h, m, ok := strings.Cut(s, ":")
	if !ok {
		return 0, false
	}
	hi, err1 := strconv.Atoi(h)
	mi, err2 := strconv.Atoi(m)
	if err1 != nil || err2 != nil || hi < 0 || hi > 24 || mi < 0 || mi > 59 {
		return 0, false
	}
	return hi*60 + mi, true
}

// OpenAt reports whether weekly hours are known and whether the place is open at t (Tehran wall-clock). A window
// whose close is before its open runs past midnight.
func OpenAt(hours []DayHours, t time.Time) (known, open bool) {
	if len(hours) == 0 {
		return false, false
	}
	day, now := JalaliWeekday(t), t.Hour()*60+t.Minute()
	prev := (day + 6) % 7
	for _, w := range hours {
		o, ok1 := hhmm(w.Open)
		c, ok2 := hhmm(w.Close)
		if !ok1 || !ok2 {
			continue
		}
		known = true
		switch {
		case w.Day == day && o <= c && now >= o && now < c:
			return true, true
		case w.Day == day && o > c && now >= o:
			return true, true
		case w.Day == prev && o > c && now < c:
			return true, true
		}
	}
	return known, false
}

// Ranked is a search hit with its computed signals.
type Ranked struct {
	Listing
	Distance          *float64 // km, nil without a position or coordinates
	AgeKnown, AgeFits bool
	OpenKnown, IsOpen bool
	Score             float64
}

// ratingScore is the Bayesian-smoothed rating on 0..1 (an unrated place sits at the prior).
func ratingScore(avg sql.NullFloat64, count int) float64 {
	sum := ratingPrior * ratingPriorN
	if avg.Valid && count > 0 {
		sum += avg.Float64 * float64(count)
	}
	return sum / float64(ratingPriorN+max(count, 0)) / 5
}

// Score combines the signals that apply to the request: distance (a position was given), age fit (an age was given)
// and the rating (always). Weights 0.5 / 0.25 / 0.25 renormalised over the signals in play. No paid signal exists.
func Score(r Ranked, withDistance, withAge bool) float64 {
	sum, weight := 0.25*ratingScore(r.RatingAvg, r.RatingCount), 0.25
	if withDistance {
		weight += 0.5
		if r.Distance != nil {
			sum += 0.5 / (1 + *r.Distance/2)
		}
	}
	if withAge {
		weight += 0.25
		switch {
		case r.AgeKnown && r.AgeFits:
			sum += 0.25
		case !r.AgeKnown:
			sum += 0.125
		}
	}
	return sum / weight
}

// SortRanked orders hits by sort; ties fall back to bloom's sort_order, then id.
func SortRanked(hits []Ranked, by string) {
	tie := func(a, b Ranked) bool {
		if a.SortOrder != b.SortOrder {
			return a.SortOrder < b.SortOrder
		}
		return a.ID < b.ID
	}
	sort.SliceStable(hits, func(i, j int) bool {
		a, b := hits[i], hits[j]
		switch by {
		case SortDistance:
			if (a.Distance == nil) != (b.Distance == nil) {
				return a.Distance != nil
			}
			if a.Distance != nil && *a.Distance != *b.Distance {
				return *a.Distance < *b.Distance
			}
		case SortRating:
			sa, sb := ratingScore(a.RatingAvg, a.RatingCount), ratingScore(b.RatingAvg, b.RatingCount)
			if sa != sb {
				return sa > sb
			}
		default:
			if a.Score != b.Score {
				return a.Score > b.Score
			}
		}
		return tie(a, b)
	})
}

// Rank filters candidates by the Go-side filters (radius, age, open-now), computes the signals and sorts them.
func Rank(cands []Listing, q Search, ranges map[string]AgeRange, serviceFit map[uint64]bool, now time.Time) []Ranked {
	withAge := q.AgeMonths >= 0
	out := make([]Ranked, 0, len(cands))
	for _, l := range cands {
		r := Ranked{Listing: l}
		if q.At != nil && l.Lat.Valid && l.Lng.Valid {
			d := DistanceKm(*q.At, Point{Lat: l.Lat.Float64, Lng: l.Lng.Float64})
			r.Distance = &d
		}
		if q.At != nil && q.RadiusKm > 0 && (r.Distance == nil || *r.Distance > q.RadiusKm) {
			continue
		}
		if withAge {
			sf, sk := serviceFit[l.ID]
			r.AgeKnown, r.AgeFits = AgeFit(l.AgeRanges, ranges, q.AgeMonths, sf, sk)
			if r.AgeKnown && !r.AgeFits {
				continue
			}
		}
		r.OpenKnown, r.IsOpen = OpenAt(l.WeeklyHours, now)
		if q.OpenNow && !r.IsOpen {
			continue
		}
		r.Score = Score(r, q.At != nil, withAge)
		out = append(out, r)
	}
	by := q.Sort
	if by == SortDistance && q.At == nil {
		by = SortRank
	}
	SortRanked(out, by)
	return out
}
