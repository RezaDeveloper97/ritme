package home

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/content"
	"github.com/ritme/backend-go/internal/content/sanitizer"
	"github.com/ritme/backend-go/internal/cycle/legacy"
	"github.com/ritme/backend-go/internal/cycle/recommendation"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/home/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// The 17 sections of backend/app/Services/HomePage/Sections, one type each. A section that
// returns an error (or panics) is logged and omitted by Page.

// allModes is AbstractHomeSection::supports(): visible in every mode.
type allModes struct{}

func (allModes) Supports(*Context) bool { return true }

// cycleOnly is `supports() = $context->isCycleMode()`.
type cycleOnly struct{}

func (cycleOnly) Supports(hc *Context) bool { return hc.IsCycleMode() }

// ---------------------------------------------------------------------------
// 1. header (HeaderSection.php)

type headerSection struct{ allModes }

func (headerSection) Key() string { return "header" }
func (headerSection) Order() int  { return 10 }

func (s headerSection) Build(hc *Context) (*Rendered, error) {
	unread, err := hc.q().CountUnreadNotifications(hc.Ctx, hc.UserID)
	if err != nil {
		return nil, fmt.Errorf("home: unread notifications: %w", err)
	}
	var name any
	greeting := hc.T("سلام", "Hi")
	if hc.UserName.Valid {
		name = hc.UserName.String
		if phpval.Truthy(hc.UserName.String) {
			greeting = hc.T("سلام "+hc.UserName.String, "Hi "+hc.UserName.String)
		}
	}
	p := hc.Profile()
	return &Rendered{
		Key: s.Key(), Type: "header", Order: s.Order(),
		Data: jsonx.Obj(
			"app_name", "ریتمی",
			"tagline", hc.T("همراه سلامتی شما", "Your health companion"),
			"greeting", greeting,
			"user", jsonx.Obj("id", hc.UserID, "name", name),
			"mode", string(hc.Mode),
			"mode_label", hc.Mode.Label(hc.Locale),
			"is_premium", p != nil && p.SubscriptionType == string(enums.SubscriptionTypePremium),
			"date", hc.Date.String(),
			"weekday_iso", hc.Date.ISOWeekday(),
			"moon_phase", MoonPhaseFor(hc.Date, hc.Locale),
			"notifications", jsonx.Obj("unread_count", unread, "has_unread", unread > 0),
		),
	}, nil
}

// ---------------------------------------------------------------------------
// 2. week_calendar (WeekCalendarSection.php)

type weekCalendarSection struct{ cycleOnly }

func (weekCalendarSection) Key() string { return "week_calendar" }
func (weekCalendarSection) Order() int  { return 20 }

func (s weekCalendarSection) Build(hc *Context) (*Rendered, error) {
	hasCycle := hc.HasCycleData()
	weekStart := hc.Date.StartOfWeek()
	days := make([]*jsonx.OrderedMap, 0, 7)
	for i := range 7 {
		day := weekStart.AddDays(i)
		var cycleDay any
		phase, fertile := "", false
		if hasCycle {
			c, err := hc.CalendarDataFor(day)
			if err != nil {
				return nil, err
			}
			if c.Complete {
				cycleDay, phase = c.Day, string(c.Phase)
			}
			fertile = c.IsFertileWindow
		}
		var phaseValue any
		if phase != "" {
			phaseValue = phase
		}
		days = append(days, jsonx.Obj(
			"date", day.String(),
			"weekday_iso", day.ISOWeekday(),
			"is_today", day == hc.Date,
			"cycle_day", cycleDay,
			"phase", phaseValue,
			"phase_label", phaseLabel(phase, hc.Locale),
			"color", phaseColor(phase),
			"is_period", phase == string(enums.CyclePhaseMenstruation),
			"is_fertile", fertile,
			"is_ovulation", phase == string(enums.CyclePhaseOvulation),
		))
	}
	return &Rendered{
		Key: s.Key(), Type: "week_calendar", Order: s.Order(),
		Data: jsonx.Obj("week_start", weekStart.String(), "days", days),
	}, nil
}

// ---------------------------------------------------------------------------
// 3. next_period (NextPeriodSection.php)

type nextPeriodSection struct{ cycleOnly }

func (nextPeriodSection) Key() string { return "next_period" }
func (nextPeriodSection) Order() int  { return 30 }

