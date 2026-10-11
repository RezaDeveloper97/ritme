package join

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cityservices/join/store"
)

func TestTrackingCode_FormatAndRandomness(t *testing.T) {
	re := regexp.MustCompile(`^[2-9A-HJ-NP-Z]{4}-[2-9A-HJ-NP-Z]{4}-[2-9A-HJ-NP-Z]{4}$`)
	seen := map[string]bool{}
	for range 500 {
		c, err := NewTrackingCode()
		require.NoError(t, err)
		assert.Regexp(t, re, c)
		assert.LessOrEqual(t, len(c), 16, "fits varchar(16)")
		assert.False(t, seen[c], "60 random bits never repeat in 500 draws")
		seen[c] = true
	}
}

func complete() Draft {
	return Draft{
		Step1: Step1{BusinessName: "Ab Pari", CategorySlugs: []string{"class"}, CityID: 1, ContactName: "Sara",
			ContactPhone: "09120000000", OwnerConfirmed: true},
		Step2: Step2{Lat: "35.700000", Lng: "51.400000", Address: "Vanak", AgeRanges: []string{"m6_12"},
			WeeklyHours: []Hours{{Day: 0, Open: "09:00", Close: "20:00"}}},
		Step3: Step3{Services: []Offer{{Kind: "session", Name: "Water"}}},
		Step4: Step4{BookingMode: "request", TermsAccepted: true},
	}
}

func TestMissing(t *testing.T) {
	docs := map[string]bool{SlotLicence: true}
	assert.Empty(t, Missing(complete(), MinPhotos, docs))

	m := Missing(Draft{}, 0, nil)
	for _, f := range []string{"business_name", "category_slugs", "city_id", "contact_name", "contact_phone",
		"owner_confirmed", "location", "address", "age_ranges", "weekly_hours", "photos", "services", "booking_mode",
		"terms", "documents.licence"} {
		assert.Contains(t, m, f)
	}
	assert.NotContains(t, m, "documents.health_cert", "only pools need the health certificate")
	for f := range m {
		assert.Contains(t, missingOrder, f, "every missing field has a place in the response order")
	}

	d := complete()
	assert.Equal(t, map[string]string{"photos": "min_photos"}, Missing(d, MinPhotos-1, docs))
	d.Lng = ""
	assert.Equal(t, "location", Missing(d, MinPhotos, docs)["location"])

	pool := complete()
	pool.CategorySlugs = []string{"class", HealthCertCategory}
	assert.Equal(t, map[string]string{"documents.health_cert": "document_required"}, Missing(pool, MinPhotos, docs))
	assert.Empty(t, Missing(pool, MinPhotos, map[string]bool{SlotLicence: true, SlotHealthCert: true}))
}

func TestStatusRules(t *testing.T) {
	assert.True(t, Editable(StatusDraft))
	assert.True(t, Editable(StatusNeedsInfo))
	for _, s := range []string{StatusSubmitted, StatusInReview, StatusApproved, StatusRejected} {
		assert.False(t, Editable(s), s)
	}
	assert.True(t, PhotosOpen(StatusSubmitted))
	assert.True(t, PhotosOpen(StatusInReview))
	assert.False(t, PhotosOpen(StatusApproved))
	assert.False(t, PhotosOpen(StatusRejected))
}

func TestDecodeRow_MalformedJSONReadsEmpty(t *testing.T) {
	a := decodeRow(store.CityServiceApplication{ID: 1, Status: StatusDraft,
		CategorySlugs: sql.NullString{String: "not json", Valid: true},
		Documents:     sql.NullString{String: `{"licence": 7}`, Valid: true},
		PhotoFileIds:  sql.NullString{String: `[3,4]`, Valid: true},
		CityID:        sql.NullInt64{Int64: 2, Valid: true},
	})
	assert.Empty(t, a.Draft.CategorySlugs)
	assert.Equal(t, uint64(7), a.Draft.Documents[SlotLicence])
	assert.Equal(t, []uint64{3, 4}, a.Draft.PhotoIDs)
	assert.Equal(t, uint64(2), a.Draft.CityID)
}

func TestTimeline(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 40, 0, 0, time.UTC)
	states := func(status string) []string {
		var out []string
		for _, o := range timeline(Application{Row: store.CityServiceApplication{Status: status,
			SubmittedAt: sql.NullTime{Time: now, Valid: true}, ReviewedAt: sql.NullTime{Time: now, Valid: true}}}) {
			v, _ := o.Get("state")
			out = append(out, v.(string))
		}
		return out
	}
	assert.Equal(t, []string{"done", "current", "pending", "pending"}, states(StatusSubmitted))
	assert.Equal(t, []string{"done", "done", "done", "done"}, states(StatusApproved))
	assert.Equal(t, []string{"done", "done", "done", "failed"}, states(StatusRejected))
	assert.Equal(t, []string{"pending", "pending", "pending", "pending"}, states("unknown"))
}

func TestTranslations_SameKeysInEveryLanguage(t *testing.T) {
	for _, key := range []string{"messages.submitted", "messages.locked", "validation.min_photos", "validation.phone",
		"attributes.business_name"} {
		en, fa := T(key, "en", map[string]string{"code": "X", "min": "4"}), T(key, "fa", map[string]string{"code": "X", "min": "4"})
		assert.NotEqual(t, "join."+key, en, key)
		assert.NotEqual(t, "join."+key, fa, key)
		assert.NotEqual(t, en, fa, key)
	}
	assert.Contains(t, T("messages.submitted", "en", map[string]string{"code": "AB12-CD34-EF56"}), "AB12-CD34-EF56")
	assert.NotEmpty(t, attributes("fa"))
}

func TestMobile(t *testing.T) {
	assert.True(t, mobile("09121234567"))
	for _, s := range []string{"9121234567", "0912123456", "08121234567", "0912123456a", "+989121234567"} {
		assert.False(t, mobile(s), s)
	}
}
