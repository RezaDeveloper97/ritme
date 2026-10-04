package children

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/children/store"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

var d = civildate.MustParse

func TestAddMonthsClampsToMonthEnd(t *testing.T) {
	assert.Equal(t, d("2026-02-28"), AddMonths(d("2026-01-31"), 1))
	assert.Equal(t, d("2028-02-29"), AddMonths(d("2028-01-31"), 1))
	assert.Equal(t, d("2027-06-20"), AddMonths(d("2026-06-20"), 12))
	assert.Equal(t, d("2025-12-15"), AddMonths(d("2026-01-15"), -1))
	assert.Equal(t, d("2032-06-20"), AddMonths(d("2026-06-20"), 72))
}

func TestAgeOnAndLabel(t *testing.T) {
	a := AgeOn(d("2026-06-20"), d("2026-10-02")) // artboard «۳ ماه و ۱۲ روز»
	assert.Equal(t, Age{Days: 104, Weeks: 14, Months: 3, Years: 0, DaysInMonth: 12}, a)
	assert.Equal(t, "۳ ماه و ۱۲ روز", a.Label("fa"))
	assert.Equal(t, "3 months and 12 days", a.Label("en"))

	assert.Equal(t, "۴ سال و ۲ ماه", AgeOn(d("2022-08-01"), d("2026-10-02")).Label("fa"))
	assert.Equal(t, "۲ سال", AgeOn(d("2024-10-02"), d("2026-10-02")).Label("fa"))
	assert.Equal(t, "۱۲ روز", AgeOn(d("2026-09-20"), d("2026-10-02")).Label("fa"))
	assert.Equal(t, "امروز به دنیا آمده", AgeOn(d("2026-10-02"), d("2026-10-02")).Label("fa"))
	assert.Equal(t, Age{}, AgeOn(d("2026-10-03"), d("2026-10-02")))
	// 31 Jan → 28 Feb is a full month.
	assert.Equal(t, 1, AgeOn(d("2026-01-31"), d("2026-02-28")).Months)
}

func TestVisitLabel(t *testing.T) {
	assert.Equal(t, "بدو تولد", VisitLabel(0, "fa"))
	assert.Equal(t, "۴ ماهگی", VisitLabel(4, "fa"))
	assert.Equal(t, "۱۸ ماهگی", VisitLabel(18, "fa"))
	assert.Equal(t, "۶ سالگی", VisitLabel(72, "fa"))
	assert.Equal(t, "At birth", VisitLabel(0, "en"))
}

func entry(code, visit string, months int) Entry {
	return Entry{Code: code, Visit: visit, AgeMonths: months, Title: json.RawMessage(`{"en":"` + code + `"}`)}
}

var testSchedule = []Entry{
	entry("bcg", "birth", 0), entry("hepb_1", "birth", 0),
	entry("penta_1", "m2", 2), entry("pcv_1", "m2", 2),
	entry("penta_2", "m4", 4), entry("pcv_2", "m4", 4),
	entry("mmr_1", "m12", 12),
}

func TestBuildScheduleStatuses(t *testing.T) {
	birth := d("2026-06-20")
	today := d("2026-10-03")
	given := []store.ChildVaccineDose{
		{DoseCode: "bcg", GivenOn: birth}, {DoseCode: "hepb_1", GivenOn: birth}, {DoseCode: "penta_1", GivenOn: d("2026-08-20")},
	}
	s := BuildSchedule(testSchedule, given, birth, today)
	require.Len(t, s.Visits, 4)
	assert.Equal(t, StatusDone, s.Visits[0].Status)
	assert.Equal(t, StatusOverdue, s.Visits[1].Status, "m2 due 2026-08-20, 44 days ago, one dose missing")
	assert.Equal(t, StatusSoon, s.Visits[2].Status, "m4 due 2026-10-20")
	assert.Equal(t, d("2026-10-20"), s.Visits[2].DueDate)
	assert.Equal(t, d("2026-10-17"), s.Visits[2].RemindOn)
	assert.Equal(t, StatusUpcoming, s.Visits[3].Status)
	assert.Equal(t, 3, s.GivenDoses)
	assert.Equal(t, 7, s.TotalDoses)
	assert.Equal(t, 1, s.CompletedVisits)
	assert.False(t, s.UpToDate)
	require.NotNil(t, s.Next)
	assert.Equal(t, "m2", s.Next.Code)
	assert.Equal(t, StatusDone, s.Visits[1].DoseStatus(s.Visits[1].Doses[0]))
	assert.Equal(t, StatusOverdue, s.Visits[1].DoseStatus(s.Visits[1].Doses[1]))

	// Due within the 30-day grace is «due», not overdue.
	s = BuildSchedule(testSchedule, nil, birth, d("2026-07-01"))
	assert.Equal(t, StatusDue, s.Visits[0].Status)
}

func TestDueRemindersAndPlan(t *testing.T) {
	birth := d("2026-06-20")
	all := []store.ChildVaccineDose{
		{DoseCode: "bcg"}, {DoseCode: "hepb_1"}, {DoseCode: "penta_1"}, {DoseCode: "pcv_1"},
	}
	// Before the reminder day: nothing.
	s := BuildSchedule(testSchedule, all, birth, d("2026-10-16"))
	assert.Empty(t, DueReminders(s, d("2026-10-16")))
	// On the reminder day (due − 3): the m4 visit, 3 days left.
	s = BuildSchedule(testSchedule, all, birth, d("2026-10-17"))
	r := DueReminders(s, d("2026-10-17"))
	require.Len(t, r, 1)
	assert.Equal(t, "m4", r[0].Visit.Code)
	assert.Equal(t, 3, r[0].Days)

	p := notifications.Defaults()
	occ := PlanReminders(p, 7, "آوا", s, d("2026-10-17"), "fa")
	require.Len(t, occ, 1)
	assert.Equal(t, notifications.Checkups, occ[0].Message.Category)
	assert.Contains(t, occ[0].Message.Title, "آوا")
	assert.Contains(t, occ[0].Message.Body, "۴ ماهگی")
	assert.Equal(t, 2026, occ[0].At.Year())
	assert.Empty(t, PlanReminders(p, 7, "آوا", s, d("2026-10-18"), "fa"), "fires once, on the reminder day")

	// A switched-off category is listed but not sent.
	p.Categories = map[notifications.Category]bool{notifications.Checkups: false}
	occ = PlanReminders(p, 7, "Ava", s, d("2026-10-17"), "en")
	require.Len(t, occ, 1)
	assert.False(t, occ[0].Decision.Send)
	assert.Equal(t, notifications.ReasonCategoryOff, occ[0].Decision.Reason)
}

func TestBandFor(t *testing.T) {
	bs := []int{2, 3, 4, 6, 9, 12}
	assert.Equal(t, 2, bandFor(bs, 0), "younger than every band: the first")
	assert.Equal(t, 3, bandFor(bs, 3))
	assert.Equal(t, 6, bandFor(bs, 8))
	assert.Equal(t, 12, bandFor(bs, 40))
	assert.Equal(t, -1, bandFor(nil, 4))
}

func TestLangFilesHaveTheSameKeys(t *testing.T) {
	for _, key := range []string{"messages.not_found", "age.months_days", "visits.months", "growth.disclaimer", "notes.vaccines", "reminder.body"} {
		for _, loc := range []string{"fa", "en"} {
			assert.NotEqual(t, "children."+key, translator().Trans("children."+key, nil, loc), "%s %s", loc, key)
		}
	}
}
