package menopause

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/ritme/backend-go/internal/care"
	carestore "github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/menopause/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Treatment screen placements of meno_tips (CB-MENO-01 seeds) and the tip carrying the review interval.
const (
	PlacementTreatment          = "treatment"
	PlacementTreatmentLifestyle = "treatment_lifestyle"
	TipHRTReview                = "hrt_review"
)

// ItemView is one item of the treatment screen for the week of a day.
type ItemView struct {
	Item       store.TreatmentItem
	Medication *care.Medication // the care reminder (nil: lifestyle, or deleted in /care)
	Day        civildate.Date
	Week       []civildate.Date                 // Saturday … Friday of Day's week
	Taken      map[civildate.Date]sql.NullInt16 // taken days of the week → amount
}

// Active reports whether the item is active on the screen's day.
func (v ItemView) Active() bool { return activeOn(v.Item, v.Day) }

// WeekAmount is the week's total towards a lifestyle goal (minutes or sessions).
func (v ItemView) WeekAmount() int {
	n := 0
	for _, a := range v.Taken {
		n += amountOf(v.Item, a)
	}
	return n
}

// WeekDays are the week's days the item was active on, up to the screen's day (the adherence denominator).
func (v ItemView) WeekDays() int {
	n := 0
	for _, d := range v.Week {
		if !d.After(v.Day) && activeOn(v.Item, d) && v.scheduledOn(d) {
			n++
		}
	}
	return n
}

// DaysTaken are the taken days of WeekDays.
func (v ItemView) DaysTaken() int {
	n := 0
	for d := range v.Taken {
		if !d.After(v.Day) && activeOn(v.Item, d) && v.scheduledOn(d) {
			n++
		}
	}
	return n
}

// scheduledOn: a daily item every day; a weekly one on its care weekday (any day without a reminder).
func (v ItemView) scheduledOn(d civildate.Date) bool {
	if v.Medication == nil {
		return true
	}
	return slices.Contains(v.Medication.Meta.Weekdays, care.SaturdayWeekday(d))
}

// AdherencePct is DaysTaken / WeekDays in whole percent (nil for lifestyle goals or before the item started).
func (v ItemView) AdherencePct() *int {
	if v.Item.Kind == KindLifestyle {
		return nil
	}
	days := v.WeekDays()
	if days == 0 {
		return nil
	}
	p := (v.DaysTaken()*100 + days/2) / days
	return &p
}

// TreatmentScreen is GET /menopause/treatment?date=.
type TreatmentScreen struct {
	Day   civildate.Date
	Week  []civildate.Date
	Items []ItemView // every item, treatment order (kind, sort order)
	// Review is the next doctor review: the earliest review_on of an active HRT item; else a suggestion from the
	// earliest HRT start + ReviewAfterMonths (Suggested).
	Review       *civildate.Date
	ReviewItemID uint64
	Suggested    bool
	// SideEffects are the week's logged side effects: day → codes (SideEffectCodes order).
	SideEffects map[civildate.Date][]string
	Tips        []catalog.Item
}

// WeekOf is the Saturday-start week of day.
func WeekOf(day civildate.Date) []civildate.Date {
	start := day.StartOfWeek()
	out := make([]civildate.Date, 7)
	for i := range out {
		out[i] = start.AddDays(i)
	}
	return out
}

// medications are the user's care medications by id.
func (s *Service) medications(ctx context.Context, userID uint64) (map[uint64]care.Medication, error) {
	rows, err := carestore.New(s.db).ListMedications(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("menopause: list medications: %w", err)
	}
	out := make(map[uint64]care.Medication, len(rows))
	for _, r := range rows {
		out[r.ID] = care.ParseMedication(r)
	}
	return out, nil
}

// ItemViews builds the views of items for the week of day.
func (s *Service) ItemViews(ctx context.Context, userID uint64, items []store.TreatmentItem, day civildate.Date,
) ([]ItemView, error) {
	week := WeekOf(day)
	intakes, err := s.itemIntakes(ctx, userID, items, week[0], week[6])
	if err != nil {
		return nil, err
	}
	meds, err := s.medications(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]ItemView, 0, len(items))
	for _, it := range items {
		v := ItemView{Item: it, Day: day, Week: week, Taken: intakes[it.ID]}
		if v.Taken == nil {
			v.Taken = map[civildate.Date]sql.NullInt16{}
		}
		if it.ReminderID.Valid {
			if m, ok := meds[uint64(it.ReminderID.Int64)]; ok { //nolint:gosec // FK id
				v.Medication = &m
			}
		}
		out = append(out, v)
	}
	return out, nil
}

