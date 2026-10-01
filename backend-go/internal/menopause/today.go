package menopause

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/checkups"
	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Home ("today") rules (nbl_Meno_Home, Main).
const (
	// BleedingAlertDays: bleeding or spotting logged in the last N days (today included) while the stage is meno or
	// post raises the postmenopausal_bleeding alert [needs clinical review].
	BleedingAlertDays = 30
	// UpcomingCheckups is the home card's row count.
	UpcomingCheckups = 3
)

// SleepNight is the sleep bucket logged for last night.
type SleepNight struct {
	Date  civildate.Date
	Code  string
	Hours float64
}

// Bleeding is the postmenopausal bleeding flag of the home.
type Bleeding struct {
	Alert  bool
	LastOn *civildate.Date // the latest bleeding/spotting day of the window (any stage)
	Item   *catalog.Item   // meno_alerts postmenopausal_bleeding when Alert
}

// Today is GET /menopause/today.
type Today struct {
	Date        civildate.Date
	Stage       Stage
	StageTip    *catalog.Item
	Flashes     FlashDay
	NightSweats int
	NightLevel  string // symptoms.general.night_sweats level logged today ("" = none)
	Sleep       *SleepNight
	Scale       Scale
	Score       *ScoreEntry
	Trend       []TrendPoint
	Plan        *checkups.Plan
	Checkups    []engine.Item
	Treatment   []TreatmentToday
	Bleeding    Bleeding
}

// Today builds the home read model.
func (s *Service) Today(ctx context.Context, userID uint64, now time.Time) (Today, error) {
	today := civildate.InTehran(now)
	t := Today{Date: today}
	var err error
	if t.Stage, err = s.Profile(ctx, userID, today); err != nil {
		return Today{}, err
	}
	if t.StageTip, err = s.StageTip(ctx, t.Stage); err != nil {
		return Today{}, err
	}
	if t.Flashes, err = s.Day(ctx, userID, today, now); err != nil {
		return Today{}, err
	}
	logs, err := s.logDays(ctx, userID, today.AddDays(-(BleedingAlertDays - 1)), today)
	if err != nil {
		return Today{}, err
	}
	if err := s.nightAndSleep(ctx, userID, today, logs, &t); err != nil {
		return Today{}, err
	}
	if t.Bleeding, err = s.bleeding(ctx, t.Stage, logs); err != nil {
		return Today{}, err
	}
	if t.Scale, err = s.Scale(ctx); err != nil {
		return Today{}, err
	}
	h, err := s.History(ctx, userID, today, TrendMonths)
	if err != nil {
		return Today{}, err
	}
	t.Trend = h.Trend
	if t.Score, err = s.latestScore(ctx, userID, today, h); err != nil {
		return Today{}, err
	}
	if t.Plan, t.Checkups, err = s.upcomingCheckups(ctx, userID, now); err != nil {
		return Today{}, err
	}
	if t.Treatment, err = s.Treatment(ctx, userID, today); err != nil {
		return Today{}, err
	}
	return t, nil
}

// latestScore is the window's newest questionnaire, else the newest one ever (with its delta).
func (s *Service) latestScore(ctx context.Context, userID uint64, today civildate.Date, h History) (*ScoreEntry, error) {
	if e := h.Latest(); e != nil {
		return e, nil
	}
	sc, err := s.PreviousScore(ctx, userID, engine.JalaliMonthEnd(today).AddDays(1))
	if err != nil || sc == nil {
		return nil, err
	}
	prev, err := s.PreviousScore(ctx, userID, sc.Month)
	if err != nil {
		return nil, err
	}
	return &ScoreEntry{Score: *sc, Previous: prev}, nil
}

