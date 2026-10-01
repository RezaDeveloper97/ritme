package menopause

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/menopause/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Hot-flash timer rules (nbl_Meno_HotFlash).
const (
	// MaxFlashSeconds caps one flash: a timer left running longer is closed at this length.
	MaxFlashSeconds = 60 * 60
	// MaxBackfillDays: a flash logged after the fact may start at most this many days ago.
	MaxBackfillDays = 7
	// NightFromHour / NightToHour: a flash starting in [22:00, 06:00) Tehran is a night flash unless she says
	// otherwise.
	NightFromHour = 22
	NightToHour   = 6
)

// Severities are the board's chips, stored 1–4 in hot_flashes.severity.
var Severities = []string{"mild", "moderate", "severe", "very_severe"}

// TriggerUnknown is the board's «نمی‌دانم» chip.
const TriggerUnknown = "unknown"

// Triggers are the trigger codes a flash may carry: the log taxonomy's menopause.triggers options (CB-MENO-01)
// plus `unknown`.
func Triggers() []string { return append(LogTriggers(), TriggerUnknown) }

// LogTriggers are the menopause.triggers options of the log taxonomy, in taxonomy order.
func LogTriggers() []string {
	var out []string
	if c, ok := taxonomy.CategoryByCode("menopause"); ok {
		if p, ok := c.Param("triggers"); ok {
			for _, o := range p.Options {
				out = append(out, o.Code)
			}
		}
	}
	return out
}

// ErrFlashNotFound: no flash with that id for the user.
var ErrFlashNotFound = errors.New("menopause: hot flash not found")

// Flash is one hot_flashes row.
type Flash struct {
	ID        uint64
	StartedAt time.Time
	Duration  *int // seconds; nil while the timer runs
	Severity  string
	Night     bool
	Sweat     bool
	Triggers  []string
}

// Running reports whether the timer still runs.
func (f Flash) Running() bool { return f.Duration == nil }

// Elapsed is the timer's length at now (capped at MaxFlashSeconds); the duration once stopped.
func (f Flash) Elapsed(now time.Time) int {
	if f.Duration != nil {
		return *f.Duration
	}
	return min(max(int(now.Sub(f.StartedAt)/time.Second), 0), MaxFlashSeconds)
}

// Day is the Tehran day the flash started on.
func (f Flash) Day() civildate.Date { return civildate.InTehran(f.StartedAt) }

func clampDuration(s int) int { return min(max(s, 1), MaxFlashSeconds) }

// IsNightHour reports whether a flash starting at t is a night flash by default.
func IsNightHour(t time.Time) bool {
	h := t.In(civildate.Tehran).Hour()
	return h >= NightFromHour || h < NightToHour
}

func flashOf(r store.HotFlash) Flash {
	f := Flash{ID: r.ID, StartedAt: r.StartedAt.In(civildate.Tehran), Night: r.Night, Sweat: r.Sweat, Triggers: []string{}}
	if r.DurationS.Valid {
		d := int(r.DurationS.Int32)
		f.Duration = &d
	}
	if r.Severity.Valid && r.Severity.Int16 >= 1 && int(r.Severity.Int16) <= len(Severities) {
		f.Severity = Severities[r.Severity.Int16-1]
	}
	if r.Triggers.Valid {
		var list []string
		if json.Unmarshal(r.Triggers.V, &list) == nil {
			f.Triggers = list
		}
	}
	return f
}

// FlashInput is a validated start (POST) or stop (POST …/stop). Set lists the keys sent.
type FlashInput struct {
	Set       map[string]bool
	StartedAt *time.Time
	Duration  *int
	Severity  string
	Night     *bool
	Sweat     *bool
	Triggers  []string
}

func severityOf(code string) sql.NullInt16 {
	i := slices.Index(Severities, code)
	if i < 0 {
		return sql.NullInt16{}
	}
	return sql.NullInt16{Int16: int16(i + 1), Valid: true} //nolint:gosec // 1–4
}

func triggersJSON(list []string) rootdb.NullRawJSON {
	if len(list) == 0 {
		return rootdb.NullRawJSON{}
	}
	b, _ := json.Marshal(list)
	return rootdb.NullRawJSON{V: b, Valid: true}
}

func durationOf(d *int) sql.NullInt32 {
	if d == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(*d), Valid: true} //nolint:gosec // ≤ MaxFlashSeconds
}

