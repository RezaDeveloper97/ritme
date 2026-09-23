package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
)

var d = civildate.MustParse

// at is a clock at noon Tehran time on day s.
func at(s string) clock.Fixed { return clock.At(d(s).TehranMidnight().Add(12 * time.Hour)) }

// The default catalog, as seeded by 00003_checkups.sql.
var (
	selfExam = Type{ID: 1, Key: "breast_self_exam", Category: CategoryMonthly, IntervalMonths: 1,
		CycleDayFrom: 7, CycleDayTo: 10, RemindLeadDays: 3, HideInPregnancy: true, IsActive: true, SortOrder: 1}
	clinical = Type{ID: 2, Key: "clinical_breast_exam", Category: CategoryAnnual, IntervalMonths: 12,
		RemindLeadDays: 30, IsActive: true, SortOrder: 2}
	pap = Type{ID: 3, Key: "pap_smear", Category: CategoryMultiYear, IntervalMonths: 36, AgeMin: 21, AgeMax: 65,
		CycleDayFrom: 10, CycleDayTo: 20, RemindLeadDays: 30, IsActive: true, SortOrder: 3}
	blood = Type{ID: 4, Key: "blood_test", Category: CategoryAnnual, IntervalMonths: 12,
		RemindLeadDays: 14, IsActive: true, SortOrder: 4}
	dentist = Type{ID: 5, Key: "dentist", Category: CategorySixMonthly, IntervalMonths: 6,
		RemindLeadDays: 14, IsActive: true, SortOrder: 5}
	mammo = Type{ID: 6, Key: "mammography", Category: CategoryAgeBased, IntervalMonths: 12, IntervalMonthsMax: 24,
		AgeMin: 40, RemindLeadDays: 30, HideInPregnancy: true, IsActive: true, SortOrder: 6}
	catalog = []Type{selfExam, clinical, pap, blood, dentist, mammo}

	// A 28-day cycle whose current period started on 2026-09-20 (today 2026-09-23 = cycle day 4).
	cycle28 = Cycle{Start: d("2026-09-20"), Length: 28}
	born34  = d("1992-06-15") // 34 on 2026-09-23, turns 40 on 2032-06-15
	born46  = d("1980-01-10")
)

const today = "2026-09-23" // 1 Mehr 1405

func rec(id, typeID uint64, doneOn string) Record {
	return Record{ID: id, TypeID: typeID, DoneOn: d(doneOn)}
}

func find(t *testing.T, res Result, typeID uint64) Item {
	t.Helper()
	for _, it := range res.Items {
		if it.TypeID == typeID {
			return it
		}
	}
	t.Fatalf("type %d not in the result", typeID)
	return Item{}
}

