package inbox

import (
	"database/sql"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func bpRule(id uint64, level, kind string) *rules.Rule {
	return &rules.Rule{ID: id, Level: level, Kind: kind, Placement: Slot, IsActive: true, Status: rules.StatusApproved,
		AudiencePercent: 100, HoursFrom: -1, HoursTo: -1, Version: 1, ApprovedVersion: 1,
		Groups: []rules.Group{{Op: "and", Items: []rules.Condition{{Fact: "bp_sys", Op: "gte", Value: 140.0}}}},
		Content: rules.Content{A: rules.Variant{Title: "High blood pressure", Body: "Recheck today"},
			Seen: "{bp_sys} systolic", Confidence: "From your readings"}}
}

func openRow(id uint64, r *rules.Rule, state, key string) store.ListUserOpenInboxRow {
	return store.ListUserOpenInboxRow{ID: id, RuleID: r.ID, Kind: r.Kind, Level: r.Level, State: state,
		EvidenceKey: sql.NullString{String: key, Valid: true}, HitCount: 1,
		LastAt: sql.NullTime{Time: time.Date(2026, 10, 30, 9, 0, 0, 0, civildate.Tehran), Valid: true}}
}

func keyOf(t *testing.T, r *rules.Rule, f rules.Facts) string {
	t.Helper()
	_, k, ok := EvidenceOf(r, f)
	require.True(t, ok)
	return k
}

// Merge: the same evidence key changes nothing; a new key merges (count + 1 is the SQL's); an acknowledged urgent row
// becomes active again and loses its ack (M-16), an acknowledged follow_up row stays acknowledged.
func TestDiff_Merge(t *testing.T) {
	r := bpRule(1, rules.LevelWorthFollowUp, rules.KindClinical)
	f := rules.Facts{"bp_sys": 150.0}
	win := []rules.Choice{{Rule: r, Variant: "a"}}
	idx := map[uint64]*rules.Rule{1: r}

	same := Diff(win, []store.ListUserOpenInboxRow{openRow(7, r, StateActive, keyOf(t, r, f))}, idx, f)
	assert.False(t, same.Changed())
	assert.Len(t, same.Open, 1)

	newer := Diff(win, []store.ListUserOpenInboxRow{openRow(7, r, StateAcknowledged, "old")}, idx, rules.Facts{"bp_sys": 160.0})
	require.Len(t, newer.Merges, 1)
	assert.Equal(t, StateAcknowledged, newer.Merges[0].State)
	assert.NotEqual(t, "old", newer.Merges[0].Key)

	u := bpRule(2, rules.LevelFollowSoon, rules.KindSafety)
	acked := openRow(8, u, StateAcknowledged, "old")
	acked.AckedAt = sql.NullTime{Time: time.Now(), Valid: true}
	urgent := Diff([]rules.Choice{{Rule: u}}, []store.ListUserOpenInboxRow{acked}, map[uint64]*rules.Rule{2: u}, f)
	require.Len(t, urgent.Merges, 1)
	assert.Equal(t, StateActive, urgent.Merges[0].State)
	assert.False(t, urgent.Merges[0].AckedAt.Valid)
	assert.Equal(t, StateActive, urgent.Open[0].State)
}

// Resolve only when the condition drops over known facts; a missing fact never resolves.
func TestDiff_Resolve(t *testing.T) {
	r := bpRule(1, rules.LevelWorthFollowUp, rules.KindClinical)
	rows := []store.ListUserOpenInboxRow{openRow(7, r, StateActive, "k")}
	idx := map[uint64]*rules.Rule{1: r}
	p := Diff(nil, rows, idx, rules.Facts{"bp_sys": 120.0})
	assert.Equal(t, []uint64{7}, p.Resolves)
	assert.Empty(t, p.Open)

	p = Diff(nil, rows, idx, rules.Facts{})
	assert.Empty(t, p.Resolves)
	assert.Len(t, p.Open, 1)

	// a rule no longer served is left alone (and still counted)
	p = Diff(nil, rows, map[uint64]*rules.Rule{}, rules.Facts{"bp_sys": 120.0})
	assert.Empty(t, p.Resolves)
	assert.Len(t, p.Open, 1)
}

// follow_up without evidence → no row; info needs none.
func TestDiff_MissingEvidence(t *testing.T) {
	r := bpRule(1, rules.LevelWorthFollowUp, rules.KindClinical)
	r.Content.Seen = ""
	p := Diff([]rules.Choice{{Rule: r}}, nil, map[uint64]*rules.Rule{1: r}, rules.Facts{"bp_sys": 150.0})
	assert.Empty(t, p.Inserts)
	assert.Equal(t, 1, p.Missing)

	r2 := bpRule(2, rules.LevelWorthFollowUp, rules.KindClinical)
	r2.Content.Seen = "{missing_fact} readings"
	p = Diff([]rules.Choice{{Rule: r2}}, nil, nil, rules.Facts{"bp_sys": 150.0})
	assert.Equal(t, 1, p.Missing)

	info := bpRule(3, rules.LevelInfo, rules.KindContent)
	info.Content.Seen, info.Content.Confidence = "", ""
	p = Diff([]rules.Choice{{Rule: info}}, nil, nil, rules.Facts{"bp_sys": 150.0})
	assert.Len(t, p.Inserts, 1)
	assert.Equal(t, StateActive, p.Open[0].State)
}

// The evidence key follows the evidence variables; without any, the condition facts.
func TestEvidenceKey(t *testing.T) {
	r := bpRule(1, rules.LevelWorthFollowUp, rules.KindClinical)
	a, b := keyOf(t, r, rules.Facts{"bp_sys": 150.0, "other": 1.0}), keyOf(t, r, rules.Facts{"bp_sys": 150.0, "other": 2.0})
	assert.Equal(t, a, b)
	assert.NotEqual(t, a, keyOf(t, r, rules.Facts{"bp_sys": 151.0}))
	ev, _, _ := EvidenceOf(r, rules.Facts{"bp_sys": 150.0})
	assert.JSONEq(t, `{"vars":{"bp_sys":150}}`, string(ev))

	info := bpRule(2, rules.LevelInfo, rules.KindContent)
	info.Content.Seen = ""
	assert.NotEqual(t, keyOf(t, info, rules.Facts{"bp_sys": 150.0}), keyOf(t, info, rules.Facts{"bp_sys": 145.0}))
}

func TestCursorRoundTrip(t *testing.T) {
	c := Cursor{Rank: 2, At: time.Date(2026, 10, 31, 8, 40, 0, 0, civildate.Tehran), ID: 99}
	got, err := ParseCursor(c.Encode())
	require.NoError(t, err)
	assert.Equal(t, c.Rank, got.Rank)
	assert.True(t, c.At.Equal(got.At))
	assert.Equal(t, c.ID, got.ID)
	for _, bad := range []string{"%%%", "bm90LWEtY3Vyc29y", "OTo5Ojk"} {
		_, err := ParseCursor(bad)
		assert.ErrorIs(t, err, ErrBadCursor, bad)
	}
	none, err := ParseCursor("")
	assert.NoError(t, err)
	assert.Nil(t, none)
}

func TestClosable(t *testing.T) {
	acked := sql.NullTime{Time: time.Now(), Valid: true}
	assert.False(t, (&Row{Level: rules.LevelFollowSoon, State: StateActive}).Closable())
	assert.True(t, (&Row{Level: rules.LevelFollowSoon, State: StateAcknowledged, AckedAt: acked}).Closable())
	assert.False(t, (&Row{Level: rules.LevelFollowSoon, State: StateNoted}).Closable())
	assert.True(t, (&Row{Level: rules.LevelFollowSoon, State: StateResolved}).Closable())
	assert.True(t, (&Row{Level: rules.LevelWorthFollowUp, State: StateActive}).Closable())
	assert.False(t, (&Row{Level: rules.LevelInfo, State: StateArchived}).Closable())
}

// A row last hit on today − 30 is before the cutoff; one hit on today − 29 is not.
func TestArchiveCutoff(t *testing.T) {
	now := time.Date(2026, 11, 30, 4, 10, 0, 0, civildate.Tehran)
	cut := ArchiveCutoff(now)
	day30 := time.Date(2026, 10, 31, 23, 59, 0, 0, civildate.Tehran)
	day29 := time.Date(2026, 11, 1, 0, 0, 0, 0, civildate.Tehran)
	assert.True(t, day30.Before(cut))
	assert.False(t, day29.Before(cut))
}

// healthTerms is the lock-screen deny-list: the push text must name no health detail in any language.
var healthTerms = []string{
	"پریود", "قاعدگی", "باردار", "بارداری", "تخمک", "باروری", "خون", "فشار", "قند", "جنین", "دارو", "علامت", "درد",
	"سقط", "یائسگی", "زایمان",
	"period", "menstru", "pregnan", "ovulat", "fertil", "blood", "pressure", "sugar", "fetal", "fetus", "bleed",
	"symptom", "pain", "medic", "pill", "menopaus", "birth", "cycle",
}

func TestNeutralPushText_NoHealthTerms(t *testing.T) {
	sub, err := fs.Sub(langFS, "lang")
	require.NoError(t, err)
	codes, err := fs.ReadDir(sub, ".")
	require.NoError(t, err)
	require.NotEmpty(t, codes)
	for _, d := range codes {
		title, body := NeutralText(d.Name())
		require.NotEmpty(t, title, d.Name())
		require.NotEmpty(t, body, d.Name())
		text := strings.ToLower(title + " " + body)
		for _, w := range healthTerms {
			assert.NotContains(t, text, w, d.Name())
		}
	}
	title, _ := NeutralText("fa")
	assert.Equal(t, "یادآور ریتمی", title)
}

func TestCategoryOf(t *testing.T) {
	assert.EqualValues(t, "articles", CategoryOf(rules.KindProduct))
	assert.EqualValues(t, "daily_log", CategoryOf(rules.KindSafety))
}
