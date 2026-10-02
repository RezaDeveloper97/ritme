package ivf

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/care"
	carestore "github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/ivf/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// MedInput is a validated POST /ivf/meds or PUT /ivf/meds/{id} (a full replace).
type MedInput struct {
	Name         string
	Role         string
	Route        string
	Dose         *string
	Unit         *string
	Times        []string // sorted "HH:MM"; for the trigger: its time
	StartsOn     civildate.Date
	EndsOn       civildate.Date // zero = ongoing; the trigger: its day
	TriggerAt    time.Time      // trigger only
	Notes        *string
	StockUnits   *int
	StockUnit    string
	DosesPerUnit int
	// SubtitleLocale is the language of the legacy reminders.subtitle (the default language, like /care).
	SubtitleLocale string
}

// formOf is the care medication form of a route.
func formOf(route string) string {
	if IsInjection(route) {
		return "injection"
	}
	return "tablet"
}

// careColumns are the care reminder columns of the medicine (care's medication meta v1: every weekday, amount 1,
// until_date when it has an end, notified).
type careColumns struct {
	Subtitle       sql.NullString
	Notes          sql.NullString
	Recurrence     string
	RecurrenceTime sql.NullString
	StartsOn       civildate.NullDate
	EndsOn         civildate.NullDate
	Meta           db.NullRawJSON
}

func (in MedInput) careColumns() (careColumns, error) {
	meta := care.MedicationMeta{
		V: care.MetaVersion, Dose: in.Dose, Unit: in.Unit, Form: formOf(in.Route), Times: in.Times,
		Weekdays: slices.Clone(care.AllWeekdays), Amount: 1, Duration: care.DurationOngoing, Notify: true,
	}
	c := careColumns{StartsOn: nd(in.StartsOn)}
	if !in.EndsOn.IsZero() {
		meta.Duration = care.DurationUntilDate
		c.EndsOn = nd(in.EndsOn)
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return c, fmt.Errorf("ivf: encode medicine: %w", err)
	}
	c.Meta = db.NullRawJSON{V: raw, Valid: true}
	c.Subtitle = meta.Subtitle(in.SubtitleLocale)
	c.Recurrence = meta.Recurrence()
	c.RecurrenceTime = meta.RecurrenceTime()
	if in.Notes != nil {
		c.Notes = sql.NullString{String: *in.Notes, Valid: true}
	}
	return c, nil
}

func stockParams(in MedInput, countedAt time.Time) (sql.NullInt16, sql.NullString, sql.NullTime) {
	if in.StockUnits == nil {
		return sql.NullInt16{}, sql.NullString{}, sql.NullTime{}
	}
	return sql.NullInt16{Int16: int16(*in.StockUnits), Valid: true}, //nolint:gosec // ≤ MaxStockUnits
		sql.NullString{String: in.StockUnit, Valid: in.StockUnit != ""}, nt(countedAt)
}

// AddMed adds a medicine to the open cycle: a care medication reminder (counted against care's cap) plus its IVF
// row; the stock (if any) is counted now.
func (s *Service) AddMed(ctx context.Context, userID uint64, in MedInput, now time.Time) error {
	cols, err := in.careColumns()
	if err != nil {
		return err
	}
	return s.inTx(ctx, func(q *store.Queries, cq *carestore.Queries) error {
		cycle, err := activeCycle(ctx, q, userID, true)
		if err != nil {
			return err
		}
		if cycle == nil {
			return ErrNoCycle
		}
		all, err := cq.ListMedications(ctx, userID)
		if err != nil {
			return fmt.Errorf("ivf: count medications: %w", err)
		}
		if len(all) >= care.MaxMedications {
			return ErrMedicationLimit
		}
		at := ts(now)
		id, err := cq.InsertMedication(ctx, carestore.InsertMedicationParams{
			UserID: userID, Title: in.Name, Subtitle: cols.Subtitle, Notes: cols.Notes, Recurrence: cols.Recurrence,
			RecurrenceTime: cols.RecurrenceTime, StartsOn: cols.StartsOn, EndsOn: cols.EndsOn, IsActive: true,
			Meta: cols.Meta, CreatedAt: at, UpdatedAt: at,
		})
		if err != nil {
			return fmt.Errorf("ivf: insert medication: %w", err)
		}
		units, unit, counted := stockParams(in, at.Time)
		if _, err := q.InsertMed(ctx, store.InsertMedParams{
			UserID: userID, CycleID: cycle.ID, ReminderID: uint64(id), Role: in.Role, Route: in.Route, //nolint:gosec // G115: auto-increment id
			TriggerAt: nt(in.TriggerAt), StockUnits: units, StockUnit: unit, DosesPerUnit: uint16(in.DosesPerUnit), //nolint:gosec // ≤ MaxDosesPerUnit
			StockCountedAt: counted, Now: at,
		}); err != nil {
			return fmt.Errorf("ivf: insert medicine: %w", err)
		}
		return nil
	})
}

