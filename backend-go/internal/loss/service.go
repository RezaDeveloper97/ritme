package loss

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/care"
	carestore "github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/companion"
	companionstore "github.com/ritme/backend-go/internal/companion/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/loss/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/pregnancy"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// ErrNoLoss: the user has recorded no loss (the follow-up routes answer 404).
var ErrNoLoss = errors.New("loss: none recorded")

// ErrNoteUnavailable: private notes are switched off (production without PRIVATE_NOTE_KEY → 503).
var ErrNoteUnavailable = errors.New("loss: private notes unavailable")

// ModeSwitcher stores a life-stage mode for the user (profile.OnboardingHandlers.SwitchLifeMode).
type ModeSwitcher interface {
	SwitchLifeMode(ctx context.Context, u *auth.User, mode enums.LifeMode, now time.Time) error
}

// Deps are the service's collaborators.
type Deps struct {
	DB      *sql.DB
	Catalog *catalog.Reader
	Modes   ModeSwitcher
	Notes   *NoteBox
	Logger  *slog.Logger
}

// Service is the loss domain.
type Service struct {
	db     *sql.DB
	cat    *catalog.Reader
	modes  ModeSwitcher
	notes  *NoteBox
	comp   *companion.Service
	logger *slog.Logger
}

// NewService wires the service.
func NewService(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Service{
		db: d.DB, cat: d.Catalog, modes: d.Modes, notes: d.Notes, logger: logger,
		comp: companion.NewService(d.DB, clock.Real{}, companion.Options{}),
	}
}

func nt(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

func (s *Service) inTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("loss: begin: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("loss: commit: %w", err)
	}
	return nil
}

// latest is the user's newest loss (ErrNoLoss when none).
func latest(ctx context.Context, q store.Querier, userID uint64) (store.PregnancyLoss, error) {
	l, err := q.GetLatestLoss(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return l, ErrNoLoss
	}
	if err != nil {
		return l, fmt.Errorf("loss: latest: %w", err)
	}
	return l, nil
}

// Record stores a loss and stops pregnancy content in one transaction: pregnancy mode off and open pregnancy alerts
// closed (pregnancy.EndForLoss), pregnancy-only reminders paused (medications «تا پایان بارداری» switched off,
// upcoming pregnancy visits cancelled). A second POST on the same Tehran day corrects that event instead of
// recording another loss. Afterwards the life-stage mode leaves pregnancy (→ cycle) and, when asked, companions with
// the pregnancy grant get the one-line notice (best effort, once per loss). Reports whether a new row was created.
func (s *Service) Record(ctx context.Context, u *auth.User, in RecordInput, now time.Time, langs i18n.Languages) (bool, error) {
	today := civildate.InTehran(now)
	var (
		created, wasPregnant, notified bool
		lossID                         uint64
	)
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		q := store.New(tx)
		if _, err := q.LockLossOwner(ctx, u.ID); err != nil { // concurrent first POSTs wait here
			return fmt.Errorf("loss: lock: %w", err)
		}
		prev, err := latest(ctx, q, u.ID)
		if err != nil && !errors.Is(err, ErrNoLoss) {
			return err
		}
		hasPrev := err == nil
		sameDay := hasPrev && prev.CreatedAt.Valid && civildate.InTehran(prev.CreatedAt.Time) == today
		if wasPregnant, err = pregnancy.EndForLoss(ctx, pregstore.New(tx), u.ID, now); err != nil {
			return err
		}
		paused, err := pauseReminders(ctx, q, u.ID, now)
		if err != nil {
			return err
		}
		if sameDay {
			paused = mergeIDs(decodeIDs(prev.PausedReminders), paused)
			lossID, notified = prev.ID, prev.CompanionNotifiedAt.Valid
			if !in.HasType {
				in.Type = prev.LossType
			}
			if !in.HasDate {
				in.OccurredOn = prev.OccurredOn
			}
			return q.UpdateLossEvent(ctx, store.UpdateLossEventParams{
				LossType: in.Type, OccurredOn: in.OccurredOn, NotifyCompanion: in.NotifyCompanion || prev.NotifyCompanion,
				Now: nt(now), PausedReminders: encodeIDs(paused), ID: prev.ID, UserID: u.ID,
			})
		}
		if hasPrev && prev.PrivateNote.Valid { // a new loss: the earlier record's private note is not kept around
			if err := q.SetLossNote(ctx, store.SetLossNoteParams{Now: nt(now), ID: prev.ID, UserID: u.ID}); err != nil {
				return fmt.Errorf("loss: clear previous note: %w", err)
			}
		}
		id, err := q.InsertLoss(ctx, store.InsertLossParams{
			UserID: u.ID, LossType: in.Type, OccurredOn: in.OccurredOn, NotifyCompanion: in.NotifyCompanion,
			Now: nt(now), PausedReminders: encodeIDs(paused),
		})
		if err != nil {
			return fmt.Errorf("loss: insert: %w", err)
		}
		lossID, created = uint64(id), true //nolint:gosec // auto-increment id
		return nil
	})
	if err != nil {
		return false, err
	}
	if wasPregnant && s.modes != nil {
		if err := s.modes.SwitchLifeMode(ctx, u, enums.LifeModeCycle, now); err != nil {
			return created, err
		}
	}
	if in.NotifyCompanion && !notified {
		if err := s.notifyCompanions(ctx, u.ID, lossID, now, langs); err != nil {
			// Best effort: the loss is stored and pregnancy content is stopped. No user or health detail is logged.
			s.logger.ErrorContext(ctx, "companion notice failed", slog.String("error", err.Error()))
		}
	}
	return created, nil
}