func TestEvaluate_Item(t *testing.T) {
	tests := []struct {
		name  string
		now   string
		typ   Type
		in    Input
		want  Status
		sect  Section
		next  string // "" = zero date
		by    string // "" = same as next
		cycle bool
	}{
		{name: "never recorded is due without a date", typ: clinical,
			want: StatusDue, sect: Section(CategoryAnnual)},
		{name: "overdue by months, cycle-timed Pap lands in this month", typ: pap,
			in:   Input{Birthday: born34, Records: []Record{rec(1, 3, "2022-03-25")}},
			want: StatusOverdue, sect: SectionThisMonth, next: "2025-03-25"},
		{name: "overdue by months, plain annual lands in the overdue section", typ: blood,
			in:   Input{Records: []Record{rec(1, 4, "2025-01-01")}},
			want: StatusOverdue, sect: SectionOverdue, next: "2026-01-01"},
		{name: "due inside the lead window", typ: clinical,
			in:   Input{Records: []Record{rec(1, 2, "2025-10-10")}},
			want: StatusDue, sect: Section(CategoryAnnual), next: "2026-10-10"},
		{name: "lead window starts exactly today", typ: clinical,
			in:   Input{Records: []Record{rec(1, 2, "2025-10-23")}},
			want: StatusDue, sect: Section(CategoryAnnual), next: "2026-10-23"},
		{name: "soon within 60 days", typ: dentist,
			in:   Input{Records: []Record{rec(1, 5, "2026-05-01")}},
			want: StatusSoon, sect: Section(CategorySixMonthly), next: "2026-11-01"},
		{name: "up to date", typ: blood,
			in:   Input{Records: []Record{rec(1, 4, "2026-02-01")}},
			want: StatusUpToDate, sect: Section(CategoryAnnual), next: "2027-02-01"},
		{name: "age not_yet carries the start date", typ: mammo,
			in:   Input{Birthday: born34},
			want: StatusNotYet, sect: Section(CategoryAgeBased), next: "2032-06-15"},
		{name: "age not_yet wins over records", typ: mammo,
			in:   Input{Birthday: born34, Records: []Record{rec(1, 6, "2020-01-01")}},
			want: StatusNotYet, sect: Section(CategoryAgeBased), next: "2032-06-15"},
		{name: "unknown birthday skips the age rules", typ: mammo,
			want: StatusDue, sect: Section(CategoryAgeBased)},
		{name: "range interval: past the minimum is due, not overdue", typ: mammo,
			in:   Input{Birthday: born46, Records: []Record{rec(1, 6, "2025-06-01")}},
			want: StatusDue, sect: Section(CategoryAgeBased), next: "2026-06-01", by: "2027-06-01"},
		{name: "range interval: past the maximum is overdue", typ: mammo,
			in:   Input{Birthday: born46, Records: []Record{rec(1, 6, "2024-06-01")}},
			want: StatusOverdue, sect: SectionOverdue, next: "2025-06-01", by: "2026-06-01"},
		{name: "range interval: before the minimum is up to date", typ: mammo,
			in:   Input{Birthday: born46, Records: []Record{rec(1, 6, "2026-01-01")}},
			want: StatusUpToDate, sect: Section(CategoryAgeBased), next: "2027-01-01", by: "2028-01-01"},
		{name: "user next_due_on override in the past is overdue", typ: dentist,
			in:   Input{Records: []Record{{ID: 1, TypeID: 5, DoneOn: d("2026-05-01"), NextDueOn: d("2026-09-20")}}},
			want: StatusOverdue, sect: SectionOverdue, next: "2026-09-20"},
		{name: "user next_due_on override in the future is up to date", typ: blood,
			in:   Input{Records: []Record{{ID: 1, TypeID: 4, DoneOn: d("2025-01-01"), NextDueOn: d("2027-03-01")}}},
			want: StatusUpToDate, sect: Section(CategoryAnnual), next: "2027-03-01"},
		{name: "only the latest record's override counts", typ: blood,
			in: Input{Records: []Record{
				{ID: 1, TypeID: 4, DoneOn: d("2025-01-01"), NextDueOn: d("2025-02-01")},
				rec(2, 4, "2026-02-01"),
			}},
			want: StatusUpToDate, sect: Section(CategoryAnnual), next: "2027-02-01"},
		{name: "same-day records: the higher id wins", typ: blood,
			in: Input{Records: []Record{
				{ID: 7, TypeID: 4, DoneOn: d("2026-02-01"), NextDueOn: d("2026-09-01")},
				rec(3, 4, "2026-02-01"),
			}},
			want: StatusOverdue, sect: SectionOverdue, next: "2026-09-01"},
		{name: "disabled setting", typ: clinical,
			in:   Input{Records: []Record{rec(1, 2, "2024-01-01")}, Settings: []Setting{{TypeID: 2, Enabled: false}}},
			want: StatusDisabled, sect: Section(CategoryAnnual), next: "2025-01-01"},
		{name: "self-exam never recorded: upcoming window this month", typ: selfExam, in: Input{Cycle: cycle28},
			want: StatusDue, sect: SectionThisMonth, next: "2026-09-26", by: "2026-09-29", cycle: true},
		{name: "self-exam never recorded after this cycle's window: next cycle, next month", typ: selfExam,
			now: "2026-10-01", in: Input{Cycle: cycle28},
			want: StatusDue, sect: Section(CategoryMonthly), next: "2026-10-24", by: "2026-10-27", cycle: true},
		{name: "self-exam done last cycle: due in the lead days before the window", typ: selfExam,
			in:   Input{Cycle: cycle28, Records: []Record{rec(1, 1, "2026-08-31")}},
			want: StatusDue, sect: SectionThisMonth, next: "2026-09-26", by: "2026-09-29", cycle: true},
		{name: "self-exam inside the window is still due, not overdue", typ: selfExam, now: "2026-09-28",
			in:   Input{Cycle: cycle28, Records: []Record{rec(1, 1, "2026-08-31")}},
			want: StatusDue, sect: SectionThisMonth, next: "2026-09-26", by: "2026-09-29", cycle: true},
		{name: "self-exam missed the last window is overdue", typ: selfExam,
			in:   Input{Cycle: cycle28, Records: []Record{rec(1, 1, "2026-08-20")}},
			want: StatusOverdue, sect: SectionThisMonth, next: "2026-08-29", by: "2026-09-01", cycle: true},
		{name: "self-exam done this cycle: next cycle's window", typ: selfExam,
			in:   Input{Cycle: cycle28, Records: []Record{rec(1, 1, "2026-09-21")}},
			want: StatusSoon, sect: Section(CategoryMonthly), next: "2026-10-24", by: "2026-10-27", cycle: true},
		{name: "self-exam without cycle data falls back to a monthly date", typ: selfExam,
			in:   Input{Records: []Record{rec(1, 1, "2026-09-01")}},
			want: StatusSoon, sect: Section(CategoryMonthly), next: "2026-10-01"},
		{name: "self-exam without cycle data, a month late", typ: selfExam,
			in:   Input{Records: []Record{rec(1, 1, "2026-08-10")}},
			want: StatusOverdue, sect: SectionThisMonth, next: "2026-09-10"},
		{name: "Pap is cycle-timed for sections only: its date stays on the calendar", typ: pap,
			in:   Input{Birthday: born34, Cycle: cycle28, Records: []Record{rec(1, 3, "2024-01-15")}},
			want: StatusUpToDate, sect: Section(CategoryMultiYear), next: "2027-01-15"},
		{name: "custom checkup", typ: Type{ID: 9, UserID: 42, Category: CategoryCustom, IntervalMonths: 3, IsActive: true},
			in:   Input{Records: []Record{rec(1, 9, "2026-08-01")}},
			want: StatusSoon, sect: Section(CategoryCustom), next: "2026-11-01"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			now := tc.now
			if now == "" {
				now = today
			}
			in := tc.in
			in.Types = []Type{tc.typ}
			res := Evaluate(in, at(now))
			require.Len(t, res.Items, 1)
			it := res.Items[0]
			assert.Equal(t, tc.want, it.Status, "status")
			assert.Equal(t, tc.sect, it.Section, "section")
			var next, by civildate.Date
			if tc.next != "" {
				next = d(tc.next)
			}
			by = next
			if tc.by != "" {
				by = d(tc.by)
			}
			assert.Equal(t, next, it.NextDueOn, "next due")
			assert.Equal(t, by, it.DueBy, "due by")
			assert.Equal(t, tc.cycle, it.CycleTimed, "cycle-timed")
			assert.Equal(t, tc.typ.Key, it.Key)
			assert.Equal(t, tc.typ.UserID != 0, it.IsCustom)
		})
	}
}