func (s nextPeriodSection) Build(hc *Context) (*Rendered, error) {
	calc, err := hc.CycleData()
	if err != nil || calc == nil || !calc.Complete {
		return nil, err
	}
	cycleDay, cycleLength := calc.Day, calc.CycleLength
	probability := calc.FinalProbability

	daysUntil := max(0, cycleLength-cycleDay+1)
	next := hc.Date.AddDays(daysUntil)
	progress := 0
	if cycleLength > 0 {
		progress = int(phpround.Round(float64(cycleDay)/float64(cycleLength)*100, 0))
	}
	level, label := "high", hc.T("بالا", "High")
	switch {
	case probability < 5:
		level, label = "low", hc.T("کم", "Low")
	case probability < 15:
		level, label = "medium", hc.T("متوسط", "Medium")
	}
	days := strconv.Itoa(daysUntil)
	return &Rendered{
		Key: s.Key(), Type: "next_period", Order: s.Order(),
		Title: str(hc.T(days+" روز تا پریود بعدی", days+" days to next period")),
		Data: jsonx.Obj(
			"days_until_period", daysUntil,
			"next_period_date", next.String(),
			"cycle_day", cycleDay,
			"cycle_length", cycleLength,
			"progress_percent", min(100, max(0, progress)),
			"is_in_period", calc.Phase == enums.CyclePhaseMenstruation,
			"is_period_tomorrow", calc.IsPeriodTomorrow,
			"is_fertile_window", calc.IsFertileWindow,
			"uncertainty_range", calc.UncertaintyRange,
			"pregnancy_probability", jsonx.Obj("percent", jsonx.Float(probability), "level", level, "label", label),
		),
		Subtitle: str(hc.T("پریود بعدی شما حدوداً "+next.String(), "Your next period is around "+next.String())),
		Action:   action("log_period_start", hc.T("شروع پریود", "Log period start")),
	}, nil
}

// ---------------------------------------------------------------------------
// 4. cycle_prediction (CyclePredictionSection.php)

type cyclePredictionSection struct{ cycleOnly }

func (cyclePredictionSection) Key() string { return "cycle_prediction" }
func (cyclePredictionSection) Order() int  { return 40 }

func (s cyclePredictionSection) Build(hc *Context) (*Rendered, error) {
	calc, err := hc.CycleData()
	if err != nil {
		return nil, err
	}
	start, ok, err := hc.CurrentCycleStart()
	if err != nil || calc == nil || !ok || !calc.Complete {
		return nil, err
	}
	cycleDay, cycleLength, ovulationDay := calc.Day, calc.CycleLength, calc.OvulationDay
	dateForDay := func(n int) string { return start.AddDays(n - 1).String() }
	nextOcc := func(day int) int {
		if day >= cycleDay {
			return day
		}
		return day + cycleLength
	}
	ovulationOcc := nextOcc(ovulationDay)
	fertileStart, fertileEnd := ovulationOcc-5, ovulationOcc+1

	cards := []*jsonx.OrderedMap{
		jsonx.Obj(
			"key", "fertile_window",
			"label", hc.T("پنجره باروری", "Fertile window"),
			"start_date", dateForDay(fertileStart),
			"end_date", dateForDay(fertileEnd),
			"days_until", max(0, fertileStart-cycleDay),
			"is_active", calc.IsFertileWindow,
		),
		jsonx.Obj(
			"key", "ovulation",
			"label", hc.T("تخمک‌گذاری", "Ovulation"), //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from PHP
			"date", dateForDay(ovulationOcc),
			"days_until", ovulationOcc-cycleDay,
		),
		jsonx.Obj(
			"key", "next_period",
			"label", hc.T("پریود بعدی", "Next period"),
			"date", dateForDay(cycleLength+1),
			"days_until", max(0, cycleLength-cycleDay+1),
		),
	}
	return &Rendered{
		Key: s.Key(), Type: "cycle_prediction", Order: s.Order(),
		Data: jsonx.Obj("cards", cards, "uncertainty_range", calc.UncertaintyRange),
	}, nil
}

// ---------------------------------------------------------------------------
// 5. recommendations (RecommendationsSection.php)

type recommendationsSection struct{ cycleOnly }

func (recommendationsSection) Key() string { return "recommendations" }
func (recommendationsSection) Order() int  { return 50 }

// tipPick is AbstractHomeSection::pick on a tip / title: `$v[$locale] ?? $v['fa'] ?? $v['en']`
// (both keys are always set, so only fa and en can answer).
func tipPick(locale, fa, en string) string {
	if locale == "en" {
		return en
	}
	return fa
}

func (s recommendationsSection) Build(hc *Context) (*Rendered, error) {
	calc, err := hc.CycleData()
	if err != nil || calc == nil {
		return nil, err
	}
	tips := calc.DailyTips
	if len(tips) == 0 {
		return nil, nil
	}
	if len(tips) > 6 {
		tips = tips[:6]
	}
	items := []*jsonx.OrderedMap{}
	var tagTypes []string
	for _, tip := range tips {
		text := tipPick(hc.Locale, tip.FA, tip.EN)
		if !phpval.Truthy(text) {
			continue
		}
		if !slices.Contains(tagTypes, tip.Type) {
			tagTypes = append(tagTypes, tip.Type)
		}
		items = append(items, jsonx.Obj(
			"type", tip.Type,
			"icon", enums.RecommendationTypeIconFor(tip.Type),
			"title", tipTitle(tip, hc.Locale),
			"text", text,
		))
	}
	if len(items) == 0 {
		return nil, nil
	}
	tags := make([]*jsonx.OrderedMap, len(tagTypes))
	for i, t := range tagTypes {
		tags[i] = jsonx.Obj("type", t, "icon", enums.RecommendationTypeIconFor(t))
	}
	return &Rendered{
		Key: s.Key(), Type: "recommendations", Order: s.Order(),
		Title: str(hc.T("توصیه‌های امروز", "Today's recommendations")), //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from PHP
		Data:  jsonx.Obj("tags", tags, "items", items),
	}, nil
}

