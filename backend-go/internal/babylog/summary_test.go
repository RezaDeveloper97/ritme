package babylog

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/babylog/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

var day = civildate.MustParse("2026-10-03")

func at(hhmm string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", day.String()+" "+hhmm, civildate.Tehran)
	if err != nil {
		panic(err)
	}
	return t
}

func ns(s string) sql.NullString  { return sql.NullString{String: s, Valid: true} }
func nt(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

func TestTimesOf(t *testing.T) {
	now := at("14:30")
	// Running breast feed: 7 min closed on the right, left running since 14:22.
	f := store.BabyFeed{Type: "breast", StartedAt: at("14:10"), RightSeconds: 420, ActiveSide: ns("left"),
		SideStartedAt: nt(at("14:22")), ActiveLock: active}
	assert.Equal(t, FeedTimes{Left: 480, Right: 420, Total: 900}, TimesOf(f, now))
	// Paused: no running segment.
	f.ActiveSide, f.SideStartedAt = sql.NullString{}, sql.NullTime{}
	assert.Equal(t, FeedTimes{Right: 420, Total: 420}, TimesOf(f, now))
	// Bottle: elapsed while running, the stored duration once ended.
	b := store.BabyFeed{Type: "bottle", StartedAt: at("14:00"), ActiveLock: active}
	assert.Equal(t, int64(1800), TimesOf(b, now).Total)
	b.ActiveLock, b.EndedAt, b.DurationSeconds = sql.NullInt16{}, nt(at("14:12")), sql.NullInt32{Int32: 720, Valid: true}
	assert.Equal(t, int64(720), TimesOf(b, now).Total)
}

func TestCloseSegment(t *testing.T) {
	f := store.BabyFeed{Type: "breast", LeftSeconds: 60, ActiveSide: ns("right"), SideStartedAt: nt(at("10:00"))}
	closeSegment(&f, at("10:05"))
	assert.Equal(t, uint32(60), f.LeftSeconds)
	assert.Equal(t, uint32(300), f.RightSeconds)
	assert.Equal(t, "right", f.LastSide.String)
	assert.False(t, f.ActiveSide.Valid)
	assert.False(t, f.SideStartedAt.Valid)
	// A clock going backwards never subtracts.
	f.ActiveSide, f.SideStartedAt = ns("left"), nt(at("11:00"))
	closeSegment(&f, at("10:59"))
	assert.Equal(t, uint32(60), f.LeftSeconds)
}

func TestSummarize(t *testing.T) {
	now := at("23:00")
	feeds := []store.BabyFeed{
		{Type: "breast", StartedAt: at("14:20"), LeftSeconds: 300, RightSeconds: 420, DurationSeconds: sql.NullInt32{Int32: 720, Valid: true}, EndedAt: nt(at("14:32"))},
		{Type: "bottle", StartedAt: at("18:00"), AmountMl: sql.NullInt16{Int16: 90, Valid: true}, DurationSeconds: sql.NullInt32{Int32: 600, Valid: true}, EndedAt: nt(at("18:10"))},
		{Type: "pump", StartedAt: at("20:00"), AmountMl: sql.NullInt16{Int16: 120, Valid: true}, DurationSeconds: sql.NullInt32{Int32: 900, Valid: true}, EndedAt: nt(at("20:15"))},
		{Type: "breast", StartedAt: at("09:00").AddDate(0, 0, -1), LeftSeconds: 600}, // yesterday
	}
	sleeps := []store.BabySleep{
		// 22:00 yesterday → 02:00 today: 2 h today.
		{StartedAt: at("22:00").AddDate(0, 0, -1), EndedAt: nt(at("02:00"))},
		{StartedAt: at("13:00"), EndedAt: nt(at("14:00"))},
		// Running since 22:30: 30 min so far.
		{StartedAt: at("22:30"), ActiveLock: active},
	}
	diapers := []store.BabyDiaper{
		{ChangedAt: at("08:00"), Kind: "wet"}, {ChangedAt: at("12:00"), Kind: "both"}, {ChangedAt: at("16:00"), Kind: "dirty"},
	}
	days := Summarize(day.AddDays(-1), 2, feeds, sleeps, diapers, now)
	require.Len(t, days, 2)
	y, d := days[0], days[1]
	assert.Equal(t, 1, y.Feeds.Count)
	assert.Equal(t, int64(2*3600), y.Sleep.Seconds, "22:00–24:00 yesterday")
	assert.Equal(t, 1, y.Sleep.Count)

	assert.Equal(t, FeedTotals{Count: 3, Breast: 1, Bottle: 1, Pump: 1, LeftSeconds: 300, RightSeconds: 420,
		TotalSeconds: 720 + 600 + 900, BottleMl: 90, PumpMl: 120}, d.Feeds)
	assert.Equal(t, SleepTotals{Count: 2, Seconds: 2*3600 + 3600 + 1800, LongestSeconds: 3600}, d.Sleep)
	assert.Equal(t, DiaperTotals{Count: 3, Wet: 2, Dirty: 2, Both: 1}, d.Diapers)
}

func TestNextSideAndAverages(t *testing.T) {
	assert.Equal(t, "", NextSide(""))
	assert.Equal(t, "left", NextSide("right"))
	assert.Equal(t, "right", NextSide("left"))

	days := []Day{
		{Feeds: FeedTotals{Count: 8, LeftSeconds: 480, RightSeconds: 520}, Sleep: SleepTotals{Seconds: 50000}, Diapers: DiaperTotals{Count: 6}},
		{Feeds: FeedTotals{Count: 7}, Sleep: SleepTotals{Seconds: 52000}, Diapers: DiaperTotals{Count: 5}},
		{Feeds: FeedTotals{Count: 8}},
	}
	raw, err := json.Marshal(AveragesJSON(days))
	require.NoError(t, err)
	assert.JSONEq(t, `{"feeds_per_day":7.7,"sleep_seconds_per_day":34000,"diapers_per_day":3.7,"left_percent":48,"right_percent":52}`, string(raw))
	raw, err = json.Marshal(AveragesJSON(nil))
	require.NoError(t, err)
	assert.JSONEq(t, `{"feeds_per_day":0,"sleep_seconds_per_day":0,"diapers_per_day":0,"left_percent":null,"right_percent":null}`, string(raw))
}

// Every language file has the English keys (English is the fallback, so a missing key would show English).
func TestLangFilesHaveTheSameKeys(t *testing.T) {
	keys := func(code string) []string {
		raw, err := fs.ReadFile(langFS, "lang/"+code+"/babylog.json")
		require.NoError(t, err)
		var m map[string]map[string]string
		require.NoError(t, json.Unmarshal(raw, &m))
		var out []string
		for g, kv := range m {
			for k := range kv {
				out = append(out, g+"."+k)
			}
		}
		return out
	}
	dirs, err := fs.ReadDir(langFS, "lang")
	require.NoError(t, err)
	for _, d := range dirs {
		assert.ElementsMatch(t, keys("en"), keys(d.Name()), d.Name())
	}
	assert.Equal(t, "Saved", T("messages.saved", "en"))
	assert.NotEmpty(t, attributes("fa"))
}
