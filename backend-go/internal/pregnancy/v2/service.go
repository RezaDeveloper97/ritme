package v2

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// Service is the v2 read model over the pregnancy store.
type Service struct{ q store.Querier }

// NewService wires the service.
func NewService(q store.Querier) *Service { return &Service{q: q} }

// ErrNotActive renders 409 {success:false, message, error_code: pregnancy_not_active}.
func errNotActive(locale string) error {
	return httpx.Fail(fiber.StatusConflict, T("messages.not_active", locale), "error_code", "pregnancy_not_active")
}

// active loads the user's profile and dates it; 409 when pregnancy mode is off or undated.
func (s *Service) active(ctx context.Context, userID uint64, today civildate.Date, locale string) (Dating, error) {
	p, err := s.q.GetProfileByUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return Dating{}, errNotActive(locale)
	}
	if err != nil {
		return Dating{}, fmt.Errorf("pregnancy v2: profile: %w", err)
	}
	if !p.PregnancyMode {
		return Dating{}, errNotActive(locale)
	}
	d, ok := Resolve(&p, today)
	if !ok {
		return Dating{}, errNotActive(locale)
	}
	return d, nil
}

func (s *Service) tipRows(ctx context.Context, week int, l Lang) ([]store.ListV2MessagePayloadsRow, error) {
	rows, err := s.q.ListV2MessagePayloads(ctx, store.ListV2MessagePayloadsParams{
		MessageGroup: "pregnancy_week_tip", ItemKey: itoa(week), Locales: l.codes(),
	})
	if err != nil {
		return nil, fmt.Errorf("pregnancy v2: week tip: %w", err)
	}
	return rows, nil
}

func (s *Service) state(ctx context.Context, userID uint64, week int) (store.PregnancyWeekUserState, error) {
	st, err := s.q.GetWeekUserState(ctx, store.GetWeekUserStateParams{UserID: userID, Week: wk(week)})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return st, fmt.Errorf("pregnancy v2: week state: %w", err)
	}
	return st, nil
}

func (s *Service) details(ctx context.Context, week int) (*store.PregnancyWeekDetail, error) {
	d, err := s.q.GetWeekDetails(ctx, wk(week))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("pregnancy v2: week details: %w", err)
	}
	return &d, nil
}

func confidenceJSON(level, locale string) *jsonx.OrderedMap {
	return jsonx.Obj("level", level, "label", T("confidence."+level, locale))
}

func rangeJSON(from, to civildate.Date) *jsonx.OrderedMap {
	return jsonx.Obj("from", from.String(), "to", to.String())
}

