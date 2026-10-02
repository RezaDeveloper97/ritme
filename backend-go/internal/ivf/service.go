package ivf

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/go-sql-driver/mysql"

	carestore "github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/ivf/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Conn is the database the service writes through in one transaction (a *sql.DB).
type Conn interface {
	store.DBTX
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// CatalogSource reads catalog groups (catalog.Reader: active items in order, cached per group).
type CatalogSource interface {
	Items(ctx context.Context, group string) ([]catalog.Item, error)
}

// CompanionLister lists the owner's companion links (companion.Service).
type CompanionLister interface {
	ListForOwner(ctx context.Context, ownerID uint64) ([]companion.Link, error)
}

// Service is the IVF domain.
type Service struct {
	conn       Conn
	catalog    CatalogSource
	companions CompanionLister
}

// NewService returns a Service on conn reading lists from cat and companion links from comp.
func NewService(conn Conn, cat CatalogSource, comp CompanionLister) *Service {
	return &Service{conn: conn, catalog: cat, companions: comp}
}

func ts(now time.Time) sql.NullTime {
	return sql.NullTime{Time: now.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}

func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries, cq *carestore.Queries) error) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ivf: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(store.New(tx), carestore.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ivf: commit: %w", err)
	}
	return nil
}

// codes are the active item codes of a catalog group, in order.
func (s *Service) codes(ctx context.Context, group string) ([]string, error) {
	items, err := s.catalog.Items(ctx, group)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Code)
	}
	return out, nil
}

// checkCode is a 422 on field when code is not an active item of group.
func (s *Service) checkCode(ctx context.Context, group, field, code string) error {
	codes, err := s.codes(ctx, group)
	if err != nil {
		return err
	}
	if !slices.Contains(codes, code) {
		return fieldErr(field, "unknown_code")
	}
	return nil
}

// Companion is the «همدمت هم در جریان باشد» context: whether an active companion link exists and whether any of
// them may see the medicines / appointments (B-N4-02 grants).
type Companion struct {
	Linked       bool
	Meds         bool
	Appointments bool
}

func (s *Service) companion(ctx context.Context, userID uint64) (Companion, error) {
	links, err := s.companions.ListForOwner(ctx, userID)
	if err != nil {
		return Companion{}, err
	}
	var c Companion
	for _, l := range links {
		if l.Status != companion.StatusActive {
			continue
		}
		c.Linked = true
		c.Meds = c.Meds || l.Grants.Of(companion.SectionMeds).CanRead()
		c.Appointments = c.Appointments || l.Grants.Of(companion.SectionAppointments).CanRead()
	}
	return c, nil
}

// activeCycle is the open cycle, nil when none.
func activeCycle(ctx context.Context, q *store.Queries, userID uint64, lock bool) (*Cycle, error) {
	get := q.GetActiveCycle
	if lock {
		get = q.LockActiveCycle
	}
	row, err := get(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ivf: load cycle: %w", err)
	}
	c := cycleFromRow(row)
	return &c, nil
}

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// StartInput is a validated POST /ivf/cycles.
type StartInput struct {
	Number          int // 0 = previous highest + 1
	Protocol        string
	Stage           string
	StartedOn       civildate.Date
	StimStartedOn   civildate.Date
	NotifyCompanion bool
}

