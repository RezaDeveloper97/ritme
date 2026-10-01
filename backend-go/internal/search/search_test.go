package search

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/resources/translations"
)

const me uint64 = 7

var today = civildate.MustParse("2026-09-23")

// fakes: every user-scoped read asserts it is asked for the requesting user only.
type fakeLogs struct {
	t     *testing.T
	mode  string
	days  []healthlog.DayEntries
	from  civildate.Date
	calls int
}

func (f *fakeLogs) LifeMode(_ context.Context, userID uint64) (string, error) {
	assert.Equal(f.t, me, userID)
	return f.mode, nil
}

func (f *fakeLogs) Range(_ context.Context, userID uint64, from, to civildate.Date) ([]healthlog.DayEntries, error) {
	assert.Equal(f.t, me, userID)
	f.calls++
	f.from = from
	var out []healthlog.DayEntries
	for _, d := range f.days {
		if !d.Date.Before(from) && !d.Date.After(to) {
			out = append(out, d)
		}
	}
	return out, nil
}

type fakePeriods struct {
	t     *testing.T
	start string
}

func (f fakePeriods) LatestPeriodStart(_ context.Context, userID uint64) (civildate.Date, bool, error) {
	assert.Equal(f.t, me, userID)
	if f.start == "" {
		return civildate.Date{}, false, nil
	}
	return civildate.MustParse(f.start), true, nil
}

type fakeArticles []Article

func (f fakeArticles) PublishedArticles(context.Context) ([]Article, error) { return f, nil }

type fakeCheckups struct {
	t    *testing.T
	list []Checkup
}

func (f fakeCheckups) VisibleCheckups(_ context.Context, userID uint64) ([]Checkup, error) {
	assert.Equal(f.t, me, userID)
	return f.list, nil
}

type fakeReminders struct {
	t    *testing.T
	list []Reminder
}

func (f fakeReminders) CareReminders(_ context.Context, userID uint64) ([]Reminder, error) {
	assert.Equal(f.t, me, userID)
	return f.list, nil
}

// panicking sources prove a scope reads nothing outside its groups.
type noLogs struct{}

func (noLogs) LifeMode(context.Context, uint64) (string, error) { panic("logs read outside scope") }
func (noLogs) Range(context.Context, uint64, civildate.Date, civildate.Date) ([]healthlog.DayEntries, error) {
	panic("logs read outside scope")
}

type noReminders struct{}

func (noReminders) CareReminders(context.Context, uint64) ([]Reminder, error) {
	panic("reminders read outside scope")
}

func entry(cat, param, item, code, num string) taxonomy.Entry {
	e := taxonomy.Entry{Category: cat, Param: param, Item: item}
	if code != "" {
		e.Code = sql.NullString{String: code, Valid: true}
	}
	if num != "" {
		e.Num = sql.NullString{String: num, Valid: true}
	}
	return e
}

func day(date string, es ...taxonomy.Entry) healthlog.DayEntries {
	return healthlog.DayEntries{Date: civildate.MustParse(date), Entries: es}
}

func tr(s string) json.RawMessage { return json.RawMessage(s) }

func query(text, scope, locale string) Query {
	store := i18n.NewTranslationStore(translations.FS, "")
	return Query{
		UserID: me, Text: text, Scope: scope, Locale: locale, Default: "fa", Today: today,
		Taxonomy: store.NamespaceMessages(locale, healthlog.TaxonomyNamespace, "fa"),
	}
}

func painLogs(t *testing.T, mode string) *fakeLogs {
	return &fakeLogs{t: t, mode: mode, days: []healthlog.DayEntries{
		day("2026-09-10", entry("pain", "location", "abdomen", "severe", "9.00")), // before the cycle
		day("2026-09-15", entry("pain", "location", "abdomen", "moderate", "4.00"),
			entry("pain", "relief", "heat", "yes", "")),
		day("2026-09-16", entry("pain", "location", "head", "mild", "6.00"),
			entry("pain", "location", "abdomen", "mild", "2.00")),
		day("2026-09-17", entry("pain", "none", "", "yes", "")), // «no pain» is not a pain day
		day("2026-09-18", entry("symptoms", "general", "fatigue", "no", "")),
	}}
}

