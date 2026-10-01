package conditions

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/conditions/store"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// PBAC pad chart (nbl_Cond_Bleed, the Higham pictorial blood-loss assessment chart). Pads per soak level are this
// package's rows; the clot chip is the log taxonomy's bleeding.clots / bleeding.clot_size (B-N3-01).
const (
	PointsLight     = 1  // «کمی خیس»
	PointsMedium    = 5  // «نیمه خیس»
	PointsHeavy     = 20 // «کاملاً خیس»
	PointsClotSmall = 1  // «کوچک (اندازه سکه)»
	PointsClotLarge = 5  // «بزرگ»
	PointsFlooding  = 5  // «نشت از نوار به لباس»

	// AlertScore: a period score at or above it shows condition_alerts pbac_over_100.
	AlertScore = 100
	// MaxPads bounds each soak level's count per day.
	MaxPads = 50
	// PeriodSpan is the longest period window the period score sums (days).
	PeriodSpan = 10
)

// Clot chips.
const (
	ClotNone  = "none"
	ClotSmall = "small"
	ClotLarge = "large"
)

// ClotChoices are the clot chip codes.
var ClotChoices = []string{ClotNone, ClotSmall, ClotLarge}

// PBACDay is one pad-chart day.
type PBACDay struct {
	Date                 civildate.Date
	Light, Medium, Heavy int
	Clots                *string
	Flooding             bool
}

// Score is the day's PBAC score.
func (d PBACDay) Score() int {
	s := d.Light*PointsLight + d.Medium*PointsMedium + d.Heavy*PointsHeavy
	if d.Clots != nil {
		switch *d.Clots {
		case ClotSmall:
			s += PointsClotSmall
		case ClotLarge:
			s += PointsClotLarge
		}
	}
	if d.Flooding {
		s += PointsFlooding
	}
	return s
}

// Logged reports whether the day has anything in the chart.
func (d PBACDay) Logged() bool {
	return d.Light > 0 || d.Medium > 0 || d.Heavy > 0 || d.Flooding || d.Clots != nil
}

// PBACPeriod is the period the day belongs to.
type PBACPeriod struct {
	Start, End civildate.Date
	DayNumber  int
	Score      int
	DaysLogged int
}

// PBACView is GET /conditions/pbac/{date}.
type PBACView struct {
	Day    PBACDay
	Period *PBACPeriod
	Alert  *catalog.Item
}

// clotsFromLog reads the clot chip from a day's log (nil when not logged).
func clotsFromLog(day []taxonomy.Entry) *string {
	var clots, size *string
	for _, e := range day {
		if e.Category != "bleeding" || !e.Code.Valid {
			continue
		}
		v := e.Code.String
		switch e.Param {
		case "clots":
			clots = &v
		case "clot_size":
			size = &v
		}
	}
	if clots == nil {
		return nil
	}
	out := ClotNone
	if *clots == taxonomy.Yes {
		out = ClotSmall // legacy sizes (none, medium) and a missing size count as small
		if size != nil && *size == ClotLarge {
			out = ClotLarge
		}
	}
	return &out
}

// PeriodStartFor finds the period a day belongs to: the latest cycle_histories start within PeriodSpan days before
// it (unless its recorded period_end_date + 1 day is past), else the first day of the run of consecutive logged
// days ending on it (at most PeriodSpan days). ok = false when the day is in no period (nothing logged, no
// recorded start).
func PeriodStartFor(starts []PeriodStart, date civildate.Date, logged func(civildate.Date) bool) (civildate.Date, bool) {
	if len(starts) > 0 {
		p := starts[0]
		ended := p.End != nil && date.After(p.End.AddDays(1)) // a recorded end (+1 day of grace) closes the period
		if n := p.Start.DiffDays(date); n >= 0 && n < PeriodSpan && !ended {
			return p.Start, true
		}
	}
	if !logged(date) {
		return civildate.Date{}, false
	}
	start := date
	for i := 1; i < PeriodSpan && logged(date.AddDays(-i)); i++ {
		start = date.AddDays(-i)
	}
	return start, true
}