// StartCycle opens a cycle and switches bloom's «IVF/IUI» flag on. One open cycle per user (ErrCycleOpen).
func (s *Service) StartCycle(ctx context.Context, userID uint64, in StartInput, now time.Time) error {
	if in.Protocol != "" {
		if err := s.checkCode(ctx, GroupProtocols, "protocol", in.Protocol); err != nil {
			return err
		}
	}
	if !in.StimStartedOn.IsZero() && in.StimStartedOn.Before(in.StartedOn) {
		return fieldErr("stim_started_on", "before_cycle_start")
	}
	if in.NotifyCompanion {
		if err := s.requireCompanion(ctx, userID); err != nil {
			return err
		}
	}
	err := s.inTx(ctx, func(q *store.Queries, _ *carestore.Queries) error {
		open, err := activeCycle(ctx, q, userID, true)
		if err != nil {
			return err
		}
		if open != nil {
			return ErrCycleOpen
		}
		number := in.Number
		if number == 0 {
			st, err := q.CycleStats(ctx, userID)
			if err != nil {
				return fmt.Errorf("ivf: cycle stats: %w", err)
			}
			number = min(int(st.MaxNumber)+1, MaxCycleNumber)
		}
		if _, err := q.InsertCycle(ctx, store.InsertCycleParams{
			UserID: userID, ActiveUserID: sql.NullInt64{Int64: int64(userID), Valid: true}, //nolint:gosec // ids fit
			Number: uint8(number), Protocol: sql.NullString{String: in.Protocol, Valid: in.Protocol != ""}, //nolint:gosec // ≤ MaxCycleNumber
			Stage: in.Stage, StartedOn: in.StartedOn,
			StimStartedOn:   civildate.NullDate{Date: in.StimStartedOn, Valid: !in.StimStartedOn.IsZero()},
			NotifyCompanion: in.NotifyCompanion, Now: ts(now),
		}); err != nil {
			if isDuplicate(err) {
				return ErrCycleOpen
			}
			return fmt.Errorf("ivf: insert cycle: %w", err)
		}
		if err := q.SetIVFSwitch(ctx, store.SetIVFSwitchParams{UserID: userID, IvfIui: true, Now: ts(now)}); err != nil {
			return fmt.Errorf("ivf: switch on: %w", err)
		}
		return nil
	})
	return err
}

func (s *Service) requireCompanion(ctx context.Context, userID uint64) error {
	c, err := s.companion(ctx, userID)
	if err != nil {
		return err
	}
	if !c.Linked {
		return fieldErr("notify_companion", "no_companion")
	}
	return nil
}

// OptDate is an optional date field of a partial update: Set with a zero V clears it.
type OptDate struct {
	Set bool
	V   civildate.Date
}

// OptTime is an optional datetime field of a partial update: Set with a zero V clears it.
type OptTime struct {
	Set bool
	V   time.Time
}

// CyclePatch is a validated PUT /ivf/cycles/current: only the fields sent change.
type CyclePatch struct {
	Number          *int
	Protocol        *string // "" clears
	Stage           *string
	StartedOn       *civildate.Date
	StimStartedOn   OptDate
	RetrievalAt     OptTime
	TransferAt      OptTime
	BetaOn          OptDate
	NextScanAt      OptTime
	NotifyCompanion *bool
}

func (p CyclePatch) apply(c Cycle) Cycle {
	if p.Number != nil {
		c.Number = *p.Number
	}
	if p.Protocol != nil {
		c.Protocol = *p.Protocol
	}
	if p.Stage != nil {
		c.Stage = *p.Stage
	}
	if p.StartedOn != nil {
		c.StartedOn = *p.StartedOn
	}
	if p.StimStartedOn.Set {
		c.StimStartedOn = p.StimStartedOn.V
	}
	if p.RetrievalAt.Set {
		c.RetrievalAt = p.RetrievalAt.V
	}
	if p.TransferAt.Set {
		c.TransferAt = p.TransferAt.V
	}
	if p.BetaOn.Set {
		c.BetaOn = p.BetaOn.V
	}
	if p.NextScanAt.Set {
		c.NextScanAt = p.NextScanAt.V
	}
	if p.NotifyCompanion != nil {
		c.NotifyCompanion = *p.NotifyCompanion
	}
	return c
}

// checkOrder validates the stage dates against each other: nothing before the cycle start, the transfer not before
// the retrieval, the beta test not before the transfer.
func checkOrder(c Cycle) error {
	start := c.StartedOn
	before := func(d civildate.Date, o civildate.Date) bool { return !d.IsZero() && !o.IsZero() && d.Before(o) }
	switch {
	case before(c.StimStartedOn, start):
		return fieldErr("stim_started_on", "before_cycle_start")
	case before(dayOf(c.RetrievalAt), start):
		return fieldErr("retrieval_at", "before_cycle_start")
	case before(dayOf(c.TransferAt), start):
		return fieldErr("transfer_at", "before_cycle_start")
	case !c.TransferAt.IsZero() && !c.RetrievalAt.IsZero() && c.TransferAt.Before(c.RetrievalAt):
		return fieldErr("transfer_at", "transfer_before_retrieval")
	case before(c.BetaOn, start):
		return fieldErr("beta_on", "before_cycle_start")
	case before(c.BetaOn, c.TransferOn()):
		return fieldErr("beta_on", "beta_before_transfer")
	case before(dayOf(c.NextScanAt), start):
		return fieldErr("next_scan_at", "before_cycle_start")
	}
	return nil
}

