package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// pinnedNow is the instant these tests run at: 23:59 Tehran on the last day of a Gregorian month and of the Gregorian
// year, where a clock read from the wall used to flip day, month and week windows.
var pinnedNow = time.Date(2026, 12, 31, 23, 59, 0, 0, time.FixedZone("IRST", 3*3600+1800))

type fakeCatalog map[string][]catalog.Item

func (f fakeCatalog) Items(_ context.Context, group string) ([]catalog.Item, error) {
	return f[group], nil
}

type fakeDocs struct {
	n     int64
	calls int
}

func (f *fakeDocs) CountRecordDocuments(_ context.Context, _ uint64) (int64, error) {
	f.calls++
	return f.n, nil
}

type fakeBookings struct{ b *Booking }

func (f fakeBookings) NextBooking(context.Context, uint64, time.Time) (*Booking, error) {
	return f.b, nil
}

func modeOf(m enums.LifeMode) ModeReader {
	return func(context.Context, uint64) (enums.LifeMode, error) { return m, nil }
}

func item(code, title, metaJSON string, audiences ...string) catalog.Item {
	it := catalog.Item{Code: code, Title: json.RawMessage(`{"fa":"` + title + `","en":"` + title + `"}`), Audiences: audiences}
	if metaJSON != "" {
		it.Meta = json.RawMessage(metaJSON)
	}
	return it
}

var loc = catalog.Localizer{Locale: "en", Default: "fa"}

func hub(t *testing.T, svc *Service) map[string]any {
	t.Helper()
	o, err := svc.Hub(context.Background(), 7, time.Date(2026, 10, 7, 10, 0, 0, 0, time.UTC), loc)
	require.NoError(t, err)
	raw, err := jsonx.Marshal(o, 0)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	return m
}

func codes(list any) []string {
	var out []string
	for _, x := range list.([]any) {
		out = append(out, x.(map[string]any)["code"].(string))
	}
	return out
}

func TestHub_EmptyCatalogFallsBackToBoardOrder(t *testing.T) {
	m := hub(t, NewService(fakeCatalog{}, &fakeDocs{}, modeOf(enums.LifeModeCycle), nil))
	assert.Equal(t, SectionCodes, codes(m["sections"]))
	assert.Nil(t, m["upcoming_booking"], "no telemedicine yet → null")
	assert.Empty(t, m["care"])
	assert.Empty(t, m["programs"])
	sos := m["sections"].([]any)[len(SectionCodes)-1].(map[string]any)
	assert.Equal(t, "115", sos["phone"])
	assert.Nil(t, sos["title"], "no catalog copy → clients use their bundled strings")
}

func TestHub_SectionsOrderToggleAndEmergencyAlwaysShown(t *testing.T) {
	cat := fakeCatalog{GroupSections: {
		item("care", "Care", ""),
		item("bogus", "?", ""),
		item("shop", "Shop", `{"categories":[{"code":"layette","icon":"bottle","title":{"fa":"س","en":"Layette"}},{"code":"Bad Code"}]}`),
		item("search", "Find", `{"href":"/search"}`),
		item("care", "dup", ""),
	}}
	m := hub(t, NewService(cat, &fakeDocs{}, modeOf(enums.LifeModeCycle), nil))
	assert.Equal(t, []string{"care", "shop", "search", "emergency"}, codes(m["sections"]))
	secs := m["sections"].([]any)
	shop := secs[1].(map[string]any)
	assert.Equal(t, "soon", shop["status"])
	assert.Equal(t, []any{map[string]any{"code": "layette", "title": "Layette", "icon": "bottle"}}, shop["categories"])
	search := secs[2].(map[string]any)
	assert.Equal(t, "/search", search["href"])
	assert.Equal(t, "live", search["status"])
}

func TestHub_TeenNeverSeesTheShop(t *testing.T) {
	cat := fakeCatalog{GroupSections: {item("shop", "Shop", ""), item("learning", "L", "")}}
	m := hub(t, NewService(cat, &fakeDocs{}, modeOf(enums.LifeModeTeen), nil))
	assert.Equal(t, []string{"learning", "emergency"}, codes(m["sections"]))
}