func (s *Service) pbacDays(ctx context.Context, q *store.Queries, logs *healthlog.Service, userID uint64, from, to civildate.Date) (map[civildate.Date]PBACDay, error) {
	rows, err := q.ListPbacEntries(ctx, store.ListPbacEntriesParams{UserID: userID, FromDate: from, ToDate: to})
	if err != nil {
		return nil, fmt.Errorf("conditions: load pbac entries: %w", err)
	}
	logDays, err := logs.Range(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	out := map[civildate.Date]PBACDay{}
	for _, r := range rows {
		out[r.EntryDate] = PBACDay{Date: r.EntryDate, Light: int(r.LightCount), Medium: int(r.MediumCount), Heavy: int(r.HeavyCount), Flooding: r.Flooding}
	}
	for _, ld := range logDays {
		if c := clotsFromLog(ld.Entries); c != nil {
			d := out[ld.Date]
			d.Date, d.Clots = ld.Date, c
			out[ld.Date] = d
		}
	}
	return out, nil
}

// PBAC loads one day with its period and the alert.
func (s *Service) PBAC(ctx context.Context, userID uint64, date civildate.Date) (PBACView, error) {
	return s.pbacView(ctx, store.New(s.conn), healthlog.NewService(s.conn), userID, date)
}

func (s *Service) pbacView(ctx context.Context, q *store.Queries, logs *healthlog.Service, userID uint64, date civildate.Date) (PBACView, error) {
	from, to := date.AddDays(-(PeriodSpan - 1)), date.AddDays(PeriodSpan-1)
	days, err := s.pbacDays(ctx, q, logs, userID, from, to)
	if err != nil {
		return PBACView{}, err
	}
	starts, err := periodStarts(ctx, q, userID, date)
	if err != nil {
		return PBACView{}, err
	}
	v := PBACView{Day: PBACDay{Date: date}}
	if d, ok := days[date]; ok {
		v.Day = d
	}
	start, ok := PeriodStartFor(starts, date, func(d civildate.Date) bool { return days[d].Logged() })
	if !ok {
		return v, nil
	}
	p := &PBACPeriod{Start: start, End: start.AddDays(PeriodSpan - 1), DayNumber: start.DiffDays(date) + 1}
	for d := p.Start; !d.After(p.End); d = d.AddDays(1) {
		if day, ok := days[d]; ok && day.Logged() {
			p.Score += day.Score()
			p.DaysLogged++
		}
	}
	v.Period = p
	if p.Score >= AlertScore {
		if v.Alert, err = s.alert(ctx, "pbac_over_100"); err != nil {
			return PBACView{}, err
		}
	}
	return v, nil
}

// ClotOptions are the clot chips the user may newly log on date (none when her mode has no clot param and the day
// stores none).
func (s *Service) ClotOptions(ctx context.Context, userID uint64, date civildate.Date) ([]string, error) {
	logs := healthlog.NewService(s.conn)
	mode, err := logs.LifeMode(ctx, userID)
	if err != nil {
		return nil, err
	}
	day, err := logs.Day(ctx, userID, date)
	if err != nil {
		return nil, err
	}
	cat, _ := taxonomy.CategoryByCode("bleeding")
	clots, _ := cat.Param("clots")
	if cat.Available(clots, mode) || clotsFromLog(day) != nil {
		return ClotChoices, nil
	}
	return []string{}, nil
}

// PBACInput is a validated PUT /conditions/pbac/{date}: Set says which keys the body sent (a sent null clears).
type PBACInput struct {
	Date                 civildate.Date
	Set                  map[string]bool
	Light, Medium, Heavy *int
	Clots                *string
	Flooding             *bool
}

// SavePBAC merges the sent keys over the day and returns the day with its period. ErrNotEnrolled without the
// heavy-bleeding program.
func (s *Service) SavePBAC(ctx context.Context, userID uint64, in PBACInput, locale string, now time.Time) (PBACView, error) {
	var view PBACView
	err := s.inTx(ctx, func(q *store.Queries, logs *healthlog.Service) error {
		if err := requireEnrolled(ctx, q, userID, ProgramHeavyBleeding); err != nil {
			return err
		}
		cur, err := s.pbacView(ctx, q, logs, userID, in.Date)
		if err != nil {
			return err
		}
		d := cur.Day
		count := func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		}
		if in.Set["light"] {
			d.Light = count(in.Light)
		}
		if in.Set["medium"] {
			d.Medium = count(in.Medium)
		}
		if in.Set["heavy"] {
			d.Heavy = count(in.Heavy)
		}
		if in.Set["flooding"] {
			d.Flooding = in.Flooding != nil && *in.Flooding
		}
		if in.Set["clots"] {
			if err := saveClots(ctx, logs, userID, in.Date, in.Clots, locale, now); err != nil {
				return err
			}
		}
		if d.Light == 0 && d.Medium == 0 && d.Heavy == 0 && !d.Flooding {
			if err := q.DeletePbacEntry(ctx, store.DeletePbacEntryParams{UserID: userID, EntryDate: in.Date}); err != nil {
				return fmt.Errorf("conditions: delete pbac entry: %w", err)
			}
		} else if err := q.UpsertPbacEntry(ctx, store.UpsertPbacEntryParams{
			UserID: userID, EntryDate: in.Date,
			LightCount:  uint8(d.Light),  //nolint:gosec // G115: validated 0…MaxPads
			MediumCount: uint8(d.Medium), //nolint:gosec // G115: validated 0…MaxPads
			HeavyCount:  uint8(d.Heavy),  //nolint:gosec // G115: validated 0…MaxPads
			Flooding:    d.Flooding, Now: tehranNow(now),
		}); err != nil {
			return fmt.Errorf("conditions: save pbac entry: %w", err)
		}
		view, err = s.pbacView(ctx, q, logs, userID, in.Date)
		return err
	})
	return view, err
}

// saveClots writes the clot chip into the log: none → clots no; small/large → clots yes + size; nil clears both.
func saveClots(ctx context.Context, logs *healthlog.Service, userID uint64, date civildate.Date, chip *string, locale string, now time.Time) error {
	var clots, size []taxonomy.Entry
	if chip != nil {
		code := taxonomy.Yes
		if *chip == ClotNone {
			code = taxonomy.No
		} else {
			size = []taxonomy.Entry{{Category: "bleeding", Param: "clot_size", Code: sql.NullString{String: *chip, Valid: true}}}
		}
		clots = []taxonomy.Entry{{Category: "bleeding", Param: "clots", Code: sql.NullString{String: code, Valid: true}}}
	}
	_, err := logs.SaveDay(ctx, userID, date, []taxonomy.Change{
		{Category: "bleeding", Param: "clots", Entries: clots},
		{Category: "bleeding", Param: "clot_size", Entries: size},
	}, locale, now)
	return err
}
