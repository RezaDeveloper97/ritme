package ivf

import (
	"encoding/json"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/care"
	carestore "github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

var d = civildate.MustParse

func at(s string) time.Time {
	t, err := time.ParseInLocation(time.DateTime, s, civildate.Tehran)
	if err != nil {
		panic(err)
	}
	return t
}

func TestTimelineAndStageDay(t *testing.T) {
	c := Cycle{Stage: StageStim, StartedOn: d("2026-09-10"), StimStartedOn: d("2026-09-17"), Open: true}
	steps := c.Timeline()
	require.Len(t, steps, 6)
	assert.Equal(t, []string{StepDone, StepCurrent, StepTodo, StepTodo, StepTodo, StepTodo},
		[]string{steps[0].Status, steps[1].Status, steps[2].Status, steps[3].Status, steps[4].Status, steps[5].Status})
	assert.Equal(t, d("2026-09-10"), steps[0].Date)
	assert.Equal(t, d("2026-09-17"), steps[1].Date)
	assert.True(t, steps[2].Date.IsZero())

	day, ok := c.StageDay(d("2026-09-23"))
	assert.True(t, ok)
	assert.Equal(t, 7, day, "board «روز ۷ تحریک»")
	_, ok = c.StageDay(d("2026-09-16"))
	assert.False(t, ok, "stage not started yet")

	c.Stage, c.Open = StageTest, false
	for _, s := range c.Timeline() {
		assert.Equal(t, StepDone, s.Status, "a closed cycle has every stage up to its last one done")
	}
}

func TestTransferAndBeta(t *testing.T) {
	c := Cycle{Stage: StageTWW, StartedOn: d("2026-09-01"), TransferAt: at("2026-09-30 11:00:00"), BetaOn: d("2026-10-11"), Open: true}
	today := d("2026-10-02")
	since, ok := c.DaysSinceTransfer(today)
	assert.True(t, ok)
	assert.Equal(t, 2, since)
	toBeta, ok := c.DaysToBeta(today)
	assert.True(t, ok)
	assert.Equal(t, 9, toBeta, "board «۹ روز تا تست خون»")
	day, ok := c.StageDay(today)
	assert.True(t, ok)
	assert.Equal(t, 3, day)
}

func TestCheckOrder(t *testing.T) {
	base := Cycle{StartedOn: d("2026-09-10")}
	cases := map[string]struct {
		c     Cycle
		field string
		key   string
	}{
		"stim before start": {Cycle{StartedOn: base.StartedOn, StimStartedOn: d("2026-09-09")}, "stim_started_on", "before_cycle_start"},
		"transfer before retrieval": {Cycle{StartedOn: base.StartedOn, RetrievalAt: at("2026-09-25 08:00:00"),
			TransferAt: at("2026-09-24 08:00:00")}, "transfer_at", "transfer_before_retrieval"},
		"beta before transfer": {Cycle{StartedOn: base.StartedOn, TransferAt: at("2026-09-28 10:00:00"),
			BetaOn: d("2026-09-27")}, "beta_on", "beta_before_transfer"},
		"scan before start": {Cycle{StartedOn: base.StartedOn, NextScanAt: at("2026-09-01 09:00:00")}, "next_scan_at", "before_cycle_start"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var fe *FieldError
			require.ErrorAs(t, checkOrder(tc.c), &fe)
			assert.Equal(t, tc.field, fe.Field)
			assert.Equal(t, tc.key, fe.Key)
		})
	}
	ok := Cycle{StartedOn: base.StartedOn, StimStartedOn: d("2026-09-12"), RetrievalAt: at("2026-09-24 08:00:00"),
		TransferAt: at("2026-09-29 10:00:00"), BetaOn: d("2026-10-10")}
	assert.NoError(t, checkOrder(ok))
}

func TestPlan(t *testing.T) {
	c := Cycle{NextScanAt: at("2026-09-24 09:00:00"), TransferAt: at("2026-09-29 10:30:00"), BetaOn: d("2026-10-10")}
	specs := plan(c)
	require.Len(t, specs, 3)
	assert.Equal(t, KindScan, specs[0].Kind)
	assert.Equal(t, KindTransfer, specs[1].Kind)
	assert.Equal(t, KindBeta, specs[2].Kind)
	assert.Equal(t, at("2026-10-10 08:00:00"), specs[2].At, "beta at BetaHour")
	assert.Empty(t, plan(Cycle{}))
}

func TestNextSteps(t *testing.T) {
	assert.Equal(t, []string{NextPregnancySetup}, NextSteps(OutcomePositive))
	assert.Equal(t, []string{NextLoss, NextNewCycle}, NextSteps(OutcomeNegative))
	assert.Equal(t, []string{NextNewCycle}, NextSteps(OutcomeCancelled))
}

