package view

import (
	"strconv"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

// Port of DailyCardBuilder.php / DailyCard.php. The copy is hardcoded fa/en exactly as in PHP:
// locale "fa" gets Persian (digits via strtr), every other locale gets English. This is a
// deliberate PHP behaviour (domain-inventory §8.7), not the language registry.

const (
	actionLogSymptoms        = "log_symptoms"
	actionLogPeriodEnd       = "log_period_end"
	actionLogPeriodStart     = "log_period_start"
	actionConfirmPeriodStart = "confirm_period_start"
	actionPeriodNotStarted   = "period_not_started"
	actionStillBleeding      = "still_bleeding"
	actionViewDetails        = "view_details"
	actionSetReminder        = "set_reminder"
	actionCompleteDay        = "complete_day"
)

// Action is a `{type, label}` call to action.
type Action struct {
	Type  string `json:"type"`
	Label string `json:"label"`
}

// DailyCard is DailyCard::toApiArray() (fields in PHP key order).
type DailyCard struct {
	Title            string               `json:"title"`
	Subtitle         string               `json:"subtitle"`
	DataStatus       enums.DataStatus     `json:"data_status"`
	FertilityLevel   enums.FertilityLevel `json:"fertility_level"`
	FertilityLabel   string               `json:"fertility_label"`
	Badges           []string             `json:"badges"`
	PrimaryAction    *Action              `json:"primary_action"`
	SecondaryActions []Action             `json:"secondary_actions"`
}

// CardInput is the argument list of DailyCardBuilder::build.
type CardInput struct {
	Selected           civildate.Date
	Today              civildate.Date
	CycleDay           int
	Subphase           enums.CycleSubphase
	PredictedNextStart civildate.Date
	OpenPeriod         OpenPeriodState
	LoggedPeriodDay    *int
	LoggedPeriodClosed bool
	// EstimatedOvulation is the zero Date for PHP null.
	EstimatedOvulation civildate.Date
	Locale             string
	// NoFertilityCopy (teen, menopause — B-N2-11b) keeps the predicted-day title and subtitle off the
	// fertile window / ovulation: those days read as plain cycle days or the period countdown.
	NoFertilityCopy bool
}

// cardBuilder carries the locale for one build (PHP $currentLocale).
type cardBuilder struct{ locale string }

func (b cardBuilder) t(fa, en string) string {
	if b.locale == "fa" {
		return fa
	}
	return en
}

func (b cardBuilder) num(v int) string {
	if b.locale != "fa" {
		return strconv.Itoa(v)
	}
	return phpround.PersianDigits(strconv.Itoa(v))
}

func (b cardBuilder) action(typ, fa, en string) *Action {
	return &Action{Type: typ, Label: b.t(fa, en)}
}

// BuildDailyCard is DailyCardBuilder::build: title chain actual > incomplete > needs_confirmation > predicted.
func BuildDailyCard(in CardInput) DailyCard {
	b := cardBuilder{locale: in.Locale}
	sel, ref := in.Selected, in.Today
	isFuture := sel.After(ref)
	isToday := sel == ref

	if in.LoggedPeriodDay != nil {
		if in.LoggedPeriodClosed {
			return b.actualPeriodDay(in.CycleDay, *in.LoggedPeriodDay)
		}
		return b.openPeriodDay(in.CycleDay, *in.LoggedPeriodDay, in.OpenPeriod, isToday)
	}

	daysUntilPeriod := sel.DiffDays(in.PredictedNextStart)

	if !isFuture {
		if daysUntilPeriod == 0 {
			return b.periodDue(in.CycleDay)
		}
		if daysUntilPeriod < 0 {
			return b.periodOverdue(-daysUntilPeriod)
		}
		if isMenstrualSubphase(in.Subphase) {
			return b.predictedPeriodDay(in.CycleDay, isToday)
		}
	}

	return b.predictedDay(in.CycleDay, in.Subphase, daysUntilPeriod, isFuture, isToday, sel, in.EstimatedOvulation, !in.NoFertilityCopy)
}

func (b cardBuilder) card(title, subtitle string, status enums.DataStatus, fertility enums.FertilityLevel, badges []string, primary *Action, secondary ...*Action) DailyCard {
	sec := make([]Action, 0, len(secondary))
	for _, a := range secondary {
		sec = append(sec, *a)
	}
	return DailyCard{
		Title:            title,
		Subtitle:         subtitle,
		DataStatus:       status,
		FertilityLevel:   fertility,
		FertilityLabel:   fertility.Label(b.locale),
		Badges:           badges,
		PrimaryAction:    primary,
		SecondaryActions: sec,
	}
}

func (b cardBuilder) actualPeriodDay(cycleDay, periodDay int) DailyCard {
	return b.card(
		b.t("روز "+b.num(periodDay)+" پریود", "Day "+strconv.Itoa(periodDay)+" of your period"),
		b.t("این روز به\u200cعنوان روز پریود ثبت شده است. مراقبت ملایم می\u200cتواند کمک\u200cکننده باشد.",
			"This day is logged as a period day. Gentle self-care can help."),
		enums.DataStatusActual, enums.FertilityLevelLow,
		b.badges(cycleDay, enums.DataStatusActual),
		b.action(actionLogSymptoms, "ثبت علائم امروز", "Log today's symptoms"),
		b.detailsAction(true, true),
	)
}

func (b cardBuilder) openPeriodDay(cycleDay, periodDay int, open OpenPeriodState, isToday bool) DailyCard {
	if isToday && open.EndOverdue {
		return b.card(
			b.t("پایان پریود هنوز ثبت نشده", "Period end not logged yet"),
			b.t("برای دقیق\u200cتر شدن تقویم، مشخص کن خون\u200cریزی چه زمانی تمام شده است.",
				"To keep your calendar accurate, tell us when the bleeding stopped."),
			enums.DataStatusIncomplete, enums.FertilityLevelLow,
			b.badges(cycleDay, enums.DataStatusIncomplete),
			b.action(actionLogPeriodEnd, "ثبت پایان پریود", "Log period end"),
			b.action(actionStillBleeding, "هنوز ادامه دارد", "Still ongoing"),
			b.detailsAction(false, true),
		)
	}

	return b.card(
		b.t("روز "+b.num(periodDay)+" پریود", "Day "+strconv.Itoa(periodDay)+" of your period"),
		b.t("پریود هنوز باز است. هر وقت خون\u200cریزی تمام شد، پایان آن را ثبت کن.",
			"This period is still open. Log its end once the bleeding stops."),
		enums.DataStatusIncomplete, enums.FertilityLevelLow,
		b.badges(cycleDay, enums.DataStatusIncomplete),
		b.action(actionLogSymptoms, "ثبت علائم امروز", "Log today's symptoms"),
		b.action(actionLogPeriodEnd, "ثبت پایان پریود", "Log period end"),
		b.detailsAction(!isToday, isToday),
	)
}

func (b cardBuilder) periodDue(cycleDay int) DailyCard {
	return b.card(
		b.t("موعد پریود پیش\u200cبینی\u200cشده رسیده", "Your predicted period is due"),
		b.t("تا وقتی شروع پریود را ثبت نکنی، این وضعیت فقط یک پیش\u200cبینی است.",
			"Until you log the start, this is only a prediction."),
		enums.DataStatusNeedsConfirmation, enums.FertilityLevelLow,
		b.badges(cycleDay, enums.DataStatusNeedsConfirmation),
		b.action(actionConfirmPeriodStart, "پریودت شروع شده؟", "Has your period started?"),
		b.action(actionPeriodNotStarted, "هنوز شروع نشده", "Not started yet"),
		b.detailsAction(false, true),
	)
}

func isMenstrualSubphase(s enums.CycleSubphase) bool {
	return s == enums.CycleSubphaseMenstruation || s == enums.CycleSubphaseMenstrual || s == enums.CycleSubphaseMenstrualPossible
}

func (b cardBuilder) predictedPeriodDay(cycleDay int, isToday bool) DailyCard {
	return b.card(
		b.t("روز "+b.num(cycleDay)+" پریود پیش\u200cبینی\u200cشده", "Day "+strconv.Itoa(cycleDay)+" of your predicted period"),
		b.t("این فقط یک پیش\u200cبینی است. اگر پریودت از این روز شروع شده، ثبتش کن تا تقویم دقیق\u200cتر شود.",
			"This is only a prediction. If your period started on this day, log it to keep your calendar accurate."),
		enums.DataStatusNeedsConfirmation, enums.FertilityLevelLow,
		b.badges(cycleDay, enums.DataStatusNeedsConfirmation),
		b.action(actionConfirmPeriodStart, "پریودم از این روز شروع شد", "My period started this day"),
		b.action(actionPeriodNotStarted, "شروع نشده بود", "It didn't start"),
		b.detailsAction(!isToday, isToday),
	)
}

func (b cardBuilder) periodOverdue(daysSince int) DailyCard {
	title := b.t("هنوز شروع پریود ثبت نشده", "Period start not logged yet")
	if daysSince <= 2 {
		title = b.t(b.num(daysSince)+" روز از موعد پیش\u200cبینی\u200cشده گذشته", strconv.Itoa(daysSince)+" day(s) past the predicted date")
	}
	return b.card(
		title,
		b.t("چند روز اختلاف با پیش\u200cبینی طبیعی است، مخصوصاً اگر زمان تخمک\u200cگذاری جابه\u200cجا شده باشد.",
			"A few days off from the prediction is normal, especially if ovulation shifted."),
		enums.DataStatusNeedsConfirmation, enums.FertilityLevelLow,
		[]string{enums.DataStatusNeedsConfirmation.Label(b.locale)},
		b.action(actionLogPeriodStart, "ثبت شروع پریود", "Log period start"),
		b.action(actionPeriodNotStarted, "هنوز شروع نشده", "Not started yet"),
		b.detailsAction(true, false),
	)
}

// predictedDay is the predicted-day card. fertility false (teen, menopause — B-N2-11b) skips the fertile
// window / ovulation copy: those days read as the period countdown or a plain cycle day.
func (b cardBuilder) predictedDay(cycleDay int, sub enums.CycleSubphase, daysUntilPeriod int, isFuture, isToday bool, selected, estimatedOvulation civildate.Date, fertility bool) DailyCard {
	hasDaysToFertile := !estimatedOvulation.IsZero()
	daysToFertile := 0
	if hasDaysToFertile {
		daysToFertile = selected.DiffDays(estimatedOvulation.AddDays(-5))
	}

	var title, subtitle string
	switch {
	case daysUntilPeriod >= 3 && daysUntilPeriod <= 7:
		title = b.t("حدود "+b.num(daysUntilPeriod)+" روز تا پریود بعدی", "About "+strconv.Itoa(daysUntilPeriod)+" days to your next period")
		subtitle = b.t("در روزهای پایانی چرخه، انرژی، خواب یا خلق ممکن است تغییر کند.", "Late in the cycle, energy, sleep or mood may shift for some.")
	case daysUntilPeriod == 2:
		title = b.t("حدود ۲ روز تا پریود بعدی", "About 2 days to your next period")
		subtitle = b.t("پریود احتمالاً نزدیک است.", "Your period is likely close.")
	case daysUntilPeriod == 1:
		title = b.t("احتمالاً پریود نزدیک است", "Your period is likely near")
		subtitle = b.t("ممکن است به\u200cزودی خون\u200cریزی شروع شود.", "Bleeding may start soon.")
	case fertility && sub == enums.CycleSubphaseFertileRising:
		title = b.t("پنجره باروری ممکن است نزدیک باشد", "Your fertile window may be approaching")
		subtitle = b.fertileNote()
	case fertility && sub == enums.CycleSubphaseHighFertility:
		title = b.t("احتمالاً در پنجره باروری هستی", "You're likely in your fertile window")
		subtitle = b.fertileNote()
	case fertility && sub == enums.CycleSubphaseOvulationLikely:
		title = b.t("تخمک\u200cگذاری احتمالی نزدیک است", "Ovulation is likely near")
		subtitle = b.fertileNote()
	case fertility && sub == enums.CycleSubphasePostOvulation:
		title = b.t("احتمالاً از پنجره باروری عبور کرده\u200cای", "You've likely passed your fertile window")
		subtitle = b.postOvulationNote() // D-27: Laravel reuses fertileNote() ("fertility is higher") on a low day
	case fertility && hasDaysToFertile && daysToFertile > 0:
		title = b.t(b.num(daysToFertile)+" روز تا پنجره باروری", strconv.Itoa(daysToFertile)+" day(s) to your fertile window")
		subtitle = b.t("بر اساس پیش\u200cبینی چرخه، پنجره باروری از حدود "+b.num(daysToFertile)+" روز دیگر شروع می\u200cشود.",
			"Based on your cycle prediction, your fertile window starts in about "+strconv.Itoa(daysToFertile)+" day(s).")
	case daysUntilPeriod > 7:
		title = b.t(b.num(daysUntilPeriod)+" روز تا پریود بعدی", strconv.Itoa(daysUntilPeriod)+" day(s) to your next period")
		subtitle = b.t("بر اساس پیش\u200cبینی، پریود بعدی حدود "+b.num(daysUntilPeriod)+" روز دیگر شروع می\u200cشود.",
			"Based on the prediction, your next period starts in about "+strconv.Itoa(daysUntilPeriod)+" day(s).")
	case isFuture && daysUntilPeriod <= 0:
		title = b.t("روز "+b.num(cycleDay)+" چرخه پیش\u200cبینی\u200cشده", "Day "+strconv.Itoa(cycleDay)+" of the predicted cycle")
		subtitle = b.t("این یک پیش\u200cبینی تقویمی است و ممکن است تغییر کند.", "This is a calendar prediction and may change.")
	default:
		title = b.t("روز "+b.num(cycleDay)+" چرخه", "Day "+strconv.Itoa(cycleDay)+" of your cycle")
		subtitle = b.t("یک روز معمول چرخه بر اساس پیش\u200cبینی.", "A typical predicted cycle day.")
	}

	return b.card(
		title, subtitle,
		enums.DataStatusPredicted, sub.FertilityLevelV11(), // D-26 (T-M2-34): Laravel uses the legacy fertilityLevel()
		b.badges(cycleDay, enums.DataStatusPredicted),
		b.predictedPrimaryAction(isFuture, isToday),
		b.detailsAction(!isToday && !isFuture, isToday),
	)
}

func (b cardBuilder) predictedPrimaryAction(isFuture, isToday bool) *Action {
	switch {
	case isFuture:
		return b.action(actionSetReminder, "یادآوری برای این روز", "Set a reminder")
	case isToday:
		return b.action(actionLogSymptoms, "ثبت علائم امروز", "Log today's symptoms")
	default:
		return b.action(actionCompleteDay, "تکمیل اطلاعات این روز", "Complete this day")
	}
}

func (b cardBuilder) fertileNote() string {
	return b.t("در این بازه احتمال باروری بر اساس پیش\u200cبینی چرخه بالاتر است، اما زمان واقعی تخمک\u200cگذاری می\u200cتواند متفاوت باشد.",
		"Fertility is estimated higher here based on your cycle, but real ovulation timing can differ.")
}

// postOvulationNote is the subtitle of the day after ovulation (fertility level low, §26). Laravel
// shows fertileNote() here, which says the chance is *higher* right under «کم» (D-27).
func (b cardBuilder) postOvulationNote() string {
	return b.t("بر اساس پیش\u200cبینی چرخه، پنجره باروری احتمالاً تمام شده و احتمال باروری از این روز کمتر است، اما زمان واقعی تخمک\u200cگذاری می\u200cتواند متفاوت باشد.",
		"Based on your cycle, your fertile window has likely closed and fertility is estimated lower from here, but real ovulation timing can differ.")
}

func (b cardBuilder) detailsAction(isPast, isToday bool) *Action {
	switch {
	case isPast:
		return b.action(actionViewDetails, "جزئیات این روز", "This day's details")
	case isToday:
		return b.action(actionViewDetails, "جزئیات امروز", "Today's details")
	default:
		return b.action(actionViewDetails, "جزئیات پیش\u200cبینی", "Prediction details")
	}
}

func (b cardBuilder) badges(cycleDay int, status enums.DataStatus) []string {
	return []string{
		b.t("روز "+b.num(cycleDay)+" چرخه", "Day "+strconv.Itoa(cycleDay)),
		status.Label(b.locale),
	}
}