// Today is GET /pregnancy/v2/today, in five queries: profile, week details (prev..next), the
// current week's state, the extras row (alerts, next appointment, next care item), the tip.
func (s *Service) Today(ctx context.Context, userID uint64, now time.Time, l Lang) (*jsonx.OrderedMap, error) {
	today := civildate.InTehran(now)
	d, err := s.active(ctx, userID, today, l.Locale)
	if err != nil {
		return nil, err
	}
	cur := d.CurrentWeek()
	first, last := max(1, cur-1), min(MaxWeek, cur+1)
	rows, err := s.q.ListWeekDetailsRange(ctx, store.ListWeekDetailsRangeParams{WeekFrom: wk(first), WeekTo: wk(last)})
	if err != nil {
		return nil, fmt.Errorf("pregnancy v2: week details: %w", err)
	}
	byWeek := map[int]*store.PregnancyWeekDetail{}
	for i := range rows {
		byWeek[int(rows[i].WeekNumber)] = &rows[i]
	}
	st, err := s.state(ctx, userID, cur)
	if err != nil {
		return nil, err
	}
	ex, err := s.q.GetV2TodayExtras(ctx, store.GetV2TodayExtrasParams{
		UserID:      userID,
		AlertsSince: sql.NullTime{Time: AlertsSince(today), Valid: true},
		Now:         sql.NullTime{Time: now.In(civildate.Tehran), Valid: true},
		CurrentWeek: wk(cur),
	})
	if err != nil {
		return nil, fmt.Errorf("pregnancy v2: today extras: %w", err)
	}
	tip, err := s.tipRows(ctx, cur, l)
	if err != nil {
		return nil, err
	}

	carousel := []any{}
	for w := first; w <= last; w++ {
		rel := d.Relation(w)
		det := byWeek[w]
		var illustration any
		if det != nil {
			illustration = nullStr(det.IllustrationKey.String, det.IllustrationKey.Valid)
		}
		carousel = append(carousel, jsonx.Obj(
			"week", w, "trimester", tri(w), "title", slideTitle(d, w, l.Locale),
			"headline", headline(det, l), "size_line", sizeLine(det, rel, l),
			"illustration_key", illustration, "relation", rel,
		))
	}

	rFrom, rTo := d.BirthRange()
	daysLeft := d.DaysLeft()
	due := jsonx.Obj(
		"date", d.Due.String(), "date_label", FullDate(d.Due, l.Locale),
		"days_left", max(0, daysLeft), "overdue_days", max(0, -daysLeft),
		"range", rangeJSON(rFrom, rTo), "range_label", RangeLabel(rFrom, rTo, l.Locale),
	)
	trimesters := []any{}
	for i, w := range trimesterStartWeeks {
		start := d.Start().AddDays(w * 7)
		trimesters = append(trimesters, jsonx.Obj(
			"trimester", i+1, "start_date", start.String(), "start_label", FullDate(start, l.Locale),
			"percent", w*100/TermWeeks,
		))
	}

	return jsonx.Obj(
		"date", today.String(),
		"date_label", FullDate(today, l.Locale),
		"source", d.Source,
		"weeks", d.Weeks,
		"days", d.Days,
		"week", cur,
		"trimester", d.Trimester(),
		"confidence", confidenceJSON(d.Confidence, l.Locale),
		"uncertainty_days", d.Uncertainty,
		"carousel", carousel,
		"due", due,
		"progress", jsonx.Obj("week", cur, "percent", d.Percent(), "trimesters", trimesters),
		"next_visit", nextVisit(ex, d, l),
		"tip", tipJSON(cur, tip, l),
		"tasks", tasksJSON(byWeek[cur], doneKeys(st.DoneTaskKeys), l),
		"unread_alerts", ex.UnreadAlerts,
	), nil
}

// AlertsWindowDays is the window of GET /pregnancy/v2/alerts; the Today badge counts the same rows.
const AlertsWindowDays = 7

// AlertsSince is the first instant of the alerts window that ends on today (Tehran midnight,
// AlertsWindowDays-1 days back), shared by the Alerts list and the Today badge.
func AlertsSince(today civildate.Date) time.Time {
	return today.AddDays(-(AlertsWindowDays - 1)).TehranMidnight()
}

// slideTitle is the carousel headline: the age «۸ هفته و ۳ روز» on the current week, and the
// completed weeks at the start of week w («۷ هفته») on the others. The eyebrow above it already
// carries the trimester · week label, so the title never repeats it (design audit B1).
func slideTitle(d Dating, w int, locale string) string {
	if w == d.CurrentWeek() {
		return ageLabel(d, locale)
	}
	return AgeLabel(w-1, 0, locale)
}

// tri is the trimester of 1-based week w.
func tri(w int) int { return calc.Trimester(w - 1) }

func headline(d *store.PregnancyWeekDetail, l Lang) any {
	if d == nil {
		return nil
	}
	return l.pickNull(d.Headline)
}

// nextVisit is the next upcoming M3 appointment, else the next care-plan item; nil when neither.
func nextVisit(ex store.GetV2TodayExtrasRow, d Dating, l Lang) any {
	if ex.AppointmentID.Valid && ex.AppointmentAt.Valid {
		at := ex.AppointmentAt.Time.In(civildate.Tehran)
		date := civildate.FromTime(at)
		var meta map[string]any
		if ex.AppointmentMeta.Valid {
			_ = json.Unmarshal(ex.AppointmentMeta.V, &meta)
		}
		var key, stage any
		if s := str(meta, "care_item_key"); s != "" {
			key = s
		}
		if s := str(meta, "stage"); s != "" {
			stage = s
		}
		return jsonx.Obj(
			"appointment_id", ex.AppointmentID.Int64, "care_item_key", key, "title", ex.AppointmentTitle.String,
			"date", date.String(), "date_label", FullDate(date, l.Locale), "time", at.Format("15:04"),
			"week", weekOn(d, date), "days_until", d.Today.DiffDays(date), "stage", stage,
		)
	}
	if ex.CareItemKey.Valid {
		week := max(int(ex.CareItemWeekFrom.Int16), d.CurrentWeek())
		date, _ := d.WeekRange(week)
		if date.Before(d.Today) {
			date = d.Today
		}
		return jsonx.Obj(
			"appointment_id", nil, "care_item_key", ex.CareItemKey.String, "title", l.pick(ex.CareItemTitle.V),
			"date", date.String(), "date_label", FullDate(date, l.Locale), "time", nil,
			"week", week, "week_from", int(ex.CareItemWeekFrom.Int16), "week_to", int(ex.CareItemWeekTo.Int16),
			"days_until", d.Today.DiffDays(date), "stage", nil,
		)
	}
	return nil
}