func TestRotate(t *testing.T) {
	codes := []string{"a_right", "a_left", "t_right", "t_left"}
	last, next := Rotate(codes, nil)
	assert.Nil(t, last)
	assert.Equal(t, "a_right", next, "never used → first in rotation order")

	uses := []SiteUse{{Site: "a_right", Date: d("2026-09-23"), Slot: "08:00"}}
	last, next = Rotate(codes, uses)
	require.NotNil(t, last)
	assert.Equal(t, "a_right", last.Site)
	assert.Equal(t, "a_left", next, "board: last abdomen right → suggest abdomen left")

	// every site used: the least recently used one; an inactive site is ignored
	uses = []SiteUse{
		{Site: "gone", Date: d("2026-09-24")},
		{Site: "t_left", Date: d("2026-09-23")},
		{Site: "a_right", Date: d("2026-09-22")},
		{Site: "a_left", Date: d("2026-09-21")},
		{Site: "t_right", Date: d("2026-09-20")},
		{Site: "t_left", Date: d("2026-09-19")},
	}
	last, next = Rotate(codes, uses)
	assert.Equal(t, "t_left", last.Site)
	assert.Equal(t, "t_right", next)

	_, next = Rotate(nil, uses)
	assert.Empty(t, next)
}

func medWith(times []string, starts, ends string, active bool, stock *Stock) Med {
	meta, _ := json.Marshal(care.MedicationMeta{V: 1, Form: "injection", Times: times, Weekdays: care.AllWeekdays, Amount: 1,
		Duration: care.DurationOngoing, Notify: true})
	r := carestore.Reminder{ID: 7, Type: care.TypeMedication, Title: "FSH", IsActive: active}
	r.Meta.V, r.Meta.Valid = meta, true
	r.StartsOn = civildate.NullDate{Date: d(starts), Valid: true}
	if ends != "" {
		r.EndsOn = civildate.NullDate{Date: d(ends), Valid: true}
	}
	return Med{ID: 3, Role: RoleStimulation, Route: RouteSubcutaneous, Stock: stock, Care: care.ParseMedication(r)}
}

func TestInventory(t *testing.T) {
	today := d("2026-09-23") // Wednesday
	stock := &Stock{Units: 3, Unit: "pen", DosesPerUnit: 1, CountedAt: at("2026-09-20 09:00:00")}
	m := medWith([]string{"20:00"}, "2026-09-17", "", true, stock)
	inv := m.Inventory(0, today)
	assert.Equal(t, 3, inv.DosesLeft)
	assert.Equal(t, 3, inv.UnitsLeft)
	assert.True(t, inv.Scheduled)
	assert.Equal(t, 3, inv.DaysLeft)
	assert.Equal(t, d("2026-09-26"), inv.RunsOutOn)
	assert.True(t, inv.Low, "board: 3 pens, enough till Friday → low")

	stock5 := &Stock{Units: 5, Unit: "prefilled_syringe", DosesPerUnit: 1}
	assert.False(t, medWith([]string{"08:00"}, "2026-09-17", "", true, stock5).Inventory(0, today).Low)

	// a pen lasting 3 doses, 2 doses used: 7 doses / 3 units left
	multi := &Stock{Units: 3, Unit: "pen", DosesPerUnit: 3}
	inv = medWith([]string{"08:00", "20:00"}, "2026-09-17", "", true, multi).Inventory(2, today)
	assert.Equal(t, 7, inv.DosesLeft)
	assert.Equal(t, 3, inv.UnitsLeft)
	assert.Equal(t, 3, inv.DaysLeft)

	// enough until the end date → not low even when few days are left
	inv = medWith([]string{"20:00"}, "2026-09-17", "2026-09-25", true, stock).Inventory(0, today)
	assert.False(t, inv.Low)
	// paused / not scheduled: days unknown, low only when empty
	inv = medWith([]string{"20:00"}, "2026-09-17", "", false, stock).Inventory(3, today)
	assert.False(t, inv.Scheduled)
	assert.False(t, inv.Low)
	// used more than counted never goes negative
	assert.Equal(t, 0, m.Inventory(10, today).DosesLeft)
	assert.Equal(t, Inventory{}, medWith([]string{"20:00"}, "2026-09-17", "", true, nil).Inventory(0, today))
}