// tipTitle is `pick($tip['title'] ?? null) ?? RecommendationType::labelFor($type, $locale)`.
func tipTitle(tip recommendation.Tip, locale string) string {
	if tip.Title != nil {
		return tipPick(locale, tip.Title.FA, tip.Title.EN)
	}
	return enums.RecommendationTypeLabelFor(tip.Type, locale)
}

// ---------------------------------------------------------------------------
// 6. tasks (TasksSection.php)

type tasksSection struct{ allModes }

func (tasksSection) Key() string { return "tasks" }
func (tasksSection) Order() int  { return 60 }

// taskTemplates is TaskTemplate::active()->forPhase($phase)->get().
func taskTemplates(hc *Context, phase string) ([]store.ListTaskTemplatesForPhaseRow, error) {
	rows, err := hc.q().ListTaskTemplatesForPhase(hc.Ctx, store.ListTaskTemplatesForPhaseParams{
		HasPhase: b2i(phpval.Truthy(phase)), Phase: sql.NullString{String: phase, Valid: phase != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("home: task templates: %w", err)
	}
	return rows, nil
}

// completedTaskIDs is the set of task templates the user completed on date.
func completedTaskIDs(hc *Context, date civildate.Date) (map[uint64]bool, error) {
	ids, err := hc.q().ListTaskCompletionIDsOn(hc.Ctx, store.ListTaskCompletionIDsOnParams{UserID: hc.UserID, CompletionDate: date})
	if err != nil {
		return nil, fmt.Errorf("home: task completions: %w", err)
	}
	set := make(map[uint64]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}

func percentOf(completed, total int) int {
	if total <= 0 {
		return 0
	}
	return int(phpround.Round(float64(completed)/float64(total)*100, 0))
}

func (s tasksSection) Build(hc *Context) (*Rendered, error) {
	phase, err := hc.Phase()
	if err != nil {
		return nil, err
	}
	templates, err := taskTemplates(hc, phase)
	if err != nil || len(templates) == 0 {
		return nil, err
	}
	done, err := completedTaskIDs(hc, hc.Date)
	if err != nil {
		return nil, err
	}
	items := make([]*jsonx.OrderedMap, 0, len(templates))
	completed := 0
	for _, t := range templates {
		isDone := done[t.ID]
		if isDone {
			completed++
		}
		var categoryLabel any
		icon := "clipboard-check"
		cat, ok := enums.TaskCategoryFrom(t.Category)
		if ok {
			categoryLabel = cat.Label(hc.Locale)
			icon = cat.Icon()
		}
		if t.Icon.Valid {
			icon = t.Icon.String
		}
		items = append(items, jsonx.Obj(
			"id", t.ID,
			"key", t.Key,
			"title", localized(t.Title, hc.Locale, hc.DefaultLocale),
			"description", localizedNull(t.Description, hc.Locale, hc.DefaultLocale),
			"category", t.Category,
			"category_label", categoryLabel,
			"icon", icon,
			"is_completed", isDone,
		))
	}
	total := len(items)
	return &Rendered{
		Key: s.Key(), Type: "tasks", Order: s.Order(),
		Title: str(hc.T("وظایف امروز", "Today's tasks")),
		Data: jsonx.Obj(
			"items", items,
			"progress", jsonx.Obj("completed", completed, "total", total, "percent", percentOf(completed, total)),
		),
		Subtitle: str(fmt.Sprintf("%d/%d", completed, total)),
	}, nil
}

// ---------------------------------------------------------------------------
// 7. challenge (ChallengeSection.php)

type challengeSection struct{ allModes }

func (challengeSection) Key() string { return "challenge" }
func (challengeSection) Order() int  { return 70 }

func (s challengeSection) Build(hc *Context) (*Rendered, error) {
	day, err := hc.CycleDay()
	if err != nil {
		return nil, err
	}
	payload, err := ChallengePayload(hc.Ctx, hc.q(), hc.UserID, hc.Date, hc.Locale, hc.DefaultLocale, day, hc.RecentLogs(3))
	if err != nil || payload == nil {
		return nil, err
	}
	return &Rendered{
		Key: s.Key(), Type: "challenge", Order: s.Order(),
		Title: str(hc.T("چالش امروز", "Today's challenge")),
		Data:  payload,
	}, nil
}

// ---------------------------------------------------------------------------
// 8. doctor_reminder (DoctorReminderSection.php)

type doctorReminderSection struct{ allModes }

func (doctorReminderSection) Key() string { return "doctor_reminder" }
func (doctorReminderSection) Order() int  { return 80 }

func (s doctorReminderSection) Build(hc *Context) (*Rendered, error) {
	r, err := hc.q().GetDoctorReminder(hc.Ctx, store.GetDoctorReminderParams{
		UserID: hc.UserID, Day: civildate.NullDate{Date: hc.Date, Valid: true},
		DayStart: sql.NullTime{Time: hc.Date.TehranMidnight(), Valid: true},
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("home: doctor reminder: %w", err)
	}
	var scheduled, date, clock any
	if r.ScheduledAt.Valid {
		t := r.ScheduledAt.Time.In(civildate.Tehran)
		scheduled, date, clock = jsonx.ISO8601(t), civildate.FromTime(t).String(), t.Format("15:04")
	}
	return &Rendered{
		Key: s.Key(), Type: "doctor_reminder", Order: s.Order(),
		Title: str(hc.T("یادآور دکتر", "Doctor reminder")),
		Data: jsonx.Obj(
			"id", r.ID,
			"title", r.Title,
			"subtitle", nullString(r.Subtitle),
			"notes", nullString(r.Notes),
			"scheduled_at", scheduled,
			"date", date,
			"time", clock,
			"meta", decodedJSON(r.Meta),
			"icon", enums.ReminderTypeDoctor.Icon(),
		),
	}, nil
}

// ---------------------------------------------------------------------------
// 9. medication_reminder (MedicationReminderSection.php)

type medicationReminderSection struct{ allModes }

func (medicationReminderSection) Key() string { return "medication_reminder" }
func (medicationReminderSection) Order() int  { return 90 }

func (s medicationReminderSection) Build(hc *Context) (*Rendered, error) {
	r, err := hc.q().GetMedicationReminder(hc.Ctx, store.GetMedicationReminderParams{
		UserID: hc.UserID, Day: civildate.NullDate{Date: hc.Date, Valid: true},
		DayStart: sql.NullTime{Time: hc.Date.TehranMidnight(), Valid: true},
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("home: medication reminder: %w", err)
	}
	var clock any
	switch {
	case r.RecurrenceTime.Valid && phpval.Truthy(r.RecurrenceTime.String):
		clock = substr(r.RecurrenceTime.String, 5)
	case r.ScheduledAt.Valid:
		clock = r.ScheduledAt.Time.In(civildate.Tehran).Format("15:04")
	}
	return &Rendered{
		Key: s.Key(), Type: "medication_reminder", Order: s.Order(),
		Title: str(hc.T("یادآور دارو", "Medication reminder")),
		Data: jsonx.Obj(
			"id", r.ID,
			"title", r.Title,
			"subtitle", nullString(r.Subtitle),
			"notes", nullString(r.Notes),
			"time", clock,
			"recurrence", r.Recurrence,
			"meta", decodedJSON(r.Meta),
			"icon", enums.ReminderTypeMedication.Icon(),
		),
	}, nil
}

// substr is substr($s, 0, n) on bytes.
func substr(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// ---------------------------------------------------------------------------
// 10. smart_tip (SmartTipSection.php)

type smartTipSection struct{ allModes }

func (smartTipSection) Key() string { return "smart_tip" }
func (smartTipSection) Order() int  { return 100 }

// smartTipFlags are the text_flags tried, in order, when the message system has nothing.
var smartTipFlags = []string{"phase_info", "probability_message", "pms_warning", "fertility_status"}

func (s smartTipSection) Build(hc *Context) (*Rendered, error) {
	var text, source any
	if res := hc.Messages(); res != nil {
		text = coalesce(res.PrimaryMessage, "long_message", "short_message")
		if phpval.Truthy(text) {
			source = "message_engine"
		}
		if !phpval.Truthy(text) && len(res.Correlations) > 0 {
			text = coalesce(res.Correlations[0], "insight_message")
			source = nil
			if phpval.Truthy(text) {
				source = "correlation"
			}
		}
	}
	if !phpval.Truthy(text) {
		calc, err := hc.CycleData()
		if err != nil {
			return nil, err
		}
		if calc != nil {
			if f, ok := findFlag(calc.TextFlags); ok {
				text, source = tipPick(hc.Locale, f.FA, f.EN), "cycle_engine"
			}
		}
	}
	if !phpval.Truthy(text) {
		return nil, nil
	}
	return &Rendered{
		Key: s.Key(), Type: "smart_tip", Order: s.Order(),
		Title: str(hc.T("نکته هوشمند", "Smart tip")),
		Data:  jsonx.Obj("text", text, "source", source),
	}, nil
}

func findFlag(flags []legacy.TextFlag) (legacy.TextFlag, bool) {
	for _, key := range smartTipFlags {
		for _, f := range flags {
			if f.Key == key {
				return f, true
			}
		}
	}
	return legacy.TextFlag{}, false
}

// coalesce is `$m[k1] ?? $m[k2] ?? null`.
func coalesce(m *jsonx.OrderedMap, keys ...string) any {
	if m == nil {
		return nil
	}
	for _, k := range keys {
		if v, ok := m.Get(k); ok && v != nil {
			return v
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// 11. affirmation (AffirmationSection.php)

type affirmationSection struct{ allModes }

func (affirmationSection) Key() string { return "affirmation" }
func (affirmationSection) Order() int  { return 110 }

func (s affirmationSection) Build(hc *Context) (*Rendered, error) {
	phase, err := hc.Phase()
	if err != nil {
		return nil, err
	}
	rows, err := hc.q().ListAffirmationsForPhase(hc.Ctx, store.ListAffirmationsForPhaseParams{
		HasPhase: b2i(phpval.Truthy(phase)), Phase: sql.NullString{String: phase, Valid: phase != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("home: affirmations: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	a := rows[hc.Date.DayOfYear()%len(rows)]
	return &Rendered{
		Key: s.Key(), Type: "affirmation", Order: s.Order(),
		Data: jsonx.Obj("id", a.ID, "text", localized(a.Text, hc.Locale, hc.DefaultLocale), "icon", "sparkle"),
	}, nil
}

// ---------------------------------------------------------------------------
// 12. weekly_summary (WeeklySummarySection.php)

type weeklySummarySection struct{ allModes }

func (weeklySummarySection) Key() string { return "weekly_summary" }
func (weeklySummarySection) Order() int  { return 120 }

const summaryWindow = 7

func (s weeklySummarySection) Build(hc *Context) (*Rendered, error) {
	logs := hc.RecentLogs(summaryWindow)
	averages := weeklyAverages(scored(logs))
	previousLogs := hc.LogsBetween(hc.Date.AddDays(-(summaryWindow*2 - 1)), hc.Date.AddDays(-summaryWindow))
	previous := weeklyAverages(scored(previousLogs))

	meta := []struct{ key, fa, en, icon string }{
		{"mood", "روحیه", "Mood", "smile"},
		{"sleep", "خواب", "Sleep", "moon"},
		{"energy", "انرژی", "Energy", "bolt"},
	}
	items := make([]*jsonx.OrderedMap, len(meta))
	var scoredValues []int
	for i, m := range meta {
		var delta any
		if averages[i] != nil && previous[i] != nil {
			delta = *averages[i] - *previous[i]
		}
		if averages[i] != nil {
			scoredValues = append(scoredValues, *averages[i])
		}
		items[i] = jsonx.Obj(
			"key", m.key,
			"label", hc.T(m.fa, m.en),
			"icon", m.icon,
			"percent", intOrNil(averages[i]),
			"previous_percent", intOrNil(previous[i]),
			"delta", delta,
		)
	}
	return &Rendered{
		Key: s.Key(), Type: "weekly_summary", Order: s.Order(),
		Title: str(hc.T("خلاصه هفته", "Weekly summary")),
		Data: jsonx.Obj(
			"items", items,
			"range", jsonx.Obj("from", hc.Date.AddDays(-(summaryWindow-1)).String(), "to", hc.Date.String()),
			"logged_days", len(logs),
			"previous_logged_days", len(previousLogs),
			"overall_percent", intOrNil(averageOrNil(scoredValues)),
		),
		Action: action("view_all", hc.T("مشاهده کامل", "View all")),
	}, nil
}

// ---------------------------------------------------------------------------
// 13. vitals (VitalsSection.php)

type vitalsSection struct{ allModes }

func (vitalsSection) Key() string { return "vitals" }
func (vitalsSection) Order() int  { return 130 }

func (s vitalsSection) Build(hc *Context) (*Rendered, error) {
	var heartRate, bloodPressure, bloodSugar any
	if log := hc.DailyLog(); log != nil {
		if log.HeartRate.Valid {
			heartRate = int(log.HeartRate.Int16)
		}
		if log.SystolicPressure.Valid && log.DiastolicPressure.Valid {
			bloodPressure = fmt.Sprintf("%d/%d", log.SystolicPressure.Int16, log.DiastolicPressure.Int16)
		}
		if log.BloodSugar.Valid {
			bloodSugar = jsonx.Float(phpval.ToFloat(log.BloodSugar.String))
		}
	}
	items := []*jsonx.OrderedMap{
		jsonx.Obj("key", "heart_rate", "label", hc.T("ضربان قلب", "Heart rate"), "value", heartRate, "unit", "bpm", "icon", "heart-pulse"),
		jsonx.Obj("key", "blood_pressure", "label", hc.T("فشار خون", "Blood pressure"), "value", bloodPressure, "unit", "mmHg", "icon", "gauge"),
		jsonx.Obj("key", "blood_sugar", "label", hc.T("قند خون", "Blood sugar"), "value", bloodSugar, "unit", "mg/dl", "icon", "droplet"),
	}
	return &Rendered{
		Key: s.Key(), Type: "vitals", Order: s.Order(),
		Title: str(hc.T("وضعیت امروز", "Today's vitals")),
		Data: jsonx.Obj(
			"items", items,
			"has_data", heartRate != nil || bloodPressure != nil || bloodSugar != nil,
			"date", hc.Date.String(),
		),
		Action: action("view_all", hc.T("مشاهده کامل", "View all")),
	}, nil
}

// ---------------------------------------------------------------------------
// 14. status_charts (StatusChartsSection.php)

type statusChartsSection struct{ allModes }

func (statusChartsSection) Key() string { return "status_charts" }
func (statusChartsSection) Order() int  { return 140 }

func (s statusChartsSection) Build(hc *Context) (*Rendered, error) {
	logs := hc.RecentLogs(7)
	series := []struct {
		key, fa, en, unit string
		value             func(cyclestore.DailyHealthLog) (any, bool)
	}{
		{"heart_rate", "ضربان قلب", "Heart rate", "bpm", func(l cyclestore.DailyHealthLog) (any, bool) {
			return jsonx.Float(float64(l.HeartRate.Int16)), l.HeartRate.Valid
		}},
		{"blood_pressure", "فشار خون", "Blood pressure", "mmHg", func(l cyclestore.DailyHealthLog) (any, bool) {
			return jsonx.Float(float64(l.SystolicPressure.Int16)), l.SystolicPressure.Valid
		}},
		{"blood_sugar", "قند خون", "Blood sugar", "mg/dl", func(l cyclestore.DailyHealthLog) (any, bool) {
			if !l.BloodSugar.Valid {
				return nil, false
			}
			if phpval.IsNumericString(l.BloodSugar.String) {
				return jsonx.Float(phpval.ToFloat(l.BloodSugar.String)), true
			}
			return l.BloodSugar.String, true
		}},
	}
	charts := make([]*jsonx.OrderedMap, len(series))
	hasAny := false
	for i, se := range series {
		points := []*jsonx.OrderedMap{}
		for _, l := range logs {
			v, ok := se.value(l)
			if !ok {
				continue
			}
			hasAny = true
			points = append(points, jsonx.Obj("date", l.LogDate.String(), "value", v))
		}
		charts[i] = jsonx.Obj("key", se.key, "label", hc.T(se.fa, se.en), "unit", se.unit, "points", points)
	}
	if !hasAny {
		return nil, nil
	}
	return &Rendered{
		Key: s.Key(), Type: "status_charts", Order: s.Order(),
		Title: str(hc.T("وضعیت امروز", "Status trends")),
		Data: jsonx.Obj(
			"charts", charts,
			"range", jsonx.Obj("from", hc.Date.AddDays(-6).String(), "to", hc.Date.String()),
		),
		Action: action("view_all", hc.T("مشاهده کامل", "View all")),
	}, nil
}

// ---------------------------------------------------------------------------
// 15. articles (ArticlesSection.php)

type articlesSection struct{ allModes }

func (articlesSection) Key() string { return "articles" }
func (articlesSection) Order() int  { return 150 }

// phaseCandidates is [subphase, canonical(subphase), phase] without empties, de-duplicated.
func phaseCandidates(hc *Context) ([]string, error) {
	sub, err := hc.Subphase()
	if err != nil {
		return nil, err
	}
	phase, err := hc.Phase()
	if err != nil {
		return nil, err
	}
	canonical := ""
	if sub != "" {
		if sp, ok := enums.CycleSubphaseFrom(sub); ok {
			canonical = string(sp.Canonical())
		}
	}
	var out []string
	for _, c := range []string{sub, canonical, phase} {
		if phpval.Truthy(c) && !slices.Contains(out, c) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s articlesSection) Build(hc *Context) (*Rendered, error) {
	candidates, err := phaseCandidates(hc)
	if err != nil {
		return nil, err
	}
	phases, err := json.Marshal(append([]string{}, candidates...))
	if err != nil {
		return nil, err
	}
	rows, err := hc.q().ListHomeArticles(hc.Ctx, store.ListHomeArticlesParams{HasPhases: b2i(len(candidates) > 0), Phases: string(phases)})
	if err != nil {
		return nil, fmt.Errorf("home: articles: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	items := make([]*jsonx.OrderedMap, len(rows))
	for i, a := range rows {
		var excerpt any
		if a.Excerpt.Valid {
			var s string
			if json.Unmarshal(i18n.Pick(a.Excerpt.V, hc.Locale, hc.DefaultLocale), &s) == nil {
				if txt, ok := sanitizer.PlainText(s); ok {
					excerpt = txt
				}
			}
		}
		var imageURL any
		switch {
		case a.ImagePath.Valid && phpval.Truthy(a.ImagePath.String):
			imageURL = content.PublicURL(hc.deps.AppURL, a.ImagePath.String)
		case a.ImageUrl.Valid:
			imageURL = a.ImageUrl.String
		}
		var phasesValue any = []any{}
		if v := decodedJSON(a.CyclePhases); v != nil {
			phasesValue = v
		}
		items[i] = jsonx.Obj(
			"id", a.ID,
			"slug", a.Slug,
			"title", localized(a.Title, hc.Locale, hc.DefaultLocale),
			"excerpt", excerpt,
			"read_time_minutes", nullInt16(a.ReadTimeMinutes),
			"image_url", imageURL,
			"category", nullString(a.Category),
			"cycle_phases", phasesValue,
		)
	}
	return &Rendered{
		Key: s.Key(), Type: "articles", Order: s.Order(),
		Title:  str(hc.T("بر اساس سیکل فعلی شما", "Based on your current cycle")),
		Data:   jsonx.Obj("items", items),
		Action: action("view_more", hc.T("مشاهده بیشتر", "View more")),
	}, nil
}

// ---------------------------------------------------------------------------
// 16. my_cycles (MyCyclesSection.php)

type myCyclesSection struct{ cycleOnly }

func (myCyclesSection) Key() string { return "my_cycles" }
func (myCyclesSection) Order() int  { return 160 }

const myCyclesPreviousLimit = 12

func (s myCyclesSection) Build(hc *Context) (*Rendered, error) {
	calc, err := hc.CycleData()
	if err != nil {
		return nil, err
	}
	start, ok, err := hc.CurrentCycleStart()
	if err != nil || calc == nil || !ok {
		return nil, err
	}
	digest := hc.HistoryDigest()
	m := hc.Metrics()

	current := jsonx.Obj(
		"id", nil,
		"cycle_day", calc.Day,
		"started_at", start.String(),
		"period_end_date", nil,
		"period_length", nil,
		"is_ongoing", false,
		"cycle_length", calc.CycleLength,
		"cycle_length_source", string(m.CycleLengthSource),
	)
	if rec, found := digest.StartingOn(start); found {
		var end any
		if !rec.End.IsZero() {
			end = rec.End.String()
		}
		current.Set("id", rec.ID).
			Set("period_end_date", end).
			Set("period_length", intOrNil(rec.PeriodLength)).
			Set("is_ongoing", rec.End.IsZero())
	}
	previous := []*jsonx.OrderedMap{}
	for _, c := range digest.Previous(myCyclesPreviousLimit) {
		previous = append(previous, c.JSON())
	}
	return &Rendered{
		Key: s.Key(), Type: "my_cycles", Order: s.Order(),
		Title: str(hc.T("سیکل‌های من", "My cycles")), //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from PHP
		Data: jsonx.Obj(
			"current", current,
			"previous_count", len(digest.Previous(-1)),
			"previous", previous,
			"averages", jsonx.Obj(
				"cycle_length", intOrNil(digest.AverageCycleLength()),
				"period_length", intOrNil(digest.AveragePeriodLength()),
				"based_on_cycles", len(digest.ValidCycleLengths()),
			),
		),
		Action: action("add_previous", hc.T("ثبت سیکل‌های قبلی", "Add previous cycles")), //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from PHP
	}, nil
}

// ---------------------------------------------------------------------------
// 17. cycle_summary (CycleSummarySection.php)

type cycleSummarySection struct{ cycleOnly }

func (cycleSummarySection) Key() string { return "cycle_summary" }
func (cycleSummarySection) Order() int  { return 170 }

var (
	normalCycleRange  = [2]int{21, 35}
	normalPeriodRange = [2]int{2, 8}
)

const minCyclesForAggregate = 2

// summaryItem is CycleSummarySection::item().
type summaryItem struct {
	key, label               string
	value, valueMin, valueMx *int
	text, unit, unitLabel    *string
	status                   string
	hint                     *string
	normalRange              *[2]int
}

func (it summaryItem) json(hc *Context) *jsonx.OrderedMap {
	var nr any
	if it.normalRange != nil {
		nr = jsonx.Obj("min", it.normalRange[0], "max", it.normalRange[1])
	}
	return jsonx.Obj(
		"key", it.key,
		"label", it.label,
		"value", intOrNil(it.value),
		"value_min", intOrNil(it.valueMin),
		"value_max", intOrNil(it.valueMx),
		"text", strOrNil(it.text),
		"unit", strOrNil(it.unit),
		"unit_label", strOrNil(it.unitLabel),
		"status", it.status,
		"status_label", statusLabel(it.status, hc),
		"hint", strOrNil(it.hint),
		"normal_range", nr,
	)
}

func rangeStatus(v *int, r [2]int) string {
	if v == nil {
		return "unknown"
	}
	if *v >= r[0] && *v <= r[1] {
		return "normal"
	}
	return "outside_range"
}

func statusLabel(status string, hc *Context) string {
	switch status {
	case "normal":
		return hc.T("در محدودهٔ معمول", "Within the usual range")
	case "outside_range":
		return hc.T("خارج از محدودهٔ معمول", "Outside the usual range")
	}
	return hc.T("هنوز مشخص نیست", "Not known yet")
}

func numericItem(hc *Context, key, label string, value *int, r [2]int, hint, emptyHint *string) summaryItem {
	h := hint
	if value == nil {
		h = emptyHint
	}
	rr := r
	return summaryItem{
		key: key, label: label, value: value,
		unit: str("days"), unitLabel: str(hc.T("روز", "days")),
		status: rangeStatus(value, r), hint: h, normalRange: &rr,
	}
}

func (s cycleSummarySection) Build(hc *Context) (*Rendered, error) {
	calc, err := hc.CycleData()
	if err != nil || calc == nil {
		return nil, err
	}
	digest := hc.HistoryDigest()
	m := hc.Metrics()
	lastCycle, lastPeriod := digest.LastCycleLength(), digest.LastPeriodLength()
	validCycles := len(digest.ValidCycleLengths())

	items := []summaryItem{
		numericItem(hc, "last_cycle_length", hc.T("طول سیکل قبلی", "Last cycle length"), lastCycle, normalCycleRange, nil,
			str(hc.T("با ثبت پریود بعدی، طول این سیکل مشخص می‌شود.", "Log your next period to learn this cycle’s length."))), //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from PHP
		numericItem(hc, "last_period_duration", hc.T("مدت پریود قبلی", "Last period duration"), lastPeriod, normalPeriodRange, nil,
			str(hc.T("پایان پریود را ثبت کنید تا مدت آن محاسبه شود.", "Record when your period ends to see its duration."))),
		regularityItem(hc, m.RegularityStatus, m.CycleVariabilityRange),
	}
	if validCycles >= minCyclesForAggregate {
		items = append(items, numericItem(hc, "average_cycle_length", hc.T("میانگین طول سیکل", "Average cycle length"),
			digest.AverageCycleLength(), normalCycleRange,
			str(hc.T("بر پایه "+hc.Num(validCycles)+" سیکل ثبت‌شده", "Based on "+strconv.Itoa(validCycles)+" recorded cycles")), nil)) //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from PHP
		shortest, longest := digest.ShortestCycle(), digest.LongestCycle()
		status := "outside_range"
		if rangeStatus(shortest, normalCycleRange) == "normal" && rangeStatus(longest, normalCycleRange) == "normal" {
			status = "normal"
		}
		items = append(items, summaryItem{
			key: "cycle_length_range", label: hc.T("کوتاه‌ترین تا بلندترین سیکل", "Shortest to longest cycle"), //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from PHP
			valueMin: shortest, valueMx: longest, unit: str("days"), unitLabel: str(hc.T("روز", "days")), status: status,
		})
	}
	list := make([]*jsonx.OrderedMap, len(items))
	for i, it := range items {
		list[i] = it.json(hc)
	}
	return &Rendered{
		Key: s.Key(), Type: "cycle_summary", Order: s.Order(),
		Title: str(hc.T("خلاصه سیکل", "Cycle summary")),
		Data: jsonx.Obj(
			"items", list,
			"based_on_cycles", validCycles,
			"has_history", lastCycle != nil || lastPeriod != nil,
			"normal_ranges", jsonx.Obj(
				"cycle_length", jsonx.Obj("min", normalCycleRange[0], "max", normalCycleRange[1]),
				"period_duration", jsonx.Obj("min", normalPeriodRange[0], "max", normalPeriodRange[1]),
			),
		),
		Action: action("view_more", hc.T("مشاهده بیشتر", "View more")),
	}, nil
}

// regularityItem is CycleSummarySection::regularityItem().
func regularityItem(hc *Context, status enums.RegularityStatus, spread *int) summaryItem {
	if status == enums.RegularityStatusNotEnoughData {
		spread = nil
	}
	it := summaryItem{
		key: "cycle_variability", label: hc.T("نوسان طول سیکل", "Cycle length variation"),
		value: spread, text: str(status.Label(hc.Locale)),
	}
	switch status {
	case enums.RegularityStatusRelativelyRegular:
		it.status = "normal"
	case enums.RegularityStatusIrregularPossible:
		it.status = "outside_range"
	default:
		it.status = "unknown"
	}
	if spread != nil {
		it.unit, it.unitLabel = str("days"), str(hc.T("روز", "days"))
		it.hint = str(hc.T("اختلاف کوتاه‌ترین و بلندترین سیکل اخیر: "+hc.Num(*spread)+" روز", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from PHP
			"Recent cycles differ by "+strconv.Itoa(*spread)+" days"))
	} else {
		it.hint = str(hc.T("برای سنجش نظم، حداقل سه سیکل کامل لازم است.",
			"At least three complete cycles are needed to judge regularity."))
	}
	return it
}

// ---------------------------------------------------------------------------
// helpers

// localized is HasLocalizedContent::localized() echoed as Laravel would.
func localized(raw json.RawMessage, locale, def string) any {
	return decodeRaw(i18n.Pick(raw, locale, def))
}

// localizedNull is localized() of a nullable column (NULL → null).
func localizedNull(v db.NullRawJSON, locale, def string) any {
	if !v.Valid {
		return nil
	}
	return localized(v.V, locale, def)
}

// decodedJSON is an `array` cast attribute echoed by json_encode (NULL → null).
func decodedJSON(v db.NullRawJSON) any {
	if !v.Valid {
		return nil
	}
	return decodeRaw(v.V)
}

func decodeRaw(raw json.RawMessage) any {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}
	v, err := phpval.Decode(raw)
	if err != nil {
		return nil
	}
	return phpval.Packed(v)
}

func nullString(v sql.NullString) any {
	if !v.Valid {
		return nil
	}
	return v.String
}

func nullInt16(v sql.NullInt16) any {
	if !v.Valid {
		return nil
	}
	return int(v.Int16)
}

func strOrNil(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func b2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// isoOrNil is `$t?->toIso8601String()`.
func isoOrNil(t sql.NullTime) any {
	if !t.Valid {
		return nil
	}
	return jsonx.ISO8601(t.Time.In(civildate.Tehran))
}
