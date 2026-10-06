package menopause

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
	"github.com/ritme/backend-go/internal/menopause/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Treatment & care (CB-MENO-03, nbl_Meno_Treatment): the user's HRT, supplements and lifestyle goals. Names and
// doses are what she types — the app never prescribes («شروع، قطع یا تغییر دوز هر دارو فقط با نظر پزشک»).
//
// Reminders are care reminders (M3), like CB-IVF-01 / CB-CONTRA-01: every HRT or supplement item IS a care medication
// reminder (treatment_items.reminder_id → reminders, type medication) at its schedule's time, so it shows in /care,
// in /care/today and in the health record's medications; `remind` is that reminder's notify switch. A day ticked in
// /care counts as taken here too, and an intake logged here ticks the care dose. Lifestyle goals have no reminder.

// More treatment kinds, schedules and units (KindHRT, KindLifestyle, ScheduleWeekly and UnitMinutes are in
// treatment.go).
const (
	KindSupplement = "supplement"

	ScheduleMorning = "morning"
	ScheduleNoon    = "noon"
	ScheduleEvening = "evening"
	ScheduleNight   = "night"

	UnitSessions = "sessions"
)

// Kinds are the treatment kinds, in screen order (هورمون‌درمانی · مکمل‌ها · برنامه سبک زندگی).
var Kinds = []string{KindHRT, KindSupplement, KindLifestyle}

// Schedules are the item schedules (hrt / supplement).
var Schedules = []string{ScheduleMorning, ScheduleNoon, ScheduleEvening, ScheduleNight, ScheduleWeekly}

// GoalUnits are the lifestyle goal units.
var GoalUnits = []string{UnitSessions, UnitMinutes}

// SideEffectCodes are the side effects the board offers («عوارض احتمالی را ثبت کن»); labels are UI strings.
var SideEffectCodes = []string{"breast_tenderness", "spotting", "headache", "bloating", "mood_change"}

// ScheduleTimes is the care reminder time (Tehran "HH:MM") of each schedule; the user moves it in /care. A weekly
// item reminds on the weekday it started.
var ScheduleTimes = map[string]string{
	ScheduleMorning: "08:00",
	ScheduleNoon:    "13:00",
	ScheduleEvening: "18:00",
	ScheduleNight:   "22:00",
	ScheduleWeekly:  "09:00",
}

// Treatment limits.
const (
	// MaxTreatmentItems caps a user's items (active and stopped).
	MaxTreatmentItems = 30
	// MaxNameLen / MaxDoseLen are the column widths.
	MaxNameLen = 120
	MaxDoseLen = 120
	// IntakeBackfillDays: an intake or a side effect can be logged for today and this many days back.
	IntakeBackfillDays = 30
	// MaxIntakeMinutes / MaxIntakeSessions bound one day's lifestyle amount.
	MaxIntakeMinutes  = 600
	MaxIntakeSessions = 10
	// MaxGoalMinutes / MaxGoalSessions bound a weekly goal.
	MaxGoalMinutes  = 3000
	MaxGoalSessions = 21
	// DefaultReviewAfterMonths: «معمولاً ۳ ماه بعد از شروع، اثر و عوارض بررسی می‌شود» — the suggested doctor review
	// of an HRT item without a review date (catalog meno_tips hrt_review meta.review_after_months overrides it)
	// [needs clinical review].
	DefaultReviewAfterMonths = 3
)

// Treatment errors.
var (
	ErrItemNotFound    = errors.New("menopause: treatment item not found")
	ErrItemLimit       = errors.New("menopause: treatment item limit reached")
	ErrMedicationLimit = errors.New("menopause: care medication limit reached")
	ErrItemInactive    = errors.New("menopause: treatment item not active on that day")
)

// ItemInput is a validated POST /menopause/treatment/items or PUT …/{id} (a full replace; kind is fixed on create).
type ItemInput struct {
	Kind       string
	Name       string
	Dose       string // "" = none
	Schedule   string // hrt / supplement; "" for lifestyle
	Form       string // care medication form (hrt / supplement)
	StartedOn  civildate.Date
	ReviewOn   civildate.Date
	StoppedOn  civildate.Date
	WeeklyGoal int    // lifestyle
	GoalUnit   string // lifestyle
	Remind     bool   // the care reminder's notify switch (hrt / supplement)
}

