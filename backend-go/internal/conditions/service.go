// Package conditions is the condition programs (CB-COND-01, canvas boards nbl_Cond_Hub / _Endo / _PMDD / _Bleed):
// enrolment in the endometriosis, PMDD, heavy-bleeding and PCOS programs, the endometriosis pain diary, the PMDD
// daily questionnaire with its cycle chart, and the PBAC pad chart.
//
// No parallel log: what bloom's log taxonomy (B-N3-01, internal/healthlog) already has a slot for is written there
// — the pain locations, the 0–10 score and the relief methods (category `pain`), associated symptoms whose catalog
// item names a taxonomy slot (`pain_associated` meta.log), and the PBAC clot chips (`bleeding.clots` /
// `bleeding.clot_size`). Only what the taxonomy has no place for lives in this package's tables. PCOS is enrolment
// only (DECISIONS #8) and reads the existing logs.
//
// The lists (pain types, associated symptoms, PMDD items) and every clinical text (alerts, the pattern sentence, the
// "needs 2 complete cycles" note, the PBAC explanation) are admin-editable catalog content, needs_review.
// Health data: every read and write is scoped to the authenticated user.
package conditions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/conditions/store"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Programs, in hub order (catalog `condition_programs` carries their copy).
const (
	ProgramEndo          = "endo"
	ProgramPMDD          = "pmdd"
	ProgramHeavyBleeding = "heavy_bleeding"
	ProgramPCOS          = "pcos"
)

// Programs are every program code, in hub order.
var Programs = []string{ProgramEndo, ProgramPMDD, ProgramHeavyBleeding, ProgramPCOS}

// Catalog groups the package reads.
const (
	GroupPainTypes      = "pain_types"
	GroupPainAssociated = "pain_associated"
	GroupPMDDItems      = "pmdd_items"
	GroupAlerts         = "condition_alerts"
)

// ErrNotEnrolled is returned when a program diary is written without being enrolled in the program.
var ErrNotEnrolled = errors.New("conditions: not enrolled")

// CatalogSource reads catalog groups (catalog.Reader: active items in order, cached per group).
type CatalogSource interface {
	Items(ctx context.Context, group string) ([]catalog.Item, error)
}

// Conn is the database the service writes through in one transaction (a *sql.DB).
type Conn interface {
	store.DBTX
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// Service reads and writes the user's programs and diaries.
type Service struct {
	conn    Conn
	catalog CatalogSource
}

// NewService returns a Service on conn.
func NewService(conn Conn, cat CatalogSource) *Service { return &Service{conn: conn, catalog: cat} }

func tehranNow(now time.Time) sql.NullTime {
	return sql.NullTime{Time: now.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}

// inTx runs fn on one READ COMMITTED transaction (the isolation healthlog's day save runs under), with this
// package's queries and a healthlog service bound to it.
func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries, logs *healthlog.Service) error) error {
	tx, err := s.conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("conditions: begin: %w", err)
	}
	if err := fn(store.New(tx), healthlog.NewService(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("conditions: commit: %w", err)
	}
	return nil
}

// Enrolment is one program and the user's enrolment in it.
type Enrolment struct {
	Program    string
	Enrolled   bool
	EnrolledOn civildate.Date
}

// Enrolments lists every program with the user's enrolment state, in hub order.
func (s *Service) Enrolments(ctx context.Context, userID uint64) ([]Enrolment, error) {
	rows, err := store.New(s.conn).ListEnrolments(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("conditions: load enrolments: %w", err)
	}
	out := make([]Enrolment, len(Programs))
	for i, p := range Programs {
		out[i] = Enrolment{Program: p}
		for _, r := range rows {
			if r.Program == p {
				out[i].Enrolled, out[i].EnrolledOn = true, r.EnrolledOn
			}
		}
	}
	return out, nil
}

// Enrol joins a program (joining again keeps the first date).
func (s *Service) Enrol(ctx context.Context, userID uint64, program string, on civildate.Date, now time.Time) error {
	if err := store.New(s.conn).Enrol(ctx, store.EnrolParams{UserID: userID, Program: program, EnrolledOn: on, Now: tehranNow(now)}); err != nil {
		return fmt.Errorf("conditions: enrol: %w", err)
	}
	return nil
}