func TestEvaluate_Applicability(t *testing.T) {
	t.Run("pregnancy hides the marked types", func(t *testing.T) {
		res := Evaluate(Input{Types: catalog, Birthday: born46, Pregnant: true}, at(today))
		keys := []string{}
		for _, it := range res.Items {
			keys = append(keys, it.Key)
		}
		assert.Equal(t, []string{"clinical_breast_exam", "pap_smear", "blood_test", "dentist"}, keys)
	})
	t.Run("not pregnant shows them", func(t *testing.T) {
		res := Evaluate(Input{Types: catalog, Birthday: born46}, at(today))
		assert.Len(t, res.Items, 6)
	})
	t.Run("older than age_max drops the type", func(t *testing.T) {
		res := Evaluate(Input{Types: catalog, Birthday: d("1950-01-01")}, at(today))
		for _, it := range res.Items {
			assert.NotEqual(t, "pap_smear", it.Key)
		}
		assert.Len(t, res.Items, 5)
		require.NotNil(t, res.Age)
		assert.Equal(t, 76, *res.Age)
	})
	t.Run("inactive types are skipped", func(t *testing.T) {
		off := blood
		off.IsActive = false
		res := Evaluate(Input{Types: []Type{off, dentist}}, at(today))
		require.Len(t, res.Items, 1)
		assert.Equal(t, "dentist", res.Items[0].Key)
	})
	t.Run("items follow sort_order, then id", func(t *testing.T) {
		a := Type{ID: 20, Category: CategoryCustom, IntervalMonths: 1, IsActive: true, SortOrder: 0}
		b := Type{ID: 10, Category: CategoryCustom, IntervalMonths: 1, IsActive: true, SortOrder: 0}
		res := Evaluate(Input{Types: []Type{dentist, a, b}}, at(today))
		ids := []uint64{}
		for _, it := range res.Items {
			ids = append(ids, it.TypeID)
		}
		assert.Equal(t, []uint64{10, 20, 5}, ids)
	})
	t.Run("settings: remind off stays enabled", func(t *testing.T) {
		res := Evaluate(Input{Types: []Type{dentist}, Settings: []Setting{{TypeID: 5, Enabled: true, Remind: false}}}, at(today))
		it := find(t, res, 5)
		assert.True(t, it.Enabled)
		assert.False(t, it.Remind)
		assert.Equal(t, StatusDue, it.Status)
	})
	t.Run("no settings row: enabled with reminders", func(t *testing.T) {
		it := find(t, Evaluate(Input{Types: []Type{dentist}}, at(today)), 5)
		assert.True(t, it.Enabled)
		assert.True(t, it.Remind)
	})
	t.Run("unknown birthday: no age", func(t *testing.T) {
		assert.Nil(t, Evaluate(Input{Types: catalog}, at(today)).Age)
	})
	t.Run("empty input: empty list, not nil", func(t *testing.T) {
		res := Evaluate(Input{}, at(today))
		assert.NotNil(t, res.Items)
		assert.Equal(t, Summary{}, res.Summary)
	})
}