// ItemView is the view of one of the user's items for the week of day.
func (s *Service) ItemView(ctx context.Context, userID, id uint64, day civildate.Date) (ItemView, error) {
	it, err := s.GetItem(ctx, userID, id)
	if err != nil {
		return ItemView{}, err
	}
	vs, err := s.ItemViews(ctx, userID, []store.TreatmentItem{it}, day)
	if err != nil {
		return ItemView{}, err
	}
	return vs[0], nil
}

type tipMeta struct {
	Placement         string `json:"placement"`
	ReviewAfterMonths int    `json:"review_after_months"`
}

// TreatmentScreen builds the treatment & care screen for the week of day.
func (s *Service) TreatmentScreen(ctx context.Context, userID uint64, day civildate.Date) (TreatmentScreen, error) {
	sc := TreatmentScreen{Day: day, Week: WeekOf(day), SideEffects: map[civildate.Date][]string{}}
	items, err := s.q.ListTreatmentItems(ctx, userID)
	if err != nil {
		return TreatmentScreen{}, fmt.Errorf("menopause: list treatment: %w", err)
	}
	slices.SortStableFunc(items, func(a, b store.TreatmentItem) int {
		return slices.Index(Kinds, a.Kind) - slices.Index(Kinds, b.Kind)
	})
	if sc.Items, err = s.ItemViews(ctx, userID, items, day); err != nil {
		return TreatmentScreen{}, err
	}
	tips, err := s.catalog.Items(ctx, GroupTips)
	if err != nil {
		return TreatmentScreen{}, fmt.Errorf("menopause: load catalog %s: %w", GroupTips, err)
	}
	reviewMonths := DefaultReviewAfterMonths
	for _, t := range tips {
		var meta tipMeta
		_ = json.Unmarshal(t.Meta, &meta)
		if meta.Placement == PlacementTreatment || meta.Placement == PlacementTreatmentLifestyle {
			sc.Tips = append(sc.Tips, t)
		}
		if t.Code == TipHRTReview && meta.ReviewAfterMonths > 0 {
			reviewMonths = meta.ReviewAfterMonths
		}
	}
	sc.review(reviewMonths)
	logs, err := s.q.ListSideEffectLogsInRange(ctx, store.ListSideEffectLogsInRangeParams{
		UserID: userID, FromDate: sc.Week[0], ToDate: sc.Week[6],
	})
	if err != nil {
		return TreatmentScreen{}, fmt.Errorf("menopause: list side effects: %w", err)
	}
	for _, l := range logs {
		sc.SideEffects[l.LogDate] = append(sc.SideEffects[l.LogDate], l.Code)
	}
	for d, codes := range sc.SideEffects {
		sc.SideEffects[d] = ordered(codes, SideEffectCodes)
	}
	return sc, nil
}

// ordered keeps codes in the order of allowed (unknown codes last, sorted).
func ordered(codes, allowed []string) []string {
	out := []string{}
	for _, a := range allowed {
		if slices.Contains(codes, a) {
			out = append(out, a)
		}
	}
	var rest []string
	for _, c := range codes {
		if !slices.Contains(allowed, c) {
			rest = append(rest, c)
		}
	}
	slices.Sort(rest)
	return append(out, rest...)
}

func (sc *TreatmentScreen) review(months int) {
	var start *civildate.Date
	for _, v := range sc.Items {
		it := v.Item
		if it.Kind != KindHRT || !v.Active() {
			continue
		}
		if it.ReviewOn.Valid && (sc.Review == nil || it.ReviewOn.Date.Before(*sc.Review)) {
			d := it.ReviewOn.Date
			sc.Review, sc.ReviewItemID = &d, it.ID
		}
		if it.StartedOn.Valid && (start == nil || it.StartedOn.Date.Before(*start)) {
			d := it.StartedOn.Date
			start = &d
		}
	}
	if sc.Review == nil && start != nil {
		d := engine.AddMonths(*start, months)
		sc.Review, sc.Suggested = &d, true
	}
}