func find(g Group, typ, id string) *Hit {
	for i := range g.Items {
		if g.Items[i].Type == typ && g.Items[i].ID == id {
			return &g.Items[i]
		}
	}
	return nil
}

func TestSearch_MineLogInsightsInCurrentCycle(t *testing.T) {
	logs := painLogs(t, taxonomy.ModeCycle)
	svc := NewService(logs, fakePeriods{t: t, start: "2026-09-14"}, fakeArticles{}, fakeCheckups{t: t},
		fakeReminders{t: t, list: []Reminder{
			{ID: 3, Kind: "medication", Title: "مسکن درد", Active: true},
			{ID: 4, Kind: "appointment", Title: "دکتر احمدی"},
		}})
	res, err := svc.Search(context.Background(), query("درد", ScopeMine, "fa"))
	require.NoError(t, err)
	require.Len(t, res.Groups, 1)
	g := res.Groups[0]
	assert.Equal(t, ScopeMine, g.Key)
	assert.Equal(t, civildate.MustParse("2026-09-14"), logs.from)

	pain := find(g, "log_insight", "pain")
	require.NotNil(t, pain)
	assert.Equal(t, "درد", pain.Title)
	assert.Equal(t, "/analysis/symptoms?category=pain", pain.Route)
	b, _ := json.Marshal(pain.JSON())
	assert.JSONEq(t, `{"type":"log_insight","id":"pain","title":"درد","subtitle":null,
		"route":"/analysis/symptoms?category=pain",
		"meta":{"category":"pain","param":null,"item":null,"days":2,
			"window":{"kind":"cycle","from":"2026-09-14","to":"2026-09-23"},
			"peak":{"score":6,"date":"2026-09-16","cycle_day":3}}}`, string(b))

	assert.NotNil(t, find(g, "log_analysis", "pain"))
	med := find(g, "reminder", "3")
	require.NotNil(t, med)
	assert.Equal(t, "/reminders/medication/3", med.Route)
	assert.Nil(t, find(g, "reminder", "4"))
	assert.Equal(t, g.Total, len(g.Items))
}

func TestSearch_MineItemInsight(t *testing.T) {
	svc := NewService(painLogs(t, taxonomy.ModeCycle), fakePeriods{t: t, start: "2026-09-14"}, fakeArticles{},
		fakeCheckups{t: t}, fakeReminders{t: t})
	res, err := svc.Search(context.Background(), query("شكم", ScopeMine, "fa")) // Arabic kaf
	require.NoError(t, err)
	hit := find(res.Groups[0], "log_insight", "pain.location.abdomen")
	require.NotNil(t, hit)
	assert.Equal(t, "شکم", hit.Title)
	assert.Equal(t, "درد", hit.Subtitle)
	days, _ := hit.Meta.Get("days")
	assert.Equal(t, 2, days)
	assert.Equal(t, "/analysis/symptoms?category=pain&item=abdomen&param=location", hit.Route)

	// A «no» level is not a logged symptom: fatigue was logged as «no» only → no hit.
	res, err = svc.Search(context.Background(), query("خستگی", ScopeMine, "fa"))
	require.NoError(t, err)
	assert.Equal(t, 0, res.Groups[0].Total)
}

