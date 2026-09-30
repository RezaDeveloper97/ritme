package pelvic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/pelvic/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// ErrNoProgram is returned when a session is saved without a program.
var ErrNoProgram = errors.New("pelvic: no program")

// LevelSource reads the pelvic_levels catalog group (catalog.Reader, cached per group).
type LevelSource interface {
	Items(ctx context.Context, group string) ([]catalog.Item, error)
}

// Service reads and writes the pelvic program, trained days and bladder diary.
type Service struct {
	q      store.Querier
	levels LevelSource
}

// NewService returns a Service.
func NewService(q store.Querier, levels LevelSource) *Service { return &Service{q: q, levels: levels} }

// Overview is the plan screen: the program (nil when none) and today's diary.
type Overview struct {
	Program *ProgramView
	Diary   Diary
}

// ProgramView is the program as of today.
type ProgramView struct {
	StartedOn  civildate.Date
	Week       int
	Completed  bool
	Level      *Level
	StreakDays int
	TodayDone  bool
	WeekDays   []WeekDay
}

// Diary is one bladder-diary day (zero values when nothing was logged).
type Diary struct {
	Date        civildate.Date
	Leak        *string
	NightVoids  *int
	UTISymptoms []string
}

// Overview loads the plan screen for today: program, levels (catalog), trained days, today's diary.
func (s *Service) Overview(ctx context.Context, userID uint64, today civildate.Date) (Overview, error) {
	diary, err := s.Diary(ctx, userID, today)
	if err != nil {
		return Overview{}, err
	}
	prog, err := s.q.GetProgram(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return Overview{Diary: diary}, nil
	}
	if err != nil {
		return Overview{}, fmt.Errorf("pelvic: load program: %w", err)
	}
	level, err := s.levelFor(ctx, prog.StartedOn, today)
	if err != nil {
		return Overview{}, err
	}
	days, err := s.q.ListSessionDays(ctx, store.ListSessionDaysParams{
		UserID: userID, FromDate: today.AddDays(-streakWindow), ToDate: today,
	})
	if err != nil {
		return Overview{}, fmt.Errorf("pelvic: load sessions: %w", err)
	}
	done := make(map[civildate.Date]bool, len(days))
	for _, d := range days {
		done[d] = true
	}
	week, completed := Week(prog.StartedOn, today)
	return Overview{Diary: diary, Program: &ProgramView{
		StartedOn: prog.StartedOn, Week: week, Completed: completed, Level: level,
		StreakDays: Streak(done, today), TodayDone: done[today], WeekDays: WeekDays(done, today),
	}}, nil
}

// levelFor is the catalog level of the program week on day (nil when the catalog has none).
func (s *Service) levelFor(ctx context.Context, startedOn, day civildate.Date) (*Level, error) {
	items, err := s.levels.Items(ctx, LevelsGroup)
	if err != nil {
		return nil, fmt.Errorf("pelvic: load levels: %w", err)
	}
	week, _ := Week(startedOn, day)
	return LevelFor(ParseLevels(items), week), nil
}