func TestDosesOn(t *testing.T) {
	day := d("2026-09-23")
	fsh := medWith([]string{"20:00"}, "2026-09-17", "", true, nil)
	anta := medWith([]string{"08:00"}, "2026-09-21", "", true, nil)
	anta.ID, anta.Care.Row.ID = 4, 8
	later := medWith([]string{"09:00"}, "2026-09-30", "", true, nil)
	later.ID, later.Care.Row.ID = 5, 9
	taken := map[doseKey]bool{{8, day, "08:00"}: true}
	sites := map[doseKey]string{{8, day, "08:00"}: "abdomen_upper_right"}
	doses := DosesOn(day, []Med{fsh, anta, later}, taken, sites)
	require.Len(t, doses, 2)
	assert.Equal(t, "08:00", doses[0].Slot)
	assert.True(t, doses[0].Taken)
	assert.Equal(t, "abdomen_upper_right", doses[0].Site)
	assert.Equal(t, "20:00", doses[1].Slot)
	assert.False(t, doses[1].Taken)
}

func TestScanGrowth(t *testing.T) {
	s := Scan{Date: d("2026-09-23"), Right: Ovary{3, 4, 2, 0}, Left: Ovary{2, 5, 3, 1}}
	assert.Equal(t, 9, s.Right.Total())
	g := s.GrowthOf()
	assert.Equal(t, 9, g.Mid)
	assert.Equal(t, 6, g.Mature)
}

func TestValidateMed(t *testing.T) {
	now := at("2026-09-23 10:00:00")
	body := func(js string) phpval.Map {
		v, err := phpval.Decode([]byte(js))
		require.NoError(t, err)
		m, _ := v.(phpval.Map)
		return m
	}
	in, err := validateMed(body(`{"name":"FSH","role":"stimulation","route":"subcutaneous","dose":150,"unit":"iu","times":["20:00","08:00","20:00"],"stock_units":3,"stock_unit":"pen"}`), "en", now, "fa")
	require.NoError(t, err)
	assert.Equal(t, []string{"08:00", "20:00"}, in.Times)
	assert.Equal(t, "150", *in.Dose)
	assert.Equal(t, d("2026-09-23"), in.StartsOn)
	assert.Equal(t, 3, *in.StockUnits)
	assert.Equal(t, 1, in.DosesPerUnit)

	in, err = validateMed(body(`{"name":"hCG","role":"trigger","route":"subcutaneous","trigger_at":"2026-09-25 22:30"}`), "en", now, "fa")
	require.NoError(t, err)
	assert.Equal(t, []string{"22:30"}, in.Times)
	assert.Equal(t, d("2026-09-25"), in.StartsOn)
	assert.Equal(t, d("2026-09-25"), in.EndsOn)

	_, err = validateMed(body(`{"name":"FSH","role":"stimulation","route":"subcutaneous"}`), "en", now, "fa")
	require.Error(t, err, "times required for non-trigger medicines")
	_, err = validateMed(body(`{"name":"hCG","role":"trigger","route":"subcutaneous"}`), "en", now, "fa")
	require.Error(t, err, "trigger_at required for the trigger")
}

func TestValidateScan(t *testing.T) {
	v, err := phpval.Decode([]byte(`{"right":{"lt_10":3,"10_14":4,"15_17":2},"left":{"18_plus":1},"endometrium_mm":8.46,"e2":"1250"}`))
	require.NoError(t, err)
	m, _ := v.(phpval.Map)
	in, err := validateScan(m, "fa", at("2026-09-23 10:00:00"))
	require.NoError(t, err)
	assert.Equal(t, Ovary{3, 4, 2, 0}, in.Right)
	assert.Equal(t, Ovary{0, 0, 0, 1}, in.Left)
	assert.Equal(t, "8.5", in.EndometriumMM)
	assert.Equal(t, "1250.00", in.E2)
	assert.Equal(t, "pg_ml", in.E2Unit)

	v, _ = phpval.Decode([]byte(`{"right":{"lt_10":61}}`))
	m, _ = v.(phpval.Map)
	_, err = validateScan(m, "fa", at("2026-09-23 10:00:00"))
	require.Error(t, err)
}

// Every lang key exists in every language file; appointment titles and attribute names resolve.
func TestLangFilesMatch(t *testing.T) {
	keys := map[string][]string{}
	err := fs.WalkDir(langFS, "lang", func(path string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		raw, err := langFS.ReadFile(path)
		if err != nil {
			return err
		}
		v, err := phpval.Decode(raw)
		if err != nil {
			return err
		}
		ks, _ := phpval.Dot(v)
		keys[path] = ks
		return nil
	})
	require.NoError(t, err)
	require.Len(t, keys, 2)
	assert.ElementsMatch(t, keys["lang/fa/ivf.json"], keys["lang/en/ivf.json"])
	for _, k := range []string{KindScan, KindRetrieval, KindTransfer, KindBeta} {
		assert.NotEqual(t, "ivf.reminders."+k, T("reminders."+k, "fa"))
	}
	assert.Equal(t, "تخمدان راست، ۱۰ تا ۱۴", attributeName("right.10_14", "fa"))
	assert.Contains(t, fieldMessage("beta_on", "before_cycle_start", "en"), "beta test date")
}