// The v14 artboards: a 34-year-old, «۴ از ۶ به‌روز · ۱ موعدش رسیده · ۱ عقب‌افتاده».
func TestEvaluate_ArtboardSummary(t *testing.T) {
	in := Input{
		Types:    catalog,
		Birthday: born34,
		Cycle:    cycle28,
		Records: []Record{
			rec(1, 1, "2026-08-31"), // self-exam last cycle → due in 3 days
			rec(2, 3, "2022-03-25"), // Pap Farvardin 1401 → overdue
			rec(3, 2, "2025-10-30"), // clinical Mehr 1404 → soon
			rec(4, 4, "2026-02-01"), // blood Bahman 1404 → up to date
			rec(5, 5, "2026-06-01"), // dentist Khordad 1405 → up to date
		},
	}
	res := Evaluate(in, at(today))
	assert.Equal(t, Summary{Total: 6, UpToDate: 4, Due: 1, Overdue: 1}, res.Summary)
	require.NotNil(t, res.Age)
	assert.Equal(t, 34, *res.Age)

	sections := map[string]Section{}
	statuses := map[string]Status{}
	for _, it := range res.Items {
		sections[it.Key], statuses[it.Key] = it.Section, it.Status
	}
	assert.Equal(t, map[string]Status{
		"breast_self_exam": StatusDue, "pap_smear": StatusOverdue, "clinical_breast_exam": StatusSoon,
		"blood_test": StatusUpToDate, "dentist": StatusUpToDate, "mammography": StatusNotYet,
	}, statuses)
	assert.Equal(t, map[string]Section{
		"breast_self_exam": SectionThisMonth, "pap_smear": SectionThisMonth, "clinical_breast_exam": "annual",
		"blood_test": "annual", "dentist": "six_monthly", "mammography": "age_based",
	}, sections)

	t.Run("a disabled item leaves the summary", func(t *testing.T) {
		in := in
		in.Settings = []Setting{{TypeID: 3, Enabled: false, Remind: true}}
		res := Evaluate(in, at(today))
		assert.Equal(t, Summary{Total: 5, UpToDate: 4, Due: 1, Overdue: 0}, res.Summary)
		assert.Len(t, res.Items, 6)
	})
}