func nd(d civildate.Date) civildate.NullDate { return civildate.NullDate{Date: d, Valid: !d.IsZero()} }

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

// HasReminder reports whether the kind is a care medication (hrt and supplements; lifestyle goals are not).
func HasReminder(kind string) bool { return kind != KindLifestyle }

// Conn is a database that can start a transaction (*sql.DB).
type Conn interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries, cq *carestore.Queries) error) error {
	conn, ok := s.db.(Conn)
	if !ok { // a transaction already (tests): run on it
		return fn(s.q, carestore.New(s.db))
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("menopause: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(store.New(tx), carestore.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("menopause: commit: %w", err)
	}
	return nil
}

// medicationMeta is the care medication meta (v1) of an hrt / supplement item: the dose as typed, one slot at the
// schedule's time, every weekday (weekly: the start weekday), ongoing, notify = remind.
func (in ItemInput) medicationMeta(today civildate.Date) care.MedicationMeta {
	start := in.StartedOn
	if start.IsZero() {
		start = today
	}
	meta := care.MedicationMeta{
		V: care.MetaVersion, Form: in.Form, Times: []string{ScheduleTimes[in.Schedule]},
		Weekdays: slices.Clone(care.AllWeekdays), Amount: 1, Duration: care.DurationOngoing, Notify: in.Remind,
	}
	if meta.Form == "" {
		meta.Form = care.Forms[0]
	}
	if in.Dose != "" {
		d := in.Dose
		meta.Dose = &d
	}
	if in.Schedule == ScheduleWeekly {
		meta.Weekdays = []int{care.SaturdayWeekday(start)}
	}
	return meta
}

// careColumns are the reminders columns of the item's care medication. A stopped item's reminder is switched off.
type careColumns struct {
	Subtitle       sql.NullString
	Recurrence     string
	RecurrenceTime sql.NullString
	StartsOn       civildate.NullDate
	IsActive       bool
	Meta           db.NullRawJSON
}

func (in ItemInput) careColumns(today civildate.Date, locale string) (careColumns, error) {
	meta := in.medicationMeta(today)
	raw, err := json.Marshal(meta)
	if err != nil {
		return careColumns{}, fmt.Errorf("menopause: encode medication: %w", err)
	}
	start := in.StartedOn
	if start.IsZero() {
		start = today
	}
	return careColumns{
		Subtitle: meta.Subtitle(locale), Recurrence: meta.Recurrence(), RecurrenceTime: meta.RecurrenceTime(),
		StartsOn: nd(start), IsActive: in.StoppedOn.IsZero() || in.StoppedOn.After(today),
		Meta: db.NullRawJSON{V: raw, Valid: true},
	}, nil
}

func (in ItemInput) goal() (sql.NullInt16, sql.NullString) {
	if in.Kind != KindLifestyle || in.WeeklyGoal <= 0 {
		return sql.NullInt16{}, sql.NullString{}
	}
	return sql.NullInt16{Int16: int16(in.WeeklyGoal), Valid: true}, ns(in.GoalUnit) //nolint:gosec // ≤ MaxGoalMinutes
}

// GetItem is one of the user's items (ErrItemNotFound for a foreign or missing id).
func (s *Service) GetItem(ctx context.Context, userID, id uint64) (store.TreatmentItem, error) {
	it, err := s.q.GetTreatmentItem(ctx, store.GetTreatmentItemParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return store.TreatmentItem{}, ErrItemNotFound
	}
	if err != nil {
		return store.TreatmentItem{}, fmt.Errorf("menopause: load treatment item: %w", err)
	}
	return it, nil
}

// CreateItem adds an item at the end of its kind's list; an hrt / supplement item gets its care medication reminder
// (counted against care's cap). subtitleLocale is the language of the legacy reminders.subtitle (the default one).
func (s *Service) CreateItem(ctx context.Context, userID uint64, in ItemInput, now time.Time, subtitleLocale string,
) (uint64, error) {
	today := civildate.InTehran(now)
	var id uint64
	err := s.inTx(ctx, func(q *store.Queries, cq *carestore.Queries) error {
		n, err := q.CountTreatmentItems(ctx, userID)
		if err != nil {
			return fmt.Errorf("menopause: count treatment: %w", err)
		}
		if n >= MaxTreatmentItems {
			return ErrItemLimit
		}
		var reminder sql.NullInt64
		if HasReminder(in.Kind) {
			rid, err := insertMedication(ctx, cq, userID, in, today, now, subtitleLocale)
			if err != nil {
				return err
			}
			reminder = sql.NullInt64{Int64: int64(rid), Valid: true} //nolint:gosec // auto-increment id
		}
		order, err := q.NextTreatmentSortOrder(ctx, store.NextTreatmentSortOrderParams{UserID: userID, Kind: in.Kind})
		if err != nil {
			return fmt.Errorf("menopause: sort order: %w", err)
		}
		goal, unit := in.goal()
		newID, err := q.InsertTreatmentItem(ctx, store.InsertTreatmentItemParams{
			UserID: userID, Kind: in.Kind, Name: in.Name, Dose: ns(in.Dose), Schedule: ns(in.Schedule),
			StartedOn: nd(in.StartedOn), ReviewOn: nd(in.ReviewOn), WeeklyGoal: goal, GoalUnit: unit,
			StoppedOn: nd(in.StoppedOn), ReminderID: reminder, SortOrder: int32(order), Now: tehranNow(now), //nolint:gosec // ≤ MaxTreatmentItems
		})
		if err != nil {
			return fmt.Errorf("menopause: insert treatment item: %w", err)
		}
		id = uint64(newID) //nolint:gosec // auto-increment id
		return nil
	})
	return id, err
}

func insertMedication(ctx context.Context, cq *carestore.Queries, userID uint64, in ItemInput, today civildate.Date,
	now time.Time, locale string,
) (uint64, error) {
	all, err := cq.ListMedications(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("menopause: count medications: %w", err)
	}
	if len(all) >= care.MaxMedications {
		return 0, ErrMedicationLimit
	}
	cols, err := in.careColumns(today, locale)
	if err != nil {
		return 0, err
	}
	at := tehranNow(now)
	id, err := cq.InsertMedication(ctx, carestore.InsertMedicationParams{
		UserID: userID, Title: in.Name, Subtitle: cols.Subtitle, Recurrence: cols.Recurrence,
		RecurrenceTime: cols.RecurrenceTime, StartsOn: cols.StartsOn, IsActive: cols.IsActive, Meta: cols.Meta,
		CreatedAt: at, UpdatedAt: at,
	})
	if err != nil {
		return 0, fmt.Errorf("menopause: insert medication: %w", err)
	}
	return uint64(id), nil //nolint:gosec // auto-increment id
}

// UpdateItem replaces the item's editable fields and syncs its care reminder: updated in place (the care switch
// stays as she left it unless the item is stopped), or created again when she deleted it in /care.
func (s *Service) UpdateItem(ctx context.Context, userID, id uint64, in ItemInput, now time.Time, subtitleLocale string,
) error {
	today := civildate.InTehran(now)
	return s.inTx(ctx, func(q *store.Queries, cq *carestore.Queries) error {
		it, err := q.GetTreatmentItem(ctx, store.GetTreatmentItemParams{ID: id, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrItemNotFound
		}
		if err != nil {
			return fmt.Errorf("menopause: load treatment item: %w", err)
		}
		in.Kind = it.Kind
		reminder := it.ReminderID
		if HasReminder(in.Kind) {
			if reminder, err = syncMedication(ctx, cq, userID, it.ReminderID, !activeOn(it, today) && it.StoppedOn.Valid,
				in, today, now, subtitleLocale); err != nil {
				return err
			}
		}
		goal, unit := in.goal()
		if err := q.UpdateTreatmentItem(ctx, store.UpdateTreatmentItemParams{
			Name: in.Name, Dose: ns(in.Dose), Schedule: ns(in.Schedule), StartedOn: nd(in.StartedOn),
			ReviewOn: nd(in.ReviewOn), WeeklyGoal: goal, GoalUnit: unit, StoppedOn: nd(in.StoppedOn),
			ReminderID: reminder, Now: tehranNow(now), ID: id, UserID: userID,
		}); err != nil {
			return fmt.Errorf("menopause: update treatment item: %w", err)
		}
		return nil
	})
}

func syncMedication(ctx context.Context, cq *carestore.Queries, userID uint64, current sql.NullInt64, wasStopped bool,
	in ItemInput, today civildate.Date, now time.Time, locale string,
) (sql.NullInt64, error) {
	if current.Valid {
		row, err := cq.GetMedication(ctx, carestore.GetMedicationParams{ID: uint64(current.Int64), UserID: userID}) //nolint:gosec // FK id
		switch {
		case err == nil:
			cols, err := in.careColumns(today, locale)
			if err != nil {
				return current, err
			}
			// stopped → off; restarted → on again; otherwise her care switch stays.
			active := cols.IsActive && (row.IsActive || wasStopped)
			if err := cq.UpdateMedication(ctx, carestore.UpdateMedicationParams{
				Title: in.Name, Subtitle: cols.Subtitle, Notes: row.Notes, Recurrence: cols.Recurrence,
				RecurrenceTime: cols.RecurrenceTime, StartsOn: cols.StartsOn, EndsOn: row.EndsOn, IsActive: active,
				Meta: cols.Meta, UpdatedAt: tehranNow(now), ID: row.ID, UserID: userID,
			}); err != nil {
				return current, fmt.Errorf("menopause: update medication: %w", err)
			}
			return current, nil
		case !errors.Is(err, sql.ErrNoRows):
			return current, fmt.Errorf("menopause: load medication: %w", err)
		}
	}
	id, err := insertMedication(ctx, cq, userID, in, today, now, locale)
	if err != nil {
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: int64(id), Valid: true}, nil //nolint:gosec // auto-increment id
}

// DeleteItem removes the item (its intakes cascade) and its care medication reminder.
func (s *Service) DeleteItem(ctx context.Context, userID, id uint64) error {
	return s.inTx(ctx, func(q *store.Queries, cq *carestore.Queries) error {
		it, err := q.GetTreatmentItem(ctx, store.GetTreatmentItemParams{ID: id, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrItemNotFound
		}
		if err != nil {
			return fmt.Errorf("menopause: load treatment item: %w", err)
		}
		if _, err := q.DeleteTreatmentItem(ctx, store.DeleteTreatmentItemParams{ID: id, UserID: userID}); err != nil {
			return fmt.Errorf("menopause: delete treatment item: %w", err)
		}
		if it.ReminderID.Valid {
			if _, err := cq.DeleteMedication(ctx, carestore.DeleteMedicationParams{
				ID: uint64(it.ReminderID.Int64), UserID: userID, //nolint:gosec // FK id
			}); err != nil {
				return fmt.Errorf("menopause: delete medication: %w", err)
			}
		}
		return nil
	})
}

// medicationSlots are the care dose slots of the item's reminder (nil without one).
func medicationSlots(ctx context.Context, cq *carestore.Queries, userID uint64, it store.TreatmentItem) ([]string, error) {
	if !it.ReminderID.Valid {
		return nil, nil
	}
	row, err := cq.GetMedication(ctx, carestore.GetMedicationParams{ID: uint64(it.ReminderID.Int64), UserID: userID}) //nolint:gosec // FK id
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("menopause: load medication: %w", err)
	}
	return care.ParseMedication(row).Meta.Times, nil
}

// LogIntake marks the item taken on day (amount = minutes or sessions for lifestyle goals; 0 = none) and ticks its
// care doses of that day. Logging the day again replaces the amount.
func (s *Service) LogIntake(ctx context.Context, userID, id uint64, day civildate.Date, amount int, now time.Time) error {
	return s.inTx(ctx, func(q *store.Queries, cq *carestore.Queries) error {
		it, err := q.GetTreatmentItem(ctx, store.GetTreatmentItemParams{ID: id, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrItemNotFound
		}
		if err != nil {
			return fmt.Errorf("menopause: load treatment item: %w", err)
		}
		if !activeOn(it, day) {
			return ErrItemInactive
		}
		var amt sql.NullInt16
		if it.Kind == KindLifestyle && amount > 0 {
			amt = sql.NullInt16{Int16: int16(amount), Valid: true} //nolint:gosec // ≤ MaxIntakeMinutes
		}
		at := tehranNow(now)
		if err := q.UpsertTreatmentIntake(ctx, store.UpsertTreatmentIntakeParams{
			UserID: userID, TreatmentItemID: id, IntakeDate: day, Amount: amt, TakenAt: at.Time, Now: at,
		}); err != nil {
			return fmt.Errorf("menopause: log intake: %w", err)
		}
		slots, err := medicationSlots(ctx, cq, userID, it)
		if err != nil {
			return err
		}
		for _, slot := range slots {
			if _, err := cq.InsertIntake(ctx, carestore.InsertIntakeParams{
				UserID: userID, ReminderID: uint64(it.ReminderID.Int64), IntakeDate: day, Slot: slot, //nolint:gosec // FK id
				TakenAt: at.Time, CreatedAt: at, UpdatedAt: at,
			}); err != nil {
				return fmt.Errorf("menopause: tick care dose: %w", err)
			}
		}
		return nil
	})
}

// UnlogIntake removes the day's intake and unticks its care doses of that day (idempotent).
func (s *Service) UnlogIntake(ctx context.Context, userID, id uint64, day civildate.Date) error {
	return s.inTx(ctx, func(q *store.Queries, cq *carestore.Queries) error {
		it, err := q.GetTreatmentItem(ctx, store.GetTreatmentItemParams{ID: id, UserID: userID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrItemNotFound
		}
		if err != nil {
			return fmt.Errorf("menopause: load treatment item: %w", err)
		}
		if _, err := q.DeleteTreatmentIntake(ctx, store.DeleteTreatmentIntakeParams{
			UserID: userID, TreatmentItemID: id, IntakeDate: day,
		}); err != nil {
			return fmt.Errorf("menopause: delete intake: %w", err)
		}
		slots, err := medicationSlots(ctx, cq, userID, it)
		if err != nil {
			return err
		}
		for _, slot := range slots {
			if _, err := cq.DeleteIntake(ctx, carestore.DeleteIntakeParams{
				ReminderID: uint64(it.ReminderID.Int64), UserID: userID, IntakeDate: day, Slot: slot, //nolint:gosec // FK id
			}); err != nil {
				return fmt.Errorf("menopause: untick care dose: %w", err)
			}
		}
		return nil
	})
}

// SideEffectsInput is a validated PUT /menopause/treatment/side-effects/{date}.
type SideEffectsInput struct {
	Codes  []string // SideEffectCodes order; empty clears the day
	ItemID uint64   // 0 = not tied to an item
}

// SaveSideEffects replaces the day's side effects (ErrItemNotFound when the item is not hers).
func (s *Service) SaveSideEffects(ctx context.Context, userID uint64, day civildate.Date, in SideEffectsInput,
	now time.Time,
) error {
	return s.inTx(ctx, func(q *store.Queries, _ *carestore.Queries) error {
		var item sql.NullInt64
		if in.ItemID > 0 {
			if _, err := q.GetTreatmentItem(ctx, store.GetTreatmentItemParams{ID: in.ItemID, UserID: userID}); errors.Is(err, sql.ErrNoRows) {
				return ErrItemNotFound
			} else if err != nil {
				return fmt.Errorf("menopause: load treatment item: %w", err)
			}
			item = sql.NullInt64{Int64: int64(in.ItemID), Valid: true} //nolint:gosec // auto-increment id
		}
		if err := q.DeleteSideEffectsOn(ctx, store.DeleteSideEffectsOnParams{UserID: userID, LogDate: day}); err != nil {
			return fmt.Errorf("menopause: clear side effects: %w", err)
		}
		for _, code := range in.Codes {
			if err := q.InsertSideEffect(ctx, store.InsertSideEffectParams{
				UserID: userID, TreatmentItemID: item, LogDate: day, Code: code, Now: tehranNow(now),
			}); err != nil {
				return fmt.Errorf("menopause: save side effect: %w", err)
			}
		}
		return nil
	})
}
