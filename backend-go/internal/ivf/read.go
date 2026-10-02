package ivf

import (
	"context"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/ivf/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// loadMeds is the cycle's medicines in the order they were added.
func loadMeds(ctx context.Context, q *store.Queries, userID, cycleID uint64) ([]Med, error) {
	rows, err := q.ListCycleMeds(ctx, store.ListCycleMedsParams{UserID: userID, CycleID: cycleID})
	if err != nil {
		return nil, fmt.Errorf("ivf: load medicines: %w", err)
	}
	out := make([]Med, 0, len(rows))
	for _, r := range rows {
		out = append(out, medFrom(r.IvfMed, r.Reminder))
	}
	return out, nil
}

// doseState is which doses were taken (care intakes) and where they were injected (IVF dose logs) over days.
type doseState struct {
	taken map[doseKey]bool
	sites map[doseKey]string
}

func loadDoseState(ctx context.Context, q *store.Queries, userID uint64, from, to civildate.Date, meds []Med) (doseState, error) {
	st := doseState{taken: map[doseKey]bool{}, sites: map[doseKey]string{}}
	intakes, err := q.ListIntakesBetween(ctx, store.ListIntakesBetweenParams{UserID: userID, FromDate: from, ToDate: to})
	if err != nil {
		return st, fmt.Errorf("ivf: load intakes: %w", err)
	}
	for _, in := range intakes {
		st.taken[doseKey{in.ReminderID, in.IntakeDate, in.Slot}] = true
	}
	logs, err := q.ListDoseLogsBetween(ctx, store.ListDoseLogsBetweenParams{UserID: userID, FromDate: from, ToDate: to})
	if err != nil {
		return st, fmt.Errorf("ivf: load dose logs: %w", err)
	}
	reminderOf := map[uint64]uint64{}
	for _, m := range meds {
		reminderOf[m.ID] = m.ReminderID()
	}
	for _, l := range logs {
		if rid, ok := reminderOf[l.IvfMedID]; ok && l.Site.Valid {
			st.sites[doseKey{rid, l.DoseDate, l.Slot}] = l.Site.String
		}
	}
	return st, nil
}

// HomeView is GET /ivf (nbl_IVF_Home).
type HomeView struct {
	Today           civildate.Date
	Now             time.Time
	Enabled         bool
	Cycle           *Cycle
	CyclesCount     int
	Doses           []Dose
	NextAppointment *LinkedAppointment
	LatestScan      *Scan
	Companion       Companion
}

// Home loads the IVF home as of now.
func (s *Service) Home(ctx context.Context, userID uint64, now time.Time) (HomeView, error) {
	q := store.New(s.conn)
	today := civildate.InTehran(now)
	v := HomeView{Today: today, Now: now, Doses: []Dose{}}
	var err error
	if v.Enabled, err = q.GetIVFSwitch(ctx, userID); err != nil {
		return v, fmt.Errorf("ivf: load switch: %w", err)
	}
	st, err := q.CycleStats(ctx, userID)
	if err != nil {
		return v, fmt.Errorf("ivf: cycle stats: %w", err)
	}
	v.CyclesCount = int(st.Cycles)
	if v.Companion, err = s.companion(ctx, userID); err != nil {
		return v, err
	}
	if v.Cycle, err = activeCycle(ctx, q, userID, false); err != nil || v.Cycle == nil {
		return v, err
	}
	meds, err := loadMeds(ctx, q, userID, v.Cycle.ID)
	if err != nil {
		return v, err
	}
	ds, err := loadDoseState(ctx, q, userID, today, today, meds)
	if err != nil {
		return v, err
	}
	v.Doses = DosesOn(today, meds, ds.taken, ds.sites)
	links, err := linkedAppointments(ctx, q, userID, v.Cycle.ID)
	if err != nil {
		return v, err
	}
	v.NextAppointment = NextAppointment(links, now)
	scans, err := q.ListScans(ctx, store.ListScansParams{UserID: userID, CycleID: v.Cycle.ID})
	if err != nil {
		return v, fmt.Errorf("ivf: load scans: %w", err)
	}
	if n := len(scans); n > 0 {
		last := scanFromRow(scans[n-1])
		v.LatestScan = &last
	}
	return v, nil
}

// MedLine is a medicine with its inventory today.
type MedLine struct {
	Med       Med
	Inventory Inventory
}

// TriggerCard is the trigger shot with whether it was taken.
type TriggerCard struct {
	Med   Med
	Taken bool
}

// MedsView is GET /ivf/meds (nbl_IVF_Meds).
type MedsView struct {
	Today     civildate.Date
	Cycle     *Cycle
	Trigger   *TriggerCard
	TodayDose []Dose
	Tomorrow  []Dose
	Meds      []MedLine
	Sites     []string
	LastSite  *SiteUse
	NextSite  string
}

// Meds loads the injection schedule of the open cycle as of now: trigger card, today's and tomorrow's doses,
// inventory per medicine and the site rotation (sites are the user's, across cycles).
func (s *Service) Meds(ctx context.Context, userID uint64, now time.Time) (MedsView, error) {
	q := store.New(s.conn)
	today := civildate.InTehran(now)
	v := MedsView{Today: today, TodayDose: []Dose{}, Tomorrow: []Dose{}, Meds: []MedLine{}}
	var err error
	if v.Sites, err = s.codes(ctx, GroupSites); err != nil {
		return v, err
	}
	recent, err := q.RecentSites(ctx, userID)
	if err != nil {
		return v, fmt.Errorf("ivf: load sites: %w", err)
	}
	uses := make([]SiteUse, 0, len(recent))
	for _, r := range recent {
		uses = append(uses, SiteUse{Site: r.Site.String, Date: r.DoseDate, Slot: r.Slot})
	}
	v.LastSite, v.NextSite = Rotate(v.Sites, uses)

	if v.Cycle, err = activeCycle(ctx, q, userID, false); err != nil || v.Cycle == nil {
		return v, err
	}
	meds, err := loadMeds(ctx, q, userID, v.Cycle.ID)
	if err != nil {
		return v, err
	}
	from, to := today, today.AddDays(1)
	var trigger *Med
	for i := range meds {
		if meds[i].IsTrigger() && !meds[i].TriggerAt.IsZero() {
			trigger = &meds[i] // the latest one added wins
		}
	}
	if trigger != nil {
		if d := dayOf(trigger.TriggerAt); d.Before(from) {
			from = d
		} else if d.After(to) {
			to = d
		}
	}
	ds, err := loadDoseState(ctx, q, userID, from, to, meds)
	if err != nil {
		return v, err
	}
	v.TodayDose = DosesOn(today, meds, ds.taken, ds.sites)
	v.Tomorrow = DosesOn(today.AddDays(1), meds, ds.taken, ds.sites)
	if trigger != nil {
		slot := trigger.TriggerAt.Format("15:04")
		v.Trigger = &TriggerCard{Med: *trigger, Taken: ds.taken[doseKey{trigger.ReminderID(), dayOf(trigger.TriggerAt), slot}]}
	}
	for _, m := range meds {
		line := MedLine{Med: m}
		if m.Stock != nil {
			used, err := q.CountIntakesSince(ctx, store.CountIntakesSinceParams{
				UserID: userID, ReminderID: m.ReminderID(), Since: m.Stock.CountedAt,
			})
			if err != nil {
				return v, fmt.Errorf("ivf: count doses: %w", err)
			}
			line.Inventory = m.Inventory(int(used), today)
		}
		v.Meds = append(v.Meds, line)
	}
	return v, nil
}

// ScansView is GET /ivf/scans (nbl_IVF_Scan).
type ScansView struct {
	Today civildate.Date
	Cycle *Cycle
	Scans []Scan
}

// Scans loads the open cycle's scans, oldest first.
func (s *Service) Scans(ctx context.Context, userID uint64, now time.Time) (ScansView, error) {
	q := store.New(s.conn)
	v := ScansView{Today: civildate.InTehran(now), Scans: []Scan{}}
	var err error
	if v.Cycle, err = activeCycle(ctx, q, userID, false); err != nil || v.Cycle == nil {
		return v, err
	}
	rows, err := q.ListScans(ctx, store.ListScansParams{UserID: userID, CycleID: v.Cycle.ID})
	if err != nil {
		return v, fmt.Errorf("ivf: load scans: %w", err)
	}
	for _, r := range rows {
		v.Scans = append(v.Scans, scanFromRow(r))
	}
	return v, nil
}

// MoodLog is one two-week-wait check-in.
type MoodLog struct {
	Date civildate.Date
	Mood string
}

// TWWView is GET /ivf/tww (nbl_IVF_TWW).
type TWWView struct {
	Today  civildate.Date
	Cycle  *Cycle
	Moods  []MoodLog
	Luteal []Dose // today's luteal-support doses
}

// TWW loads the two-week wait of the open cycle as of now.
func (s *Service) TWW(ctx context.Context, userID uint64, now time.Time) (TWWView, error) {
	q := store.New(s.conn)
	today := civildate.InTehran(now)
	v := TWWView{Today: today, Moods: []MoodLog{}, Luteal: []Dose{}}
	var err error
	if v.Cycle, err = activeCycle(ctx, q, userID, false); err != nil || v.Cycle == nil {
		return v, err
	}
	logs, err := q.ListTWWLogs(ctx, store.ListTWWLogsParams{UserID: userID, CycleID: v.Cycle.ID})
	if err != nil {
		return v, fmt.Errorf("ivf: load moods: %w", err)
	}
	for _, l := range logs {
		v.Moods = append(v.Moods, MoodLog{Date: l.LogDate, Mood: l.Mood})
	}
	meds, err := loadMeds(ctx, q, userID, v.Cycle.ID)
	if err != nil {
		return v, err
	}
	var luteal []Med
	for _, m := range meds {
		if m.Role == RoleLutealSupport {
			luteal = append(luteal, m)
		}
	}
	ds, err := loadDoseState(ctx, q, userID, today, today, luteal)
	if err != nil {
		return v, err
	}
	v.Luteal = DosesOn(today, luteal, ds.taken, ds.sites)
	return v, nil
}