func TestSearch_MineWindowFallsBackOutsideCycleModes(t *testing.T) {
	for _, c := range []struct{ mode, start string }{
		{taxonomy.ModePregnancy, "2026-09-14"},
		{taxonomy.ModeCycle, ""},           // no period known
		{taxonomy.ModeCycle, "2026-05-01"}, // stale
		{taxonomy.ModeCycle, "2026-09-30"}, // future
	} {
		logs := painLogs(t, c.mode)
		svc := NewService(logs, fakePeriods{t: t, start: c.start}, fakeArticles{}, fakeCheckups{t: t}, fakeReminders{t: t})
		res, err := svc.Search(context.Background(), query("درد", ScopeMine, "fa"))
		require.NoError(t, err)
		assert.Equal(t, civildate.MustParse("2026-08-25"), logs.from, c)
		pain := find(res.Groups[0], "log_insight", "pain")
		require.NotNil(t, pain, c)
		days, _ := pain.Meta.Get("days")
		assert.Equal(t, 3, days, c) // 09-10 joins the 30-day window
		b, _ := json.Marshal(pain.Meta)
		assert.Contains(t, string(b), `"window":{"kind":"days","from":"2026-08-25","to":"2026-09-23"}`, c)
		assert.Contains(t, string(b), `"peak":{"score":9,"date":"2026-09-10","cycle_day":null}`, c)
	}
}

func TestSearch_MineSkipsLogsWhenNoTaxonomyMatch(t *testing.T) {
	logs := painLogs(t, taxonomy.ModeCycle)
	svc := NewService(logs, fakePeriods{t: t}, fakeArticles{}, fakeCheckups{t: t},
		fakeReminders{t: t, list: []Reminder{{ID: 9, Kind: "medication", Title: "قرص آهن"}}})
	res, err := svc.Search(context.Background(), query("قرص آهن", ScopeMine, "fa"))
	require.NoError(t, err)
	assert.Zero(t, logs.calls, "no taxonomy label matched → the log entries are not read")
	require.Equal(t, 1, res.Groups[0].Total)
	assert.Equal(t, "reminder", res.Groups[0].Items[0].Type)
}

func TestSearch_ScopeReadsOnlyItsSources(t *testing.T) {
	svc := NewService(noLogs{}, fakePeriods{t: t}, fakeArticles{
		{Slug: "period-pain", Title: tr(`{"fa":"درد پريود؛ كي عادي است؟","en":"Period pain"}`), ReadTime: 4},
	}, fakeCheckups{t: t}, noReminders{})
	res, err := svc.Search(context.Background(), query("درد پریود", ScopeEducation, "fa"))
	require.NoError(t, err)
	require.Len(t, res.Groups, 1)
	g := res.Groups[0]
	assert.Equal(t, ScopeEducation, g.Key)
	require.Len(t, g.Items, 1)
	b, _ := json.Marshal(g.Items[0].JSON())
	assert.JSONEq(t, `{"type":"article","id":"period-pain","title":"درد پريود؛ كي عادي است؟","subtitle":null,
		"route":"/articles/period-pain","meta":{"kind":"article","category":null,"read_time_minutes":4}}`, string(b))
}

func TestSearch_AllScopeGroupsLimitsAndTotals(t *testing.T) {
	var arts fakeArticles
	for _, s := range []string{"a", "b", "c", "d", "e"} {
		arts = append(arts, Article{Slug: s, Title: tr(`{"fa":"یوگا ` + s + `"}`)})
	}
	arts = append(arts, Article{Slug: "x", Title: tr(`{"fa":"تغذیه"}`), Excerpt: tr(`{"fa":"یوگا و غذا"}`)})
	svc := NewService(painLogs(t, taxonomy.ModeCycle), fakePeriods{t: t}, arts, fakeCheckups{t: t}, fakeReminders{t: t})

	res, err := svc.Search(context.Background(), query("یوگا", ScopeAll, "fa"))
	require.NoError(t, err)
	keys := []string{}
	for _, g := range res.Groups {
		keys = append(keys, g.Key)
	}
	assert.Equal(t, []string{"mine", "programs", "education", "services"}, keys)
	edu := res.Groups[2]
	assert.Equal(t, 6, edu.Total)
	require.Len(t, edu.Items, DefaultLimitAll)
	assert.Equal(t, "a", edu.Items[0].ID) // title hits before the excerpt-only hit, source order kept

	q := query("یوگا", ScopeEducation, "fa")
	q.Limit = 5
	res, err = svc.Search(context.Background(), q)
	require.NoError(t, err)
	assert.Len(t, res.Groups[0].Items, 5)
	assert.Equal(t, 6, res.Groups[0].Total)

	q.Limit = 0
	res, err = svc.Search(context.Background(), q)
	require.NoError(t, err)
	assert.Len(t, res.Groups[0].Items, 6)
	assert.Equal(t, "x", res.Groups[0].Items[5].ID)
}