// nightAndSleep fills last night's sweats (sweaty flashes of the night, else 1 when night sweats are logged today)
// and sleep (today's log, else yesterday's).
func (s *Service) nightAndSleep(ctx context.Context, userID uint64, today civildate.Date, logs map[civildate.Date]LogDay, t *Today) error {
	flashes, err := s.Flashes(ctx, userID, today.AddDays(-1), today)
	if err != nil {
		return err
	}
	from := today.AddDays(-1).TehranMidnight().Add(NightFromHour * time.Hour)
	to := today.TehranMidnight().Add(NightToHour * time.Hour)
	for _, f := range flashes {
		if f.Sweat && !f.StartedAt.Before(from) && f.StartedAt.Before(to) {
			t.NightSweats++
		}
	}
	t.NightLevel = logs[today].Level(SlotNightSweats)
	if t.NightSweats == 0 && logs[today].Has(SlotNightSweats) {
		t.NightSweats = 1
	}
	for _, d := range []civildate.Date{today, today.AddDays(-1)} {
		if v := logs[d].View; v != nil {
			if h, ok := v.SleepHours(); ok {
				t.Sleep = &SleepNight{Date: d, Code: v.Sleep, Hours: h}
				break
			}
		}
	}
	return nil
}

// bleeding is the latest bleeding day of the window and, for stage meno / post, the alert.
func (s *Service) bleeding(ctx context.Context, st Stage, logs map[civildate.Date]LogDay) (Bleeding, error) {
	var b Bleeding
	for d, l := range logs {
		if l.Bleeding() && (b.LastOn == nil || d.After(*b.LastOn)) {
			day := d
			b.LastOn = &day
		}
	}
	if b.LastOn == nil || !st.PostMenopausal() {
		return b, nil
	}
	it, err := s.item(ctx, GroupAlerts, AlertPostmenopausalBleeding)
	if err != nil {
		return Bleeding{}, err
	}
	b.Alert, b.Item = true, it
	return b, nil
}

// checkupKeys are the checkup_types keys of the catalog meno_checkup_groups (meta.checkups), in group order.
func (s *Service) checkupKeys(ctx context.Context) ([]string, error) {
	groups, err := s.catalog.Items(ctx, GroupCheckupGroups)
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, g := range groups {
		var meta struct {
			Checkups []string `json:"checkups"`
		}
		_ = json.Unmarshal(g.Meta, &meta)
		keys = append(keys, meta.Checkups...)
	}
	return keys, nil
}

func statusRank(st engine.Status) int {
	switch st {
	case engine.StatusOverdue:
		return 0
	case engine.StatusDue:
		return 1
	case engine.StatusSoon:
		return 2
	}
	return 3
}

// upcomingCheckups are up to UpcomingCheckups items of her M4 plan — the menopause groups' checkups (and her own
// custom ones) that are overdue, due, soon or never done — overdue first, then by due date.
func (s *Service) upcomingCheckups(ctx context.Context, userID uint64, now time.Time) (*checkups.Plan, []engine.Item, error) {
	keys, err := s.checkupKeys(ctx)
	if err != nil {
		return nil, nil, err
	}
	p, err := checkups.LoadPlan(ctx, s.db, userID, clock.At(now))
	if err != nil {
		return nil, nil, err
	}
	var out []engine.Item
	for _, it := range p.Result.Items {
		if !it.Enabled || it.Status == engine.StatusDisabled || it.Status == engine.StatusNotYet {
			continue
		}
		if len(keys) > 0 && !it.IsCustom && !slices.Contains(keys, it.Key) {
			continue
		}
		if statusRank(it.Status) == 3 && !it.LastDoneOn.IsZero() {
			continue // up to date
		}
		out = append(out, it)
	}
	slices.SortStableFunc(out, func(a, b engine.Item) int {
		if c := cmp.Compare(statusRank(a.Status), statusRank(b.Status)); c != 0 {
			return c
		}
		switch {
		case a.NextDueOn.IsZero() && b.NextDueOn.IsZero():
			return 0
		case a.NextDueOn.IsZero():
			return 1
		case b.NextDueOn.IsZero():
			return -1
		}
		return a.NextDueOn.Compare(b.NextDueOn)
	})
	return p, out[:min(len(out), UpcomingCheckups)], nil
}