// pauseReminders switches off the pregnancy-only reminders (see ListPregnancyReminders) and returns their ids.
func pauseReminders(ctx context.Context, q store.Querier, userID uint64, now time.Time) ([]uint64, error) {
	rows, err := q.ListPregnancyReminders(ctx, store.ListPregnancyRemindersParams{UserID: userID, Now: nt(now)})
	if err != nil {
		return nil, fmt.Errorf("loss: pregnancy reminders: %w", err)
	}
	ids := make([]uint64, 0, len(rows))
	for _, r := range rows {
		if r.Type == care.TypeAppointment {
			err = q.CancelAppointmentReminder(ctx, store.CancelAppointmentReminderParams{Now: nt(now), ID: r.ID, UserID: userID})
		} else {
			err = q.PauseMedicationReminder(ctx, store.PauseMedicationReminderParams{Now: nt(now), ID: r.ID, UserID: userID})
		}
		if err != nil {
			return nil, fmt.Errorf("loss: pause reminder: %w", err)
		}
		ids = append(ids, r.ID)
	}
	return ids, nil
}

func decodeIDs(raw rootdb.NullRawJSON) []uint64 {
	var ids []uint64
	if raw.Valid {
		_ = json.Unmarshal(raw.V, &ids)
	}
	return ids
}

