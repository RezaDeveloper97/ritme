package projections

import (
	"context"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	carestore "github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/family"
	healthlogstore "github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/vitals"
)

var now = time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)

// fakeStates serves one prepared engine state.
type fakeStates struct{ st *engine.State }

func (f fakeStates) ForUser(context.Context, uint64, civildate.Date) (*engine.State, error) {
	return f.st, nil
}

func day(s string) engine.Day { return engine.DayOf(civildate.MustParse(s)) }

var history29 = []engine.Period{
	{Start: day("2026-06-01"), End: day("2026-06-05")}, {Start: day("2026-06-30"), End: day("2026-07-04")},
	{Start: day("2026-07-29"), End: day("2026-08-02")}, {Start: day("2026-08-27"), End: day("2026-08-31")},
}

func v6Facts(t *testing.T, in engine.Input, today string) CycleFacts {
	t.Helper()
	f, err := v6Cycle{states: fakeStates{st: engine.Prepare(in, engine.DefaultParams())}}.Facts(context.Background(), 1, civildate.MustParse(today))
	require.NoError(t, err)
	return f
}

// snap is a live link of role with the given grants, stamped with the owner's epoch 1, owner in mode.
func snap(role family.Role, mode enums.LifeMode, grants map[family.Domain]family.Level) family.Snapshot {
	s := family.Snapshot{OwnerID: 1, ActorID: 2, HasLink: true, LinkID: 9, Role: role, OwnerMode: family.ModeOf(string(mode)), OwnerEpoch: 1}
	for d, l := range grants {
		s.Grants[d] = family.Grant{Level: l, Epoch: 1, Stamped: true}
	}
	return s
}