// CB-NAV-03 (IA_Nav «نوجوان»): doctor & lab tiles only with a linked parent; other modes never ask.
func TestHub_TeenDoctorAndLabsOnlyWithParent(t *testing.T) {
	cat := fakeCatalog{GroupCare: {item("doctors", "D", ""), item("record", "R", ""), item("labs", "L", ""), item("vitals", "V", "")}}
	calls := 0
	linked := func(v bool) ParentLinked {
		return func(context.Context, uint64) (bool, error) { calls++; return v, nil }
	}
	m := hub(t, NewService(cat, &fakeDocs{}, modeOf(enums.LifeModeTeen), nil).WithTeenParent(linked(false)))
	assert.Equal(t, []string{"record", "vitals"}, codes(m["care"]))
	assert.Equal(t, 1, calls, "looked up once per hub")

	m = hub(t, NewService(cat, &fakeDocs{}, modeOf(enums.LifeModeTeen), nil).WithTeenParent(linked(true)))
	assert.Equal(t, []string{"doctors", "record", "labs", "vitals"}, codes(m["care"]))

	m = hub(t, NewService(cat, &fakeDocs{}, modeOf(enums.LifeModeTeen), nil))
	assert.Equal(t, []string{"record", "vitals"}, codes(m["care"]), "no checker wired → hidden")

	calls = 0
	m = hub(t, NewService(cat, &fakeDocs{}, modeOf(enums.LifeModeCycle), nil).WithTeenParent(linked(false)))
	assert.Equal(t, []string{"doctors", "record", "labs", "vitals"}, codes(m["care"]))
	assert.Zero(t, calls)

	failing := func(context.Context, uint64) (bool, error) { return false, errors.New("db down") }
	_, err := NewService(cat, &fakeDocs{}, modeOf(enums.LifeModeTeen), nil).WithTeenParent(failing).
		Hub(context.Background(), 7, pinnedNow, loc)
	require.Error(t, err)
}

func TestHub_EmergencyPhoneValidated(t *testing.T) {
	for meta, want := range map[string]string{`{"phone":"112"}`: "112", `{"phone":"javascript:x"}`: "115", `not json`: "115"} {
		cat := fakeCatalog{GroupSections: {item("emergency", "SOS", meta)}}
		m := hub(t, NewService(cat, &fakeDocs{}, modeOf(enums.LifeModeCycle), nil))
		assert.Equal(t, want, m["sections"].([]any)[0].(map[string]any)["phone"], meta)
	}
}

func TestHub_CareTilesHrefCountAndAudience(t *testing.T) {
	docs := &fakeDocs{n: 41}
	cat := fakeCatalog{GroupCare: {
		item("record", "Record", `{"icon":"fileDoc","tone":"brand","href":"/record","counter":"record_documents"}`),
		item("doctors", "Doctors", `{"icon":"stetho","tone":"data"}`),
		item("evil", "Evil", `{"icon":"<svg>","tone":"neon","href":"https://evil.example"}`),
		item("proto", "Proto", `{"href":"//evil.example"}`),
		item("preg", "Preg only", `{"href":"/pregnancy"}`, "pregnancy"),
	}}
	m := hub(t, NewService(cat, docs, modeOf(enums.LifeModeCycle), nil))
	care := m["care"].([]any)
	assert.Equal(t, []string{"record", "doctors", "evil", "proto"}, codes(care))
	rec := care[0].(map[string]any)
	assert.Equal(t, float64(41), rec["count"])
	assert.Equal(t, "live", rec["status"])
	doc := care[1].(map[string]any)
	assert.Equal(t, "soon", doc["status"])
	assert.Nil(t, doc["href"])
	assert.Nil(t, doc["count"])
	evil := care[2].(map[string]any)
	assert.Nil(t, evil["href"], "external links are never passed through")
	assert.Equal(t, "grid", evil["icon"])
	assert.Equal(t, "neutral", evil["tone"])
	assert.Nil(t, care[3].(map[string]any)["href"], "protocol-relative URL rejected")
	assert.Equal(t, 1, docs.calls)
}

