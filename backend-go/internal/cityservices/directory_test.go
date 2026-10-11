package cityservices

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func tehran(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, civildate.Tehran)
	if err != nil {
		panic(err)
	}
	return t
}

func TestJalaliWeekday(t *testing.T) {
	assert.Equal(t, 0, JalaliWeekday(tehran("2026-09-26 10:00")), "Saturday")
	assert.Equal(t, 4, JalaliWeekday(tehran("2026-09-23 10:00")), "Wednesday")
	assert.Equal(t, 6, JalaliWeekday(tehran("2026-09-25 10:00")), "Friday")
}

func TestOpenAt(t *testing.T) {
	wed := []DayHours{{Day: 4, Open: "09:00", Close: "20:00"}}
	known, open := OpenAt(wed, tehran("2026-09-23 10:00"))
	assert.True(t, known)
	assert.True(t, open)
	_, open = OpenAt(wed, tehran("2026-09-23 20:00"))
	assert.False(t, open, "close is exclusive")
	_, open = OpenAt(wed, tehran("2026-09-24 10:00"))
	assert.False(t, open, "Thursday is missing = closed")
	known, open = OpenAt(nil, tehran("2026-09-23 10:00"))
	assert.False(t, known)
	assert.False(t, open)
	night := []DayHours{{Day: 4, Open: "20:00", Close: "02:00"}}
	_, open = OpenAt(night, tehran("2026-09-23 23:00"))
	assert.True(t, open)
	_, open = OpenAt(night, tehran("2026-09-24 01:00"))
	assert.True(t, open, "past midnight belongs to the previous day's window")
	known, _ = OpenAt([]DayHours{{Day: 4, Open: "x", Close: "y"}}, tehran("2026-09-23 10:00"))
	assert.False(t, known, "malformed windows are ignored")
}

func TestAgeRangesAndFit(t *testing.T) {
	ranges := AgeRangesFrom([]catalog.Item{
		{Code: "pregnancy", Meta: json.RawMessage(`{"pregnancy":true}`)},
		{Code: "m6_12", Meta: json.RawMessage(`{"min_months":6,"max_months":12}`)},
		{Code: "y1_2", Meta: json.RawMessage(`{"min_months":12,"max_months":24}`)},
	})
	known, fit := AgeFit([]string{"m6_12", "y1_2"}, ranges, 14, false, false)
	assert.True(t, known)
	assert.True(t, fit)
	known, fit = AgeFit([]string{"m6_12"}, ranges, 30, false, false)
	assert.True(t, known)
	assert.False(t, fit)
	known, fit = AgeFit([]string{"pregnancy"}, ranges, 3, false, false)
	assert.True(t, known, "a pregnancy-only place is known")
	assert.False(t, fit)
	known, _ = AgeFit(nil, ranges, 3, false, false)
	assert.False(t, known, "no age data")
	known, fit = AgeFit([]string{"m6_12"}, ranges, 30, true, true)
	assert.True(t, known)
	assert.True(t, fit, "a fitting service counts")
	known, _ = AgeFit([]string{"unknown_code"}, ranges, 3, false, false)
	assert.False(t, known)
}

func TestParseBBox(t *testing.T) {
	b, ok := ParseBBox("51.40,35.75,51.42,35.78")
	require.True(t, ok)
	assert.Equal(t, BBox{MinLng: 51.40, MinLat: 35.75, MaxLng: 51.42, MaxLat: 35.78}, *b)
	for _, bad := range []string{"", "1,2,3", "51.42,35.75,51.40,35.78", "a,b,c,d", "0,95,1,96", "200,0,201,1"} {
		_, ok := ParseBBox(bad)
		assert.False(t, ok, bad)
	}
}

func TestDistanceAndAround(t *testing.T) {
	vanak, jordan := Point{Lat: 35.7576, Lng: 51.4099}, Point{Lat: 35.772, Lng: 51.418}
	d := DistanceKm(vanak, jordan)
	assert.InDelta(t, 1.8, d, 0.1)
	assert.InDelta(t, 0, DistanceKm(vanak, vanak), 1e-9)
	b := around(vanak, 5)
	assert.Less(t, b.MinLat, vanak.Lat)
	assert.Greater(t, b.MaxLng, vanak.Lng)
	assert.InDelta(t, 5/111.32, b.MaxLat-vanak.Lat, 1e-9)
}

func listing(id uint64, lat, lng float64, rating float64, count int) Listing {
	l := Listing{ID: id}
	if lat != 0 {
		l.Lat, l.Lng = sql.NullFloat64{Float64: lat, Valid: true}, sql.NullFloat64{Float64: lng, Valid: true}
	}
	if count > 0 {
		l.RatingAvg, l.RatingCount = sql.NullFloat64{Float64: rating, Valid: true}, count
	}
	return l
}