func jsonKeys(t *testing.T, v any) []string {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Every DTO is an allow-list: its JSON keys are exactly the declared fields — nothing else can be serialised.
func TestDTOs_AreExactAllowLists(t *testing.T) {
	want := map[reflect.Type][]string{
		reflect.TypeOf(CyclePhase{}):         {"main_phase", "period_soon"},
		reflect.TypeOf(TTCFertility{}):       {"best_days", "fertile_end", "fertile_start", "ovulation_date"},
		reflect.TypeOf(Pregnancy{}):          {"current_week", "edd", "trimester"},
		reflect.TypeOf(FetalActivity{}):      {"kicks_today", "last_contraction_session"},
		reflect.TypeOf(KicksToday{}):         {"date", "sessions", "status"},
		reflect.TypeOf(ContractionSummary{}): {"count", "five_one_one_met", "in_progress", "last_at"},
		reflect.TypeOf(Appointment{}):        {"id", "location", "scheduled_at", "status", "title", "with"},
		reflect.TypeOf(Med{}):                {"dose", "form", "id", "times", "title", "today", "unit", "unit_label"},
		reflect.TypeOf(Vitals{}):             {"bp", "glucose", "weight"},
		reflect.TypeOf(Reading{}):            {"category", "date", "unit", "value"},
		reflect.TypeOf(BP{}):                 {"category", "date", "diastolic", "systolic"},
	}
	for typ, keys := range want {
		var got []string
		for i := range typ.NumField() {
			f := typ.Field(i)
			require.True(t, f.IsExported(), "%s.%s", typ.Name(), f.Name)
			tag := strings.Split(f.Tag.Get("json"), ",")[0]
			require.NotEmpty(t, tag, "%s.%s has an explicit json name", typ.Name(), f.Name)
			got = append(got, tag)
		}
		sort.Strings(got)
		assert.Equal(t, keys, got, typ.Name())
	}
}

func TestCyclePhase_NoNumbers(t *testing.T) {
	f := v6Facts(t, engine.Input{Periods: history29}, "2026-09-05")
	require.NotNil(t, f.DaysToPeriod, "the facts carry numbers …")
	for _, role := range []family.Role{family.RolePartner, family.RoleFriend} {
		dto := CyclePhaseOf(f, snap(role, enums.LifeModeCycle, map[family.Domain]family.Level{family.DomainCyclePhase: family.LevelView}), now)
		keys := jsonKeys(t, dto)
		assert.Subset(t, []string{"main_phase", "period_soon"}, keys, "… the DTO does not")
		raw, _ := json.Marshal(dto)
		assert.NotContains(t, string(raw), "2026", "no date")
		assert.NotRegexp(t, `:\s*\d`, string(raw), "no number")
	}
	// Late: no period_soon, never a lateness hint.
	late := v6Facts(t, engine.Input{Periods: history29}, "2026-10-01")
	assert.True(t, late.Late)
	dto := CyclePhaseOf(late, snap(family.RolePartner, enums.LifeModeCycle, nil), now)
	assert.False(t, *dto.PeriodSoon)
	assert.Nil(t, CyclePhaseOf(CycleFacts{}, snap(family.RoleFriend, enums.LifeModeCycle, nil), now).MainPhase, "no data → null")
}

func TestCyclePhase_PeriodSoonPartnerOnly(t *testing.T) {
	soon := v6Facts(t, engine.Input{Periods: history29}, "2026-09-23") // next start 2026-09-25
	far := v6Facts(t, engine.Input{Periods: history29}, "2026-09-05")
	p := snap(family.RolePartner, enums.LifeModeCycle, nil)
	assert.True(t, *CyclePhaseOf(soon, p, now).PeriodSoon)
	assert.False(t, *CyclePhaseOf(far, p, now).PeriodSoon)
	for _, r := range []family.Role{family.RoleFriend, family.RoleCarer, family.RoleOtherParent} {
		assert.Nil(t, CyclePhaseOf(soon, snap(r, enums.LifeModeCycle, nil), now).PeriodSoon, r.String())
	}
}

func TestFertileLabel_HiddenUnderHormonalContraception(t *testing.T) {
	s := snap(family.RolePartner, enums.LifeModeCycle, map[family.Domain]family.Level{family.DomainCyclePhase: family.LevelView})
	in := engine.Input{Periods: history29}
	assert.Equal(t, "fertile", *CyclePhaseOf(v6Facts(t, in, "2026-09-09"), s, now).MainPhase, "cycle mode, no pill: fertile is allowed (F-02)")
	in.Hormonal = true
	for d := day("2026-09-01"); d < day("2026-09-26"); d++ {
		f := v6Facts(t, in, d.String())
		if p := CyclePhaseOf(f, s, now).MainPhase; p != nil {
			assert.NotEqual(t, "fertile", *p, d.String())
		}
		ttc := TTCOf(f)
		assert.Nil(t, ttc.FertileStart, d.String())
		assert.Nil(t, ttc.OvulationDate, d.String())
		assert.Empty(t, ttc.BestDays, d.String())
	}
	// A source that cannot tell suppression never exposes fertile.
	assert.Equal(t, "follicular", *CyclePhaseOf(CycleFacts{HasData: true, Phase: enums.MainPhaseFertile}, s, now).MainPhase)
}

func TestTTCSwitch_PartnerOnly(t *testing.T) {
	b := &Builder{cycle: v6Cycle{states: fakeStates{st: engine.Prepare(engine.Input{Periods: history29, Mode: engine.ModeTTC}, engine.DefaultParams())}}}
	at := time.Date(2026, 9, 9, 10, 0, 0, 0, civildate.Tehran)
	both := map[family.Domain]family.Level{family.DomainCyclePhase: family.LevelView, family.DomainTTCFertility: family.LevelView}

	partner := snap(family.RolePartner, enums.LifeModeTTC, both)
	v, err := b.Project(context.Background(), partner, family.DomainTTCFertility, "en", at)
	require.NoError(t, err)
	ttc := v.(TTCFertility)
	require.NotNil(t, ttc.FertileStart)
	require.NotNil(t, ttc.OvulationDate)
	assert.NotEmpty(t, ttc.BestDays)
	v, err = b.Project(context.Background(), partner, family.DomainCyclePhase, "en", at)
	require.NoError(t, err)
	assert.Equal(t, "fertile", *v.(CyclePhase).MainPhase, "the switch on: the fertile label shows in TTC mode")

	// The switch off (default) → no TTC data, and the fertile label reads follicular.
	off := snap(family.RolePartner, enums.LifeModeTTC, map[family.Domain]family.Level{family.DomainCyclePhase: family.LevelView})
	_, err = b.Project(context.Background(), off, family.DomainTTCFertility, "en", at)
	require.ErrorIs(t, err, ErrDenied)
	v, err = b.Project(context.Background(), off, family.DomainCyclePhase, "en", at)
	require.NoError(t, err)
	assert.Equal(t, "follicular", *v.(CyclePhase).MainPhase)

	// Any other role, even holding the row: denied. Out of TTC mode: denied.
	friend := snap(family.RoleFriend, enums.LifeModeTTC, both)
	_, err = b.Project(context.Background(), friend, family.DomainTTCFertility, "en", at)
	require.ErrorIs(t, err, ErrDenied)
	v, err = b.Project(context.Background(), friend, family.DomainCyclePhase, "en", at)
	require.NoError(t, err)
	assert.Equal(t, "follicular", *v.(CyclePhase).MainPhase)
	_, err = b.Project(context.Background(), snap(family.RolePartner, enums.LifeModeCycle, both), family.DomainTTCFertility, "en", at)
	require.ErrorIs(t, err, ErrDenied)
}

// A paused (stale epoch), revoked, expired or blocked grant returns nothing — Project decides before loading.
func TestProject_PausedOrRevokedReturnsNothing(t *testing.T) {
	b := &Builder{} // no sources: any load would panic, so a denial must come first
	grants := map[family.Domain]family.Level{}
	for _, d := range Domains {
		grants[d] = family.LevelView
	}
	base := snap(family.RolePartner, enums.LifeModeCycle, grants)
	paused := base
	paused.OwnerEpoch = 2
	revoked := base
	revoked.HasLink = false
	expired := base
	expired.ExpiresAt = now.Add(-time.Minute)
	blocked := base
	blocked.OwnerBlocked = true
	for name, s := range map[string]family.Snapshot{"paused": paused, "revoked": revoked, "expired": expired, "blocked": blocked} {
		for _, d := range Domains {
			v, err := b.Project(context.Background(), s, d, "en", now)
			require.ErrorIs(t, err, ErrDenied, "%s %s", name, d)
			assert.Nil(t, v)
		}
	}
	_, err := b.Project(context.Background(), base, family.NeverSymptoms, "en", now)
	require.ErrorIs(t, err, ErrUnknownDomain)
}

// fakeCare serves medication rows (other methods are not used).
type fakeCare struct {
	carestore.Querier
	meds    []carestore.Reminder
	intakes []carestore.ListIntakesOnDateRow
	asked   bool
}

func (f *fakeCare) ListActiveMedications(context.Context, uint64) ([]carestore.Reminder, error) {
	return f.meds, nil
}
func (f *fakeCare) ListIntakesOnDate(context.Context, carestore.ListIntakesOnDateParams) ([]carestore.ListIntakesOnDateRow, error) {
	f.asked = true
	return f.intakes, nil
}

func TestMeds_DoseStatusOnlyAtNudge(t *testing.T) {
	med := carestore.Reminder{ID: 5, Type: "medication", Title: "Iron", IsActive: true}
	med.Meta.Valid = true
	med.Meta.V = []byte(`{"v":1,"dose":"two pills","unit":"tablet","form":"tablet","times":["08:00","08:30","09:30"],"weekdays":[0,1,2,3,4,5,6],"amount":1,"duration":"ongoing","notify":true}`)
	care := &fakeCare{meds: []carestore.Reminder{med}, intakes: []carestore.ListIntakesOnDateRow{{ReminderID: 5, Slot: "08:00"}}}
	b := &Builder{care: care}

	v, err := b.Project(context.Background(), snap(family.RolePartner, enums.LifeModeCycle,
		map[family.Domain]family.Level{family.DomainMeds: family.LevelView}), family.DomainMeds, "en", now)
	require.NoError(t, err)
	list := v.([]Med)
	require.Len(t, list, 1)
	assert.Nil(t, list[0].Today)
	assert.False(t, care.asked, "intakes are not even read below nudge")
	assert.Nil(t, list[0].Dose, "free-text dose dropped")
	assert.NotContains(t, jsonKeys(t, list[0]), "today")

	for _, l := range []family.Level{family.LevelNudge, family.LevelWrite} {
		v, err = b.Project(context.Background(), snap(family.RolePartner, enums.LifeModeCycle,
			map[family.Domain]family.Level{family.DomainMeds: l}), family.DomainMeds, "en", now)
		require.NoError(t, err)
		list = v.([]Med)
		require.NotNil(t, list[0].Today, l.String())
		assert.Equal(t, []DoseStatus{{"08:00", DoseTaken}, {"08:30", DoseMissed}, {"09:30", DosePending}}, *list[0].Today)
	}
}

func TestVitalsProjection_LatestOnly(t *testing.T) {
	latest := map[string]*vitals.Reading{
		vitals.TypeBP:      {Type: vitals.TypeBP, Date: civildate.MustParse("2026-09-21"), Systolic: 118, Diastolic: 76, Note: "SENTINEL", Arm: "left", MeasuredAt: now},
		vitals.TypeGlucose: {Type: vitals.TypeGlucose, Date: civildate.MustParse("2026-09-20"), MgDl: 90, Unit: vitals.UnitMmolL, Context: "fasting"},
		vitals.TypeHR:      {Type: vitals.TypeHR, Date: civildate.MustParse("2026-09-20"), Pulse: 70},
	}
	num := func(s string) healthlogstore.HealthLogEntry {
		e := healthlogstore.HealthLogEntry{Category: "measurements", Param: "weight"}
		e.ValueNum.Valid, e.ValueNum.String = true, s
		return e
	}
	w1, w2 := num("61.00"), num("60.40")
	w1.LogDate, w2.LogDate = civildate.MustParse("2026-09-01"), civildate.MustParse("2026-09-19")
	out := VitalsOf(latest, []healthlogstore.HealthLogEntry{w1, w2})
	raw, err := json.Marshal(out)
	require.NoError(t, err)
	assert.JSONEq(t, `{"weight":{"value":60.4,"unit":"kg","category":null,"date":"2026-09-19"},
		"bp":{"systolic":118,"diastolic":76,"category":`+mustJSON(t, out.BP.Category)+`,"date":"2026-09-21"},
		"glucose":{"value":5,"unit":"mmol_l","category":`+mustJSON(t, out.Glucose.Category)+`,"date":"2026-09-20"}}`, string(raw))
	assert.JSONEq(t, `{"weight":null,"bp":null,"glucose":null}`, mustJSON(t, VitalsOf(map[string]*vitals.Reading{}, nil)))
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return string(raw)
}

func TestFetalActivity_PregnancyOnly(t *testing.T) {
	b := &Builder{}
	g := map[family.Domain]family.Level{family.DomainFetalActivity: family.LevelView}
	for _, m := range []enums.LifeMode{enums.LifeModeCycle, enums.LifeModeTTC, enums.LifeModePostpartum, enums.LifeModeMenopause, enums.LifeModeTeen} {
		_, err := b.Project(context.Background(), snap(family.RolePartner, m, g), family.DomainFetalActivity, "en", now)
		require.ErrorIs(t, err, ErrDenied, string(m))
	}
	_, err := b.Project(context.Background(), snap(family.RoleCarer, enums.LifeModePregnancy, g), family.DomainFetalActivity, "en", now)
	require.ErrorIs(t, err, ErrDenied, "partner only")
}

func TestPregnancyProjection_ModeAndEpochOnly(t *testing.T) {
	b := &Builder{}
	g := map[family.Domain]family.Level{family.DomainPregnancy: family.LevelView}
	for _, m := range []enums.LifeMode{enums.LifeModeCycle, enums.LifeModeTTC, enums.LifeModePostpartum, enums.LifeModeMenopause} {
		_, err := b.Project(context.Background(), snap(family.RolePartner, m, g), family.DomainPregnancy, "en", now)
		require.ErrorIs(t, err, ErrDenied, string(m))
	}
	stale := snap(family.RolePartner, enums.LifeModePregnancy, g)
	stale.OwnerEpoch = 3
	_, err := b.Project(context.Background(), stale, family.DomainPregnancy, "en", now)
	require.ErrorIs(t, err, ErrDenied, "a grant of an earlier pregnancy never opens this one")
}