func TestHub_ProgramsFilteredByMode(t *testing.T) {
	cat := fakeCatalog{GroupPrograms: {
		item("pmdd", "PMDD", `{"icon":"smile","tone":"brand"}`),
		item("contraception", "Contra", `{"icon":"pill","href":"/contraception"}`, "cycle", "postpartum"),
	}}
	assert.Equal(t, []string{"pmdd", "contraception"}, codes(hub(t, NewService(cat, &fakeDocs{}, modeOf(enums.LifeModeCycle), nil))["programs"]))
	assert.Equal(t, []string{"pmdd"}, codes(hub(t, NewService(cat, &fakeDocs{}, modeOf(enums.LifeModePregnancy), nil))["programs"]))
}

func TestHub_UpcomingBooking(t *testing.T) {
	at := time.Date(2026, 10, 15, 18, 0, 0, 0, time.FixedZone("IRST", 12600))
	svc := NewService(fakeCatalog{}, &fakeDocs{}, modeOf(enums.LifeModeTTC),
		fakeBookings{&Booking{ID: 9, Kind: "video", ProviderName: "Dr A", StartsAt: at, DurationMinutes: 20, Href: "/services/bookings/9"}})
	b := hub(t, svc)["upcoming_booking"].(map[string]any)
	assert.Equal(t, map[string]any{
		"id": float64(9), "kind": "video", "provider_name": "Dr A", "starts_at": "2026-10-15T18:00:00+03:30",
		"duration_minutes": float64(20), "href": "/services/bookings/9",
	}, b)
}

func TestHub_ModeErrorPropagates(t *testing.T) {
	boom := errors.New("db down")
	svc := NewService(fakeCatalog{}, &fakeDocs{}, func(context.Context, uint64) (enums.LifeMode, error) { return "", boom }, nil)
	_, err := svc.Hub(context.Background(), 1, pinnedNow, loc)
	assert.ErrorIs(t, err, boom)
}

func TestHub_InsuranceTileAlwaysForAdultsNeverTeen(t *testing.T) {
	cat := fakeCatalog{GroupCare: {item("insurance", "Insurance", `{"icon":"shield","tone":"data"}`), item("labs", "Labs", "")}}
	codes := func(mode enums.LifeMode) []string {
		svc := NewService(cat, &fakeDocs{}, modeOf(mode), nil).WithInsuranceTile()
		var out []string
		for _, c := range hub(t, svc)["care"].([]any) {
			out = append(out, c.(map[string]any)["code"].(string))
		}
		return out
	}
	assert.Equal(t, []string{"insurance", "labs"}, codes(enums.LifeModeCycle), "no published page needed")
	assert.NotContains(t, codes(enums.LifeModeTeen), "insurance")
	svc := NewService(cat, &fakeDocs{}, modeOf(enums.LifeModeCycle), nil).WithInsuranceTile()
	tile := hub(t, svc)["care"].([]any)[0].(map[string]any)
	assert.Equal(t, "/insurance", tile["href"])
	assert.Equal(t, "live", tile["status"])
}

func TestHub_MotherChildSectionGoesLiveWithCityListings(t *testing.T) {
	cat := fakeCatalog{GroupSections: {item("mother_child", "Mother & child", "")}}
	section := func(svc *Service) map[string]any { return hub(t, svc)["sections"].([]any)[0].(map[string]any) }
	base := func() *Service { return NewService(cat, &fakeDocs{}, modeOf(enums.LifeModeCycle), nil) }

	s := section(base())
	assert.Equal(t, "soon", s["status"])
	s = section(base().WithCityServices(func(context.Context) (bool, error) { return false, nil }))
	assert.Equal(t, "soon", s["status"], "no listings yet")
	s = section(base().WithCityServices(func(context.Context) (bool, error) { return true, nil }))
	assert.Equal(t, "/city-services", s["href"])
	assert.Equal(t, "live", s["status"])
	_, err := base().WithCityServices(func(context.Context) (bool, error) { return false, errors.New("db") }).
		Hub(context.Background(), 1, pinnedNow, loc)
	assert.Error(t, err)
}