func tehranNow(now time.Time) sql.NullTime {
	return sql.NullTime{Time: now.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}

// Start starts (or restarts) the program on startedOn.
func (s *Service) Start(ctx context.Context, userID uint64, startedOn civildate.Date, now time.Time) error {
	if err := s.q.UpsertProgram(ctx, store.UpsertProgramParams{UserID: userID, StartedOn: startedOn, Now: tehranNow(now)}); err != nil {
		return fmt.Errorf("pelvic: start program: %w", err)
	}
	return nil
}

// Stop deletes the program; trained days and the diary stay (the user's own log).
func (s *Service) Stop(ctx context.Context, userID uint64) error {
	if err := s.q.DeleteProgram(ctx, userID); err != nil {
		return fmt.Errorf("pelvic: stop program: %w", err)
	}
	return nil
}

// AddSession adds a session to its day, tagged with the level of that day's program week. ErrNoProgram without a
// program.
func (s *Service) AddSession(ctx context.Context, userID uint64, in SessionInput, now time.Time) error {
	prog, err := s.q.GetProgram(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoProgram
	}
	if err != nil {
		return fmt.Errorf("pelvic: load program: %w", err)
	}
	level, err := s.levelFor(ctx, prog.StartedOn, in.Date)
	if err != nil {
		return err
	}
	var code sql.NullString
	if level != nil {
		code = sql.NullString{String: level.Item.Code, Valid: true}
	}
	if err := s.q.AddSession(ctx, store.AddSessionParams{
		UserID: userID, SessionDate: in.Date,
		SetsCompleted: uint16(in.SetsCompleted), //nolint:gosec // G115: validated 0…maxSets
		DurationSec:   uint32(in.DurationSec),   //nolint:gosec // G115: validated 1…maxDurationSec
		LevelCode:     code, Now: tehranNow(now),
	}); err != nil {
		return fmt.Errorf("pelvic: add session: %w", err)
	}
	return nil
}

// Diary is one bladder-diary day (empty when nothing was logged).
func (s *Service) Diary(ctx context.Context, userID uint64, date civildate.Date) (Diary, error) {
	row, err := s.q.GetBladderLog(ctx, store.GetBladderLogParams{UserID: userID, LogDate: date})
	if errors.Is(err, sql.ErrNoRows) {
		return Diary{Date: date, UTISymptoms: []string{}}, nil
	}
	if err != nil {
		return Diary{}, fmt.Errorf("pelvic: load diary: %w", err)
	}
	return diaryFromRow(row), nil
}

func diaryFromRow(row store.PelvicBladderLog) Diary {
	d := Diary{Date: row.LogDate, UTISymptoms: []string{}}
	if row.Leak.Valid {
		s := row.Leak.String
		d.Leak = &s
	}
	if row.NightVoids.Valid {
		n := int(row.NightVoids.Int16)
		d.NightVoids = &n
	}
	if row.UtiSymptoms.Valid {
		var syms []string
		if json.Unmarshal(row.UtiSymptoms.V, &syms) == nil && syms != nil {
			d.UTISymptoms = syms
		}
	}
	return d
}

// SaveDiary merges the sent keys over the day's row; a row left empty is deleted. Returns the saved day.
func (s *Service) SaveDiary(ctx context.Context, userID uint64, in DiaryInput, now time.Time) (Diary, error) {
	d, err := s.Diary(ctx, userID, in.Date)
	if err != nil {
		return Diary{}, err
	}
	if in.Set["leak"] {
		d.Leak = in.Leak
	}
	if in.Set["night_voids"] {
		d.NightVoids = in.NightVoids
	}
	if in.Set["uti_symptoms"] {
		d.UTISymptoms = in.UTISymptoms
		if d.UTISymptoms == nil {
			d.UTISymptoms = []string{}
		}
	}
	if d.Leak == nil && d.NightVoids == nil && len(d.UTISymptoms) == 0 {
		if err := s.q.DeleteBladderLog(ctx, store.DeleteBladderLogParams{UserID: userID, LogDate: in.Date}); err != nil {
			return Diary{}, fmt.Errorf("pelvic: delete diary: %w", err)
		}
		return d, nil
	}
	params := store.UpsertBladderLogParams{UserID: userID, LogDate: in.Date, Now: tehranNow(now)}
	if d.Leak != nil {
		params.Leak = sql.NullString{String: *d.Leak, Valid: true}
	}
	if d.NightVoids != nil {
		params.NightVoids = sql.NullInt16{Int16: int16(*d.NightVoids), Valid: true} //nolint:gosec // G115: validated 0…maxNightVoids
	}
	if len(d.UTISymptoms) > 0 {
		raw, err := json.Marshal(d.UTISymptoms)
		if err != nil {
			return Diary{}, fmt.Errorf("pelvic: encode symptoms: %w", err)
		}
		params.UtiSymptoms = db.NullRawJSON{V: raw, Valid: true}
	}
	if err := s.q.UpsertBladderLog(ctx, params); err != nil {
		return Diary{}, fmt.Errorf("pelvic: save diary: %w", err)
	}
	return d, nil
}