func nt(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func nd(d civildate.Date) civildate.NullDate { return civildate.NullDate{Date: d, Valid: !d.IsZero()} }

// UpdateCycle applies a partial update to the open cycle and syncs its care appointments (scan, retrieval,
// transfer, beta) in the same transaction; appointment titles are written in locale.
func (s *Service) UpdateCycle(ctx context.Context, userID uint64, p CyclePatch, now time.Time, locale string) error {
	if p.Protocol != nil && *p.Protocol != "" {
		if err := s.checkCode(ctx, GroupProtocols, "protocol", *p.Protocol); err != nil {
			return err
		}
	}
	if p.NotifyCompanion != nil && *p.NotifyCompanion {
		if err := s.requireCompanion(ctx, userID); err != nil {
			return err
		}
	}
	return s.inTx(ctx, func(q *store.Queries, cq *carestore.Queries) error {
		cur, err := activeCycle(ctx, q, userID, true)
		if err != nil {
			return err
		}
		if cur == nil {
			return ErrNoCycle
		}
		c := p.apply(*cur)
		if err := checkOrder(c); err != nil {
			return err
		}
		if err := q.UpdateCycle(ctx, store.UpdateCycleParams{
			Number: uint8(c.Number), Protocol: sql.NullString{String: c.Protocol, Valid: c.Protocol != ""}, //nolint:gosec // ≤ MaxCycleNumber
			Stage: c.Stage, StartedOn: c.StartedOn, StimStartedOn: nd(c.StimStartedOn), RetrievalAt: nt(c.RetrievalAt),
			TransferAt: nt(c.TransferAt), BetaOn: nd(c.BetaOn), NextScanAt: nt(c.NextScanAt),
			NotifyCompanion: c.NotifyCompanion, Now: ts(now), ID: c.ID, UserID: userID,
		}); err != nil {
			return fmt.Errorf("ivf: update cycle: %w", err)
		}
		return syncAppointments(ctx, q, cq, userID, c.ID, plan(c), now, locale)
	})
}

// RecordOutcome closes the open cycle with its result on date. Upcoming linked appointments are removed; after a
// negative or cancelled cycle the cycle's medicine reminders are switched off (a positive result keeps them: the
// doctor decides when luteal support stops). Returns the closed cycle.
func (s *Service) RecordOutcome(ctx context.Context, userID uint64, outcome string, date civildate.Date, now time.Time) (Cycle, error) {
	var closed Cycle
	err := s.inTx(ctx, func(q *store.Queries, cq *carestore.Queries) error {
		cur, err := activeCycle(ctx, q, userID, true)
		if err != nil {
			return err
		}
		if cur == nil {
			return ErrNoCycle
		}
		if date.Before(cur.StartedOn) {
			return fieldErr("date", "before_cycle_start")
		}
		if err := q.CloseCycle(ctx, store.CloseCycleParams{
			Outcome: sql.NullString{String: outcome, Valid: true}, OutcomeOn: nd(date), Now: ts(now),
			ID: cur.ID, UserID: userID,
		}); err != nil {
			return fmt.Errorf("ivf: close cycle: %w", err)
		}
		if err := dropUpcoming(ctx, q, cq, userID, cur.ID, now); err != nil {
			return err
		}
		if outcome != OutcomePositive {
			meds, err := q.ListCycleMeds(ctx, store.ListCycleMedsParams{UserID: userID, CycleID: cur.ID})
			if err != nil {
				return fmt.Errorf("ivf: load medicines: %w", err)
			}
			for _, m := range meds {
				if err := q.SetReminderActive(ctx, store.SetReminderActiveParams{
					IsActive: false, Now: ts(now), ID: m.Reminder.ID, UserID: userID,
				}); err != nil {
					return fmt.Errorf("ivf: stop medicine reminder: %w", err)
				}
			}
		}
		closed = *cur
		closed.Open, closed.Outcome, closed.OutcomeOn = false, outcome, date
		return nil
	})
	return closed, err
}