func mergeIDs(a, b []uint64) []uint64 {
	out := slices.Clone(a)
	for _, id := range b {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

func encodeIDs(ids []uint64) rootdb.NullRawJSON {
	if len(ids) == 0 {
		return rootdb.NullRawJSON{}
	}
	b, _ := json.Marshal(ids)
	return rootdb.NullRawJSON{V: b, Valid: true}
}

// notifyCompanions writes the one-line notice (catalog loss_support/companion_notice, one entry per active language)
// to the inbox of every active companion holding the pregnancy grant (view or edit). Nothing else about the loss is
// shared: no type, date or follow-up. Marks the loss notified when at least one notice went out.
func (s *Service) notifyCompanions(ctx context.Context, ownerID, lossID uint64, now time.Time, langs i18n.Languages) error {
	item, ok, err := s.catalogItem(ctx, GroupSupport, ItemCompanionNotice)
	if err != nil || !ok {
		return err // no copy (deactivated by an admin) → nothing is sent
	}
	links, err := s.comp.ListForOwner(ctx, ownerID)
	if err != nil {
		return err
	}
	names, err := s.comp.Names(ctx, ownerID)
	if err != nil {
		return err
	}
	titles, someone := translations(item.Title), map[string]string{}
	var meta struct {
		Someone map[string]string `json:"someone"`
	}
	if len(item.Meta) > 0 && json.Unmarshal(item.Meta, &meta) == nil && meta.Someone != nil {
		someone = meta.Someone
	}
	def := langs.DefaultCode()
	title := map[string]string{}
	for _, code := range langs.Codes() {
		name := names[ownerID]
		if name == "" {
			name = pickText(someone, code, def)
		}
		title[code] = strings.ReplaceAll(pickText(titles, code, def), "{name}", name)
	}
	rawTitle, err := json.Marshal(title)
	if err != nil {
		return fmt.Errorf("loss: notice title: %w", err)
	}
	cq := companionstore.New(s.db)
	sent := false
	for _, l := range links {
		if l.Status != companion.StatusActive || l.CompanionUserID == 0 || !l.Grants.Of(companion.SectionPregnancy).CanRead() {
			continue
		}
		data, err := json.Marshal(map[string]any{"companion_id": l.ID, "event": companion.EventPregnancyNotContinuing})
		if err != nil {
			return fmt.Errorf("loss: notice data: %w", err)
		}
		if err := cq.InsertUserNotification(ctx, companionstore.InsertUserNotificationParams{
			UserID: l.CompanionUserID, Type: companion.NotificationType, Title: json.RawMessage(rawTitle),
			ActionUrl: sql.NullString{String: CompanionNoticeAction, Valid: true},
			Data:      rootdb.NullRawJSON{V: data, Valid: true}, Now: nt(now),
		}); err != nil {
			return fmt.Errorf("loss: notice: %w", err)
		}
		sent = true
	}
	if !sent {
		return nil
	}
	if err := store.New(s.db).MarkCompanionNotified(ctx, store.MarkCompanionNotifiedParams{Now: nt(now), ID: lossID, UserID: ownerID}); err != nil {
		return fmt.Errorf("loss: mark notified: %w", err)
	}
	return nil
}

// catalogItem is one active item of a catalog group (ok = false when absent or inactive).
func (s *Service) catalogItem(ctx context.Context, group, code string) (catalog.Item, bool, error) {
	if s.cat == nil {
		return catalog.Item{}, false, nil
	}
	items, err := s.cat.Items(ctx, group)
	if err != nil {
		return catalog.Item{}, false, err
	}
	for _, it := range items {
		if it.Code == code {
			return it, true, nil
		}
	}
	return catalog.Item{}, false, nil
}

func translations(raw json.RawMessage) map[string]string {
	m := map[string]string{}
	_ = json.Unmarshal(raw, &m)
	return m
}

// pickText is the code's translation, else the default language's, else any non-empty one (sorted for stability).
func pickText(m map[string]string, code, def string) string {
	if s := m[code]; s != "" {
		return s
	}
	if s := m[def]; s != "" {
		return s
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		if m[k] != "" {
			return m[k]
		}
	}
	return ""
}

// State is everything GET /loss shows.
type State struct {
	Loss          *store.PregnancyLoss
	Count         int64
	BleedingToday string
	Moods         []store.ListLossMoodsSinceRow
	Beta, Visit   *care.Appointment
	Today         civildate.Date
	Now           time.Time
}

// State loads the user's loss state at now.
func (s *Service) State(ctx context.Context, userID uint64, now time.Time) (State, error) {
	st := State{Today: civildate.InTehran(now), Now: now}
	q := store.New(s.db)
	l, err := latest(ctx, q, userID)
	if errors.Is(err, ErrNoLoss) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	st.Loss = &l
	if st.Count, err = q.CountLosses(ctx, userID); err != nil {
		return st, fmt.Errorf("loss: count: %w", err)
	}
	if st.Moods, err = q.ListLossMoodsSince(ctx, store.ListLossMoodsSinceParams{
		UserID: userID, LossID: l.ID, DateFrom: st.Today.AddDays(-(MoodDays - 1)),
	}); err != nil {
		return st, fmt.Errorf("loss: moods: %w", err)
	}
	day, err := healthlog.NewService(s.db).Day(ctx, userID, st.Today)
	if err != nil {
		return st, err
	}
	st.BleedingToday = bleedingOf(day)
	cq := carestore.New(s.db)
	if st.Beta, err = appointment(ctx, cq, userID, l.BetaReminderID); err != nil {
		return st, err
	}
	if st.Visit, err = appointment(ctx, cq, userID, l.VisitReminderID); err != nil {
		return st, err
	}
	return st, nil
}

// bleedingOf is today's logged bleeding from bloom's day log: the flow level, else "spotting", else "".
func bleedingOf(day []taxonomy.Entry) string {
	spotting := false
	for _, e := range day {
		if e.Category != "bleeding" {
			continue
		}
		if e.Param == "flow" && e.Code.Valid {
			return e.Code.String
		}
		if e.Param == "spotting" || (e.Param == "presence" && e.Code.Valid && e.Code.String != "none") {
			spotting = true
		}
	}
	if spotting {
		return "spotting"
	}
	return ""
}

func appointment(ctx context.Context, q *carestore.Queries, userID uint64, id sql.NullInt64) (*care.Appointment, error) {
	if !id.Valid {
		return nil, nil
	}
	row, err := q.GetAppointment(ctx, carestore.GetAppointmentParams{ID: uint64(id.Int64), UserID: userID}) //nolint:gosec // positive id
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("loss: appointment: %w", err)
	}
	a := care.ParseAppointment(row)
	return &a, nil
}