func TestRank(t *testing.T) {
	now := tehran("2026-09-23 10:00")
	near := listing(1, 35.7576, 51.4099, 4.2, 10)
	far := listing(2, 35.9, 51.6, 5.0, 200)
	unrated := listing(3, 0, 0, 0, 0)
	unrated.SortOrder = -1
	at := &Point{Lat: 35.7576, Lng: 51.4099}

	// No position, no age: rating only; an unrated place sits at the prior.
	hits := Rank([]Listing{near, far, unrated}, Search{AgeMonths: -1}, nil, nil, now)
	require.Len(t, hits, 3)
	assert.Equal(t, []uint64{2, near.ID, 3}, ids(hits))

	// With a position the near place wins over the better-rated far one.
	hits = Rank([]Listing{far, near}, Search{AgeMonths: -1, At: at}, nil, nil, now)
	assert.Equal(t, []uint64{1, 2}, ids(hits))
	require.NotNil(t, hits[0].Distance)

	// Radius drops places out of range or without a pin.
	hits = Rank([]Listing{far, near, unrated}, Search{AgeMonths: -1, At: at, RadiusKm: 5}, nil, nil, now)
	assert.Equal(t, []uint64{1}, ids(hits))

	// Distance sort puts pinless places last; distance without a position falls back to rank.
	hits = Rank([]Listing{unrated, far, near}, Search{AgeMonths: -1, At: at, Sort: SortDistance}, nil, nil, now)
	assert.Equal(t, []uint64{1, 2, 3}, ids(hits))
	hits = Rank([]Listing{near, far}, Search{AgeMonths: -1, Sort: SortDistance}, nil, nil, now)
	assert.Equal(t, []uint64{2, 1}, ids(hits))

	// Age: a known misfit is dropped, an unknown one is kept but ranks below a fit.
	ranges := map[string]AgeRange{"y1_2": {Min: 12, Max: 24}, "m6_12": {Min: 6, Max: 12}}
	fit, misfit, unknown := listing(10, 0, 0, 0, 0), listing(11, 0, 0, 0, 0), listing(12, 0, 0, 0, 0)
	fit.AgeRanges, misfit.AgeRanges = []string{"y1_2"}, []string{"m6_12"}
	unknown.SortOrder = -5
	hits = Rank([]Listing{unknown, misfit, fit}, Search{AgeMonths: 14}, ranges, nil, now)
	assert.Equal(t, []uint64{10, 12}, ids(hits))
	assert.True(t, hits[0].AgeFits)

	// Open now keeps only places known to be open.
	open, closed := listing(20, 0, 0, 0, 0), listing(21, 0, 0, 0, 0)
	open.WeeklyHours = []DayHours{{Day: 4, Open: "09:00", Close: "20:00"}}
	closed.WeeklyHours = []DayHours{{Day: 5, Open: "09:00", Close: "20:00"}}
	hits = Rank([]Listing{closed, open, unknown}, Search{AgeMonths: -1, OpenNow: true}, nil, nil, now)
	assert.Equal(t, []uint64{20}, ids(hits))

	// Rating sort.
	hits = Rank([]Listing{near, far}, Search{AgeMonths: -1, At: at, Sort: SortRating}, nil, nil, now)
	assert.Equal(t, []uint64{2, 1}, ids(hits))
}

func ids(hits []Ranked) []uint64 {
	out := make([]uint64, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.ID)
	}
	return out
}

func TestFirstNameAndPhotos(t *testing.T) {
	assert.Equal(t, sql.NullString{String: "Raha", Valid: true}, firstName(sql.NullString{String: " Raha Ahmadi ", Valid: true}))
	assert.False(t, firstName(sql.NullString{String: "  ", Valid: true}).Valid)
	assert.False(t, firstName(sql.NullString{}).Valid)

	urls := FilePhotoURLs("https://api.example/", []uint64{3, 9, 4}, map[uint64]string{4: "place_photo/1/b.webp.bin", 3: "place_photo/1/a.webp.bin"})
	assert.Equal(t, []string{"https://api.example/api/v1/files/public/3/a.webp", "https://api.example/api/v1/files/public/4/b.webp"}, urls)
}

func TestSlotJSONAndSafeURL(t *testing.T) {
	s := Slot{ID: 1, StartsAt: tehran("2026-09-23 17:00"), Capacity: 6, BookedCount: 7}
	raw, err := json.Marshal(SlotJSON(s))
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":1,"service_id":null,"starts_at":"2026-09-23T17:00:00+03:30","ends_at":null,"capacity":6,"seats_left":0,"is_full":true}`, string(raw))
	assert.Nil(t, safeURL(sql.NullString{String: "javascript:alert(1)", Valid: true}))
	assert.Equal(t, "https://x.example", safeURL(sql.NullString{String: "https://x.example", Valid: true}))
}

func TestScoreHasNoPaidSignal(t *testing.T) {
	// Score depends only on distance, age fit and rating: two places identical in those score the same whatever
	// their admin sort order.
	a, b := Ranked{Listing: listing(1, 0, 0, 4.5, 10)}, Ranked{Listing: listing(2, 0, 0, 4.5, 10)}
	b.SortOrder = -100
	assert.InDelta(t, Score(a, true, true), Score(b, true, true), 1e-12)
}