// Leave leaves a program; the diary data stays (the user's own log).
func (s *Service) Leave(ctx context.Context, userID uint64, program string) error {
	if err := store.New(s.conn).Leave(ctx, store.LeaveParams{UserID: userID, Program: program}); err != nil {
		return fmt.Errorf("conditions: leave: %w", err)
	}
	return nil
}

func requireEnrolled(ctx context.Context, q *store.Queries, userID uint64, program string) error {
	ok, err := q.IsEnrolled(ctx, store.IsEnrolledParams{UserID: userID, Program: program})
	if err != nil {
		return fmt.Errorf("conditions: load enrolment: %w", err)
	}
	if !ok {
		return ErrNotEnrolled
	}
	return nil
}

// codes are the item codes of a catalog group (active items, catalog order).
func (s *Service) codes(ctx context.Context, group string) ([]catalog.Item, []string, error) {
	items, err := s.catalog.Items(ctx, group)
	if err != nil {
		return nil, nil, fmt.Errorf("conditions: load catalog %s: %w", group, err)
	}
	codes := make([]string, len(items))
	for i, it := range items {
		codes[i] = it.Code
	}
	return items, codes, nil
}

// alert is the catalog item `code` of condition_alerts (nil when inactive or missing).
func (s *Service) alert(ctx context.Context, code string) (*catalog.Item, error) {
	items, err := s.catalog.Items(ctx, GroupAlerts)
	if err != nil {
		return nil, fmt.Errorf("conditions: load alerts: %w", err)
	}
	i := slices.IndexFunc(items, func(it catalog.Item) bool { return it.Code == code })
	if i < 0 {
		return nil, nil
	}
	return &items[i], nil
}

// alerts are the active condition_alerts items among codes, in the given order.
func (s *Service) alerts(ctx context.Context, codes ...string) ([]catalog.Item, error) {
	out := []catalog.Item{}
	for _, c := range codes {
		it, err := s.alert(ctx, c)
		if err != nil {
			return nil, err
		}
		if it != nil {
			out = append(out, *it)
		}
	}
	return out, nil
}

// cycleDefaults are the profile's cycle and period lengths (0 when unset).
func cycleDefaults(ctx context.Context, q *store.Queries, userID uint64) (cycleLen, periodLen int, err error) {
	row, err := q.GetCycleDefaults(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, fmt.Errorf("conditions: load cycle defaults: %w", err)
	}
	if row.CycleDuration.Valid {
		cycleLen = int(row.CycleDuration.Int16)
	}
	if row.PeriodDuration.Valid {
		periodLen = int(row.PeriodDuration.Int16)
	}
	return cycleLen, periodLen, nil
}

// periodStarts are the user's latest period starts on or before day, newest first (at most 3).
func periodStarts(ctx context.Context, q *store.Queries, userID uint64, day civildate.Date) ([]PeriodStart, error) {
	rows, err := q.ListPeriodStarts(ctx, store.ListPeriodStartsParams{UserID: userID, Day: day})
	if err != nil {
		return nil, fmt.Errorf("conditions: load period starts: %w", err)
	}
	out := make([]PeriodStart, len(rows))
	for i, r := range rows {
		out[i] = PeriodStart{Start: r.PeriodStartDate}
		if r.PeriodEndDate.Valid {
			end := r.PeriodEndDate.Date
			out[i].End = &end
		}
		if r.BleedingLength.Valid && r.BleedingLength.Int32 > 0 {
			out[i].BleedingDays = int(r.BleedingLength.Int32)
		}
	}
	return out, nil
}

// PeriodStart is one cycle_histories period.
type PeriodStart struct {
	Start        civildate.Date
	End          *civildate.Date // period_end_date
	BleedingDays int             // bleeding_length (0 = unknown)
}

// LastPeriodDay is the period's last day: period_end_date, else start + bleeding length − 1, else start +
// fallback − 1 (fallback ≤ 0 → 5 days).
func (p PeriodStart) LastPeriodDay(fallback int) civildate.Date {
	switch {
	case p.End != nil && !p.End.Before(p.Start):
		return *p.End
	case p.BleedingDays > 0:
		return p.Start.AddDays(p.BleedingDays - 1)
	case fallback > 0:
		return p.Start.AddDays(fallback - 1)
	default:
		return p.Start.AddDays(defaultPeriodDays - 1)
	}
}

const (
	defaultPeriodDays = 5
	defaultCycleDays  = 28
)