// StartFlash starts the timer, or logs a finished flash when a duration is sent. Starting while a timer runs
// returns that timer (created = false); a timer forgotten for longer than MaxFlashSeconds is closed at that length
// first.
func (s *Service) StartFlash(ctx context.Context, userID uint64, in FlashInput, now time.Time) (Flash, bool, error) {
	if in.Duration == nil {
		run, err := s.q.GetRunningHotFlash(ctx, userID)
		switch {
		case err == nil:
			f := flashOf(run)
			if now.Sub(f.StartedAt) < MaxFlashSeconds*time.Second {
				return f, false, nil
			}
			if err := s.update(ctx, userID, f, intPtr(MaxFlashSeconds), now); err != nil {
				return Flash{}, false, err
			}
		case !errors.Is(err, sql.ErrNoRows):
			return Flash{}, false, fmt.Errorf("menopause: load running flash: %w", err)
		}
	}
	start := now
	if in.StartedAt != nil {
		start = *in.StartedAt
	} else if in.Duration != nil {
		start = now.Add(-time.Duration(*in.Duration) * time.Second)
	}
	f := Flash{StartedAt: start.In(civildate.Tehran).Truncate(time.Second), Duration: in.Duration, Severity: in.Severity,
		Night: IsNightHour(start), Triggers: in.Triggers}
	if in.Night != nil {
		f.Night = *in.Night
	}
	if in.Sweat != nil {
		f.Sweat = *in.Sweat
	}
	id, err := s.q.CreateHotFlash(ctx, store.CreateHotFlashParams{
		UserID: userID, StartedAt: f.StartedAt, DurationS: durationOf(f.Duration), Severity: severityOf(f.Severity),
		Night: f.Night, Sweat: f.Sweat, Triggers: triggersJSON(f.Triggers), Now: tehranNow(now),
	})
	if err != nil {
		return Flash{}, false, fmt.Errorf("menopause: create flash: %w", err)
	}
	f.ID = uint64(id) //nolint:gosec // auto-increment id
	if f.Triggers == nil {
		f.Triggers = []string{}
	}
	return f, true, nil
}

func intPtr(n int) *int { return &n }

func (s *Service) update(ctx context.Context, userID uint64, f Flash, duration *int, now time.Time) error {
	if err := s.q.UpdateHotFlash(ctx, store.UpdateHotFlashParams{
		ID: f.ID, UserID: userID, DurationS: durationOf(duration), Severity: severityOf(f.Severity),
		Night: f.Night, Sweat: f.Sweat, Triggers: triggersJSON(f.Triggers), Now: tehranNow(now),
	}); err != nil {
		return fmt.Errorf("menopause: update flash: %w", err)
	}
	return nil
}

// StopFlash stops a running timer (duration = now − start, capped) and applies the details sent; on a stopped
// flash it only edits the details (the duration stays).
func (s *Service) StopFlash(ctx context.Context, userID, id uint64, in FlashInput, now time.Time) (Flash, error) {
	row, err := s.q.GetHotFlash(ctx, store.GetHotFlashParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Flash{}, ErrFlashNotFound
	}
	if err != nil {
		return Flash{}, fmt.Errorf("menopause: load flash: %w", err)
	}
	f := flashOf(row)
	if f.Running() {
		f.Duration = intPtr(clampDuration(f.Elapsed(now)))
	}
	if in.Set["severity"] {
		f.Severity = in.Severity
	}
	if in.Set["night"] && in.Night != nil {
		f.Night = *in.Night
	}
	if in.Set["sweat"] && in.Sweat != nil {
		f.Sweat = *in.Sweat
	}
	if in.Set["triggers"] {
		f.Triggers = in.Triggers
		if f.Triggers == nil {
			f.Triggers = []string{}
		}
	}
	if err := s.update(ctx, userID, f, f.Duration, now); err != nil {
		return Flash{}, err
	}
	return f, nil
}

// Flashes are the user's flashes started in [from, to] (whole Tehran days), newest first.
func (s *Service) Flashes(ctx context.Context, userID uint64, from, to civildate.Date) ([]Flash, error) {
	rows, err := s.q.ListHotFlashesInRange(ctx, store.ListHotFlashesInRangeParams{
		UserID: userID, FromAt: from.TehranMidnight(), ToAt: to.AddDays(1).TehranMidnight(),
	})
	if err != nil {
		return nil, fmt.Errorf("menopause: list flashes: %w", err)
	}
	out := make([]Flash, len(rows))
	for i, r := range rows {
		out[i] = flashOf(r)
	}
	return out, nil
}

// FlashDay is one day of the timer screen: the flashes (newest first) and the tiles.
type FlashDay struct {
	Date    civildate.Date
	Flashes []Flash
	Running *Flash // the timer still running (any day), nil when none
}

// Count is the number of flashes of the day.
func (d FlashDay) Count() int { return len(d.Flashes) }

// NightCount is the number of night flashes of the day.
func (d FlashDay) NightCount() int {
	n := 0
	for _, f := range d.Flashes {
		if f.Night {
			n++
		}
	}
	return n
}

// AvgDuration is the mean length of the day's finished flashes in whole seconds (nil when none finished).
func (d FlashDay) AvgDuration() *int {
	sum, n := 0, 0
	for _, f := range d.Flashes {
		if f.Duration != nil {
			sum += *f.Duration
			n++
		}
	}
	if n == 0 {
		return nil
	}
	avg := (sum + n/2) / n
	return &avg
}

// Day loads one day of flashes and the running timer (a timer forgotten for MaxFlashSeconds or more is not shown as
// running; the next start closes it).
func (s *Service) Day(ctx context.Context, userID uint64, day civildate.Date, now time.Time) (FlashDay, error) {
	list, err := s.Flashes(ctx, userID, day, day)
	if err != nil {
		return FlashDay{}, err
	}
	out := FlashDay{Date: day, Flashes: list}
	run, err := s.q.GetRunningHotFlash(ctx, userID)
	switch {
	case err == nil:
		if f := flashOf(run); now.Sub(f.StartedAt) < MaxFlashSeconds*time.Second {
			out.Running = &f
		}
	case !errors.Is(err, sql.ErrNoRows):
		return FlashDay{}, fmt.Errorf("menopause: load running flash: %w", err)
	}
	return out, nil
}