func loadMed(ctx context.Context, q *store.Queries, userID, id uint64) (Med, error) {
	row, err := q.GetMed(ctx, store.GetMedParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Med{}, ErrMedNotFound
	}
	if err != nil {
		return Med{}, fmt.Errorf("ivf: load medicine: %w", err)
	}
	return medFrom(row.IvfMed, row.Reminder), nil
}

// UpdateMed replaces the medicine (name, schedule, role, route, trigger time, notes, inventory). The care switch
// (is_active) stays as the user left it. The stock keeps its count when the units sent equal the units left today
// with the same unit and doses per unit (sending GET's values back changes nothing); otherwise it is recounted now.
func (s *Service) UpdateMed(ctx context.Context, userID, id uint64, in MedInput, now time.Time) error {
	cols, err := in.careColumns()
	if err != nil {
		return err
	}
	return s.inTx(ctx, func(q *store.Queries, cq *carestore.Queries) error {
		m, err := loadMed(ctx, q, userID, id)
		if err != nil {
			return err
		}
		at := ts(now)
		if err := cq.UpdateMedication(ctx, carestore.UpdateMedicationParams{
			Title: in.Name, Subtitle: cols.Subtitle, Notes: cols.Notes, Recurrence: cols.Recurrence,
			RecurrenceTime: cols.RecurrenceTime, StartsOn: cols.StartsOn, EndsOn: cols.EndsOn, IsActive: m.Care.Row.IsActive,
			Meta: cols.Meta, UpdatedAt: at, ID: m.ReminderID(), UserID: userID,
		}); err != nil {
			return fmt.Errorf("ivf: update medication: %w", err)
		}
		countedAt := at.Time
		if keep, err := keepStock(ctx, q, userID, m, in, civildate.InTehran(now)); err != nil {
			return err
		} else if keep {
			countedAt = m.Stock.CountedAt
			units := m.Stock.Units
			in.StockUnits = &units
		}
		units, unit, counted := stockParams(in, countedAt)
		if err := q.UpdateMed(ctx, store.UpdateMedParams{
			Role: in.Role, Route: in.Route, TriggerAt: nt(in.TriggerAt), StockUnits: units, StockUnit: unit,
			DosesPerUnit: uint16(in.DosesPerUnit), StockCountedAt: counted, Now: at, ID: m.ID, UserID: userID, //nolint:gosec // ≤ MaxDosesPerUnit
		}); err != nil {
			return fmt.Errorf("ivf: update medicine: %w", err)
		}
		return nil
	})
}

// keepStock reports whether the PUT carries the stock unchanged (units left today, unit, doses per unit).
func keepStock(ctx context.Context, q *store.Queries, userID uint64, m Med, in MedInput, today civildate.Date) (bool, error) {
	if m.Stock == nil || in.StockUnits == nil || in.StockUnit != m.Stock.Unit || in.DosesPerUnit != m.Stock.DosesPerUnit {
		return false, nil
	}
	used, err := q.CountIntakesSince(ctx, store.CountIntakesSinceParams{
		UserID: userID, ReminderID: m.ReminderID(), Since: m.Stock.CountedAt,
	})
	if err != nil {
		return false, fmt.Errorf("ivf: count doses: %w", err)
	}
	return m.Inventory(int(used), today).UnitsLeft == *in.StockUnits, nil
}

// DeleteMed removes the medicine: its care reminder goes, and with it (FK cascade) the IVF row, its intakes and
// its dose logs.
func (s *Service) DeleteMed(ctx context.Context, userID, id uint64) error {
	q := store.New(s.conn)
	m, err := loadMed(ctx, q, userID, id)
	if err != nil {
		return err
	}
	if _, err := carestore.New(s.conn).DeleteMedication(ctx, carestore.DeleteMedicationParams{ID: m.ReminderID(), UserID: userID}); err != nil {
		return fmt.Errorf("ivf: delete medication: %w", err)
	}
	return nil
}

// LogDose marks a dose taken — the care intake (so /care/today agrees) plus the IVF dose log with the injection
// site. The slot must be one of the medicine's times and the medicine scheduled on date; a site must be an active
// catalog site and only goes with an injection.
func (s *Service) LogDose(ctx context.Context, userID, id uint64, date civildate.Date, slot, site string, now time.Time) error {
	if site != "" {
		if err := s.checkCode(ctx, GroupSites, "site", site); err != nil {
			return err
		}
	}
	return s.inTx(ctx, func(q *store.Queries, cq *carestore.Queries) error {
		m, err := loadMed(ctx, q, userID, id)
		if err != nil {
			return err
		}
		switch {
		case !slices.Contains(m.Care.Meta.Times, slot):
			return fieldErr("slot", "slot_unknown")
		case !m.Care.Covers(date):
			return fieldErr("date", "date_not_scheduled")
		case site != "" && !IsInjection(m.Route):
			return fieldErr("site", "site_not_injection")
		}
		at := ts(now)
		if _, err := cq.InsertIntake(ctx, carestore.InsertIntakeParams{
			UserID: userID, ReminderID: m.ReminderID(), IntakeDate: date, Slot: slot, TakenAt: at.Time,
			CreatedAt: at, UpdatedAt: at,
		}); err != nil {
			return fmt.Errorf("ivf: insert intake: %w", err)
		}
		if err := q.UpsertDoseLog(ctx, store.UpsertDoseLogParams{
			UserID: userID, IvfMedID: m.ID, DoseDate: date, Slot: slot,
			Site: sql.NullString{String: site, Valid: site != ""}, Now: at,
		}); err != nil {
			return fmt.Errorf("ivf: save dose: %w", err)
		}
		return nil
	})
}