func TestSearch_ProgramsAndServices(t *testing.T) {
	svc := NewService(noLogs{}, fakePeriods{t: t}, fakeArticles{}, fakeCheckups{t: t, list: []Checkup{
		{ID: 11, Key: "pap_smear", Category: "screening", Title: tr(`{"fa":"پاپ اسمير","en":"Pap smear"}`)},
		{ID: 12, Key: "breast_self_exam", Category: "self", Title: tr(`{"fa":"معاینه پستان در خانه"}`)},
		{ID: 13, Custom: true, Category: "custom", Title: tr(`{"fa":"آزمایش تیروئید"}`)},
	}}, noReminders{})

	res, err := svc.Search(context.Background(), query("کگل", ScopePrograms, "fa"))
	require.NoError(t, err)
	require.Equal(t, 1, res.Groups[0].Total)
	assert.Equal(t, "pelvic", res.Groups[0].Items[0].ID)
	assert.Equal(t, "/programs/pelvic", res.Groups[0].Items[0].Route)

	res, err = svc.Search(context.Background(), query("iud", ScopePrograms, "en"))
	require.NoError(t, err)
	require.Equal(t, 1, res.Groups[0].Total)
	assert.Equal(t, "Contraception", res.Groups[0].Items[0].Title)

	res, err = svc.Search(context.Background(), query("پاپ", ScopeServices, "fa"))
	require.NoError(t, err)
	g := res.Groups[0]
	assert.NotNil(t, find(g, "service", "checkups")) // keyword «پاپ اسمیر»
	pap := find(g, "checkup", "11")
	require.NotNil(t, pap)
	assert.Equal(t, "/checkups/11", pap.Route)
	assert.Equal(t, "checkup", g.Items[0].Type, "the title hit ranks above the keyword hit")

	res, err = svc.Search(context.Background(), query("معاينه", ScopeServices, "fa"))
	require.NoError(t, err)
	self := find(res.Groups[0], "checkup", "12")
	require.NotNil(t, self)
	assert.Equal(t, "/checkups/self-exam", self.Route)

	res, err = svc.Search(context.Background(), query("تیروئید", ScopeServices, "fa"))
	require.NoError(t, err)
	custom := find(res.Groups[0], "checkup", "13")
	require.NotNil(t, custom)
	isCustom, _ := custom.Meta.Get("custom")
	assert.Equal(t, true, isCustom)
}

func TestSearch_EmptyQueryReadsNothing(t *testing.T) {
	svc := NewService(noLogs{}, fakePeriods{t: t}, fakeArticles{}, fakeCheckups{t: t}, noReminders{})
	res, err := svc.Search(context.Background(), query("،،", ScopeAll, "fa"))
	require.NoError(t, err)
	require.Len(t, res.Groups, 4)
	for _, g := range res.Groups {
		assert.Zero(t, g.Total)
		assert.NotNil(t, g.Items)
	}
}

func TestLangFiles_SameKeys(t *testing.T) {
	for _, d := range append(append([]destination{}, programRegistry...), serviceRegistry...) {
		for _, loc := range []string{"fa", "en"} {
			section := "programs."
			for _, s := range serviceRegistry {
				if s.code == d.code {
					section = "services."
				}
			}
			assert.NotEmpty(t, T(section+d.code+".title", loc), "%s %s", loc, d.code)
			assert.NotEmpty(t, T(section+d.code+".keywords", loc), "%s %s", loc, d.code)
		}
	}
	assert.Len(t, attributes("fa"), 6)
	assert.Len(t, attributes("en"), 6)
}