// weekOn is the 1-based week that contains date.
func weekOn(d Dating, date civildate.Date) int {
	return min(MaxWeek, max(1, d.Start().DiffDays(date)/7+1))
}

// Week is GET /pregnancy/v2/weeks/{n}.
func (s *Service) Week(ctx context.Context, userID uint64, n int, now time.Time, l Lang) (*jsonx.OrderedMap, error) {
	d, err := s.active(ctx, userID, civildate.InTehran(now), l.Locale)
	if err != nil {
		return nil, err
	}
	det, err := s.details(ctx, n)
	if err != nil {
		return nil, err
	}
	st, err := s.state(ctx, userID, n)
	if err != nil {
		return nil, err
	}
	tip, err := s.tipRows(ctx, n, l)
	if err != nil {
		return nil, err
	}
	from, to := d.WeekRange(n)
	done := doneKeys(st.DoneTaskKeys)
	rel := d.Relation(n)
	return jsonx.Obj(
		"week", n,
		"trimester", tri(n),
		"title", WeekLabel(n, l.Locale),
		"relation", rel,
		"current_week", d.CurrentWeek(),
		"range", rangeJSON(from, to),
		"range_label", RangeLabel(from, to, l.Locale),
		"size_line", sizeLine(det, rel, l),
		"bookmarked", st.Bookmarked,
		"done_task_keys", done,
		"details", detailsJSON(det, l),
		"tasks", tasksJSON(det, done, l),
		"tip", tipJSON(n, tip, l),
	), nil
}

// StateInput is a validated PUT /weeks/{n}/state (nil = keep).
type StateInput struct {
	Bookmarked   *bool
	DoneTaskKeys []string // nil = keep
}

// SaveState merges in with the stored row (unknown task keys are dropped) and upserts it.
func (s *Service) SaveState(ctx context.Context, userID uint64, n int, in StateInput, now time.Time, locale string) (*jsonx.OrderedMap, error) {
	if _, err := s.active(ctx, userID, civildate.InTehran(now), locale); err != nil {
		return nil, err
	}
	st, err := s.state(ctx, userID, n)
	if err != nil {
		return nil, err
	}
	bookmarked, done := st.Bookmarked, doneKeys(st.DoneTaskKeys)
	if in.Bookmarked != nil {
		bookmarked = *in.Bookmarked
	}
	if in.DoneTaskKeys != nil {
		det, err := s.details(ctx, n)
		if err != nil {
			return nil, err
		}
		known, seen := taskKeys(det), map[string]bool{}
		done = []string{}
		for _, k := range in.DoneTaskKeys {
			if known[k] && !seen[k] {
				seen[k] = true
				done = append(done, k)
			}
		}
	}
	raw, err := json.Marshal(done)
	if err != nil {
		return nil, fmt.Errorf("pregnancy v2: done keys: %w", err)
	}
	if err := s.q.UpsertWeekUserState(ctx, store.UpsertWeekUserStateParams{
		UserID: userID, Week: wk(n), Bookmarked: bookmarked,
		DoneTaskKeys: db.NullRawJSON{V: raw, Valid: true},
		Now:          sql.NullTime{Time: now.In(civildate.Tehran).Truncate(time.Second), Valid: true},
	}); err != nil {
		return nil, fmt.Errorf("pregnancy v2: save week state: %w", err)
	}
	return jsonx.Obj("week", n, "bookmarked", bookmarked, "done_task_keys", done), nil
}

// wk is a 1..MaxWeek week as the tinyint the store takes.
func wk(n int) uint8 { return uint8(min(MaxWeek, max(0, n))) } //nolint:gosec // clamped to 0..42