func TestNextDueAfter(t *testing.T) {
	assert.Equal(t, Due{From: d("2029-09-23"), By: d("2029-09-23")}, NextDueAfter(pap, d("2026-09-23"), cycle28))
	assert.Equal(t, Due{From: d("2027-09-23"), By: d("2028-09-23")}, NextDueAfter(mammo, d("2026-09-23"), Cycle{}))
	assert.Equal(t, Due{From: d("2026-10-24"), By: d("2026-10-27"), CycleTimed: true},
		NextDueAfter(selfExam, d("2026-09-23"), cycle28))
	// A cycle start before the anchor (negative offset) floors to the right cycle.
	assert.Equal(t, Due{From: d("2026-08-01"), By: d("2026-08-04"), CycleTimed: true},
		NextDueAfter(selfExam, d("2026-07-25"), cycle28))
	// interval 0 (bad admin input) is treated as monthly.
	assert.Equal(t, d("2026-10-23"), NextDueAfter(Type{}, d("2026-09-23"), Cycle{}).From)
}

func TestAddMonths(t *testing.T) {
	cases := []struct {
		from string
		n    int
		want string
	}{
		{"2026-01-31", 1, "2026-02-28"},
		{"2024-01-31", 1, "2024-02-29"},
		{"2026-03-31", 6, "2026-09-30"},
		{"2026-11-15", 3, "2027-02-15"},
		{"2026-01-15", -1, "2025-12-15"},
		{"2026-01-15", -13, "2024-12-15"},
		{"2024-02-29", 12, "2025-02-28"},
	}
	for _, c := range cases {
		assert.Equal(t, d(c.want), AddMonths(d(c.from), c.n), "%s %+d", c.from, c.n)
	}
}

func TestCycleFromHistory(t *testing.T) {
	confirmed := func(start string) model.History {
		return model.History{PeriodStart: d(start), PeriodEnd: d(start).AddDays(4), IsConfirmed: true, Source: "user_logged",
			BleedingLength: model.Int(5)}
	}
	t.Run("logged cycles give the current start and the median length", func(t *testing.T) {
		h := []model.History{confirmed("2026-07-26"), confirmed("2026-08-23"), confirmed("2026-09-20")}
		assert.Equal(t, Cycle{Start: d("2026-09-20"), Length: 28}, CycleFromHistory(h, nil, at(today)))
	})
	t.Run("profile only: last period and cycle duration", func(t *testing.T) {
		p := &model.Profile{LastPeriodStart: d("2026-09-10"), CycleDuration: model.Int(30)}
		c := CycleFromHistory(nil, p, at(today))
		assert.True(t, c.Known())
		assert.Equal(t, 30, c.Length)
		assert.Equal(t, d("2026-09-10"), c.Start)
	})
	t.Run("no data", func(t *testing.T) {
		c := CycleFromHistory(nil, nil, at(today))
		assert.False(t, c.Known())
		assert.Equal(t, Cycle{}, c)
	})
}