// --- JSON -----------------------------------------------------------------------------------------------------------

func nullDate(d civildate.NullDate) any {
	if !d.Valid {
		return nil
	}
	return d.Date
}

func nullString(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return s.String
}

// ItemJSON is one item of the treatment screen.
func ItemJSON(v ItemView) *jsonx.OrderedMap {
	it := v.Item
	var goal, reminder any
	if it.WeeklyGoal.Valid {
		goal = jsonx.Obj("target", int(it.WeeklyGoal.Int16), "unit", it.GoalUnit.String, "amount", v.WeekAmount(),
			"done", v.WeekAmount() >= int(it.WeeklyGoal.Int16))
	}
	if m := v.Medication; m != nil {
		var at any
		if len(m.Meta.Times) > 0 {
			at = m.Meta.Times[0]
		}
		reminder = jsonx.Obj("id", m.Row.ID, "time", at, "notify", m.Meta.Notify, "active", m.Row.IsActive)
	}
	week := make([]*jsonx.OrderedMap, len(v.Week))
	for i, d := range v.Week {
		a, taken := v.Taken[d]
		var amount any
		if a.Valid {
			amount = int(a.Int16)
		}
		week[i] = jsonx.Obj("date", d, "taken", taken, "amount", amount)
	}
	_, today := v.Taken[v.Day]
	var form any
	if v.Medication != nil {
		form = v.Medication.Meta.Form
	}
	return jsonx.Obj(
		"id", it.ID,
		"kind", it.Kind,
		"name", it.Name,
		"dose", nullString(it.Dose),
		"schedule", nullString(it.Schedule),
		"form", form,
		"started_on", nullDate(it.StartedOn),
		"review_on", nullDate(it.ReviewOn),
		"stopped_on", nullDate(it.StoppedOn),
		"active", v.Active(),
		"taken_today", today,
		"week", week,
		"days_taken", v.DaysTaken(),
		"days", v.WeekDays(),
		"adherence_pct", ptr(v.AdherencePct()),
		"goal", goal,
		"reminder", reminder,
	)
}

// TreatmentScreenJSON is GET /menopause/treatment.
func TreatmentScreenJSON(sc TreatmentScreen, loc catalog.Localizer) *jsonx.OrderedMap {
	groups := jsonx.NewObject()
	for _, k := range Kinds {
		groups.Set(k, []*jsonx.OrderedMap{})
	}
	stopped := []*jsonx.OrderedMap{}
	for _, v := range sc.Items {
		if !v.Active() && v.Item.StoppedOn.Valid && !v.Item.StoppedOn.Date.After(sc.Day) {
			stopped = append(stopped, ItemJSON(v))
			continue
		}
		cur, _ := groups.Get(v.Item.Kind)
		list, _ := cur.([]*jsonx.OrderedMap)
		groups.Set(v.Item.Kind, append(list, ItemJSON(v)))
	}
	var review any
	if sc.Review != nil {
		var item any
		if sc.ReviewItemID > 0 {
			item = sc.ReviewItemID
		}
		review = jsonx.Obj("on", *sc.Review, "item_id", item, "suggested", sc.Suggested)
	}
	week := make([]*jsonx.OrderedMap, len(sc.Week))
	for i, d := range sc.Week {
		codes := sc.SideEffects[d]
		if codes == nil {
			codes = []string{}
		}
		week[i] = jsonx.Obj("date", d, "codes", codes)
	}
	today := sc.SideEffects[sc.Day]
	if today == nil {
		today = []string{}
	}
	tips := make([]any, len(sc.Tips))
	for i, t := range sc.Tips {
		tips[i] = loc.Public(t)
	}
	return jsonx.Obj(
		"date", sc.Day,
		"week", jsonx.Obj("from", sc.Week[0], "to", sc.Week[6]),
		"items", groups,
		"stopped", stopped,
		"review", review,
		"side_effects", jsonx.Obj("codes", SideEffectCodes, "today", today, "week", week),
		"tips", tips,
	)
}