// UnlogDose undoes a dose (care intake and site); a dose not taken is a no-op.
func (s *Service) UnlogDose(ctx context.Context, userID, id uint64, date civildate.Date, slot string) error {
	return s.inTx(ctx, func(q *store.Queries, cq *carestore.Queries) error {
		m, err := loadMed(ctx, q, userID, id)
		if err != nil {
			return err
		}
		if _, err := cq.DeleteIntake(ctx, carestore.DeleteIntakeParams{
			ReminderID: m.ReminderID(), UserID: userID, IntakeDate: date, Slot: slot,
		}); err != nil {
			return fmt.Errorf("ivf: delete intake: %w", err)
		}
		if err := q.DeleteDoseLog(ctx, store.DeleteDoseLogParams{UserID: userID, IvfMedID: m.ID, DoseDate: date, Slot: slot}); err != nil {
			return fmt.Errorf("ivf: delete dose: %w", err)
		}
		return nil
	})
}

// openCycleOn is the open cycle when date is inside it (from its start); ErrNoCycle / a 422 on `date` otherwise.
func openCycleOn(ctx context.Context, q *store.Queries, userID uint64, date civildate.Date) (*Cycle, error) {
	c, err := activeCycle(ctx, q, userID, false)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNoCycle
	}
	if date.Before(c.StartedOn) {
		return nil, fieldErr("date", "before_cycle_start")
	}
	return c, nil
}

// ScanInput is a validated PUT /ivf/scans/{date}. Decimal fields are formatted strings ("" = not recorded).
type ScanInput struct {
	Right, Left   Ovary
	EndometriumMM string
	E2            string
	E2Unit        string
	Notes         string
}

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

// SaveScan records (or replaces) the open cycle's scan of date.
func (s *Service) SaveScan(ctx context.Context, userID uint64, date civildate.Date, in ScanInput, now time.Time) error {
	q := store.New(s.conn)
	c, err := openCycleOn(ctx, q, userID, date)
	if err != nil {
		return err
	}
	u8 := func(n int) uint8 { return uint8(n) } //nolint:gosec // ≤ MaxFollicles
	if err := q.UpsertScan(ctx, store.UpsertScanParams{
		UserID: userID, CycleID: c.ID, ScanDate: date,
		RightLt10: u8(in.Right[0]), Right1014: u8(in.Right[1]), Right1517: u8(in.Right[2]), Right18Plus: u8(in.Right[3]),
		LeftLt10: u8(in.Left[0]), Left1014: u8(in.Left[1]), Left1517: u8(in.Left[2]), Left18Plus: u8(in.Left[3]),
		EndometriumMm: ns(in.EndometriumMM), E2: ns(in.E2), E2Unit: ns(in.E2Unit), Notes: ns(in.Notes), Now: ts(now),
	}); err != nil {
		return fmt.Errorf("ivf: save scan: %w", err)
	}
	return nil
}

// DeleteScan removes the open cycle's scan of date (no-op when none).
func (s *Service) DeleteScan(ctx context.Context, userID uint64, date civildate.Date) error {
	q := store.New(s.conn)
	c, err := activeCycle(ctx, q, userID, false)
	if err != nil {
		return err
	}
	if c == nil {
		return ErrNoCycle
	}
	if _, err := q.DeleteScan(ctx, store.DeleteScanParams{UserID: userID, CycleID: c.ID, ScanDate: date}); err != nil {
		return fmt.Errorf("ivf: delete scan: %w", err)
	}
	return nil
}

// SaveMood records the two-week-wait mood of date ("" clears it).
func (s *Service) SaveMood(ctx context.Context, userID uint64, date civildate.Date, mood string, now time.Time) error {
	q := store.New(s.conn)
	c, err := openCycleOn(ctx, q, userID, date)
	if err != nil {
		return err
	}
	if mood == "" {
		if err := q.DeleteTWWLog(ctx, store.DeleteTWWLogParams{UserID: userID, CycleID: c.ID, LogDate: date}); err != nil {
			return fmt.Errorf("ivf: delete mood: %w", err)
		}
		return nil
	}
	if err := q.UpsertTWWLog(ctx, store.UpsertTWWLogParams{UserID: userID, CycleID: c.ID, LogDate: date, Mood: mood, Now: ts(now)}); err != nil {
		return fmt.Errorf("ivf: save mood: %w", err)
	}
	return nil
}
