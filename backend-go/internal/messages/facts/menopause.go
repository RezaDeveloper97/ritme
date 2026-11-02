package facts

import (
	"context"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Menopause message facts (T-M10-09, D-166; SPEC §5.4.1): the signals the menopause messages engine detected per
// request (internal/messages/menomessages, D-43) as facts, so every menopause message is a message rule (rule_key
// meno.<name>) the admin can switch off, re-word or re-threshold. The detection inputs stay the menopause API's own
// rules (menopause.Service.Signals: the stage rule, the 30-day bleeding flag, the newest questionnaire, the checkups
// plan, the active HRT items); the thresholds that used to be constants (score rise ≥ 4, review within 14 days, two
// checkup states) are conditions of the seeded rules.
//
// Every condition fact is present whenever the input is (a user in menopause mode, or the explicit «nothing» input of
// another mode), so an open inbox row resolves when its pattern ends; without an input every fact is absent. The text
// variables (meno_*, Variable: true) are the evidence of a hit: ISO dates and plain numbers (language-free), the
// checkup title in the language of the evaluating request and the treatment name as the user typed it.

// MenopauseInput is one user's menopause message input on the target day (menomessages.Input builds it from the
// menopause signals).
type MenopauseInput struct {
	// Stage is her effective stage (peri | meno | post | unsure — unsure when she did not answer); "" = not in
	// menopause mode (the stage fact is then absent and every other fact holds its «nothing» value).
	Stage string
	// Bleeding is the menopause home's bleeding flag (bleeding or spotting of the last 30 days at stage meno / post);
	// BleedingOn the newest such day (zero = unknown).
	Bleeding   bool
	BleedingOn civildate.Date
	// Overdue and Due count the menopause checkups of her plan in those states; the titles are the first one's.
	Overdue, Due           int
	OverdueTitle, DueTitle string
	// ScoreRise is the rise of her newest questionnaire (this or the previous Jalali month) over the one before it;
	// 0 without a fresh questionnaire with a previous one. ScoreTotal / ScorePrevious are its totals.
	ScoreRise, ScoreTotal, ScorePrevious int
	// HRTDays are the days from today to the nearest review_on (today or later) of an active HRT item; -1 = none.
	// HRTName / HRTOn are that item's name and review day.
	HRTDays int
	HRTName string
	HRTOn   civildate.Date
}

// MenopauseSource reads a user's menopause message input on day (nil = no input: every fact absent). Any Sources
// field that also implements it produces the group (the feed's fact source once it wires the menopause signals).
type MenopauseSource interface {
	MenopauseInput(ctx context.Context, day civildate.Date) (*MenopauseInput, error)
}

// GroupMenopause is the menopause message producer group.
const GroupMenopause = "menopause"

// MenopauseStages are the values of meno.stage.
var MenopauseStages = []string{"peri", "meno", "post", "unsure"}

// NoHRTReview is meno.hrt_review_in_days without an upcoming review.
const NoHRTReview = -1

func init() {
	g := GroupMenopause
	cond := func(name string, t Type, enum []string, unit, fa, en string) Def {
		return Def{Name: name, Type: t, Enum: enum, Unit: unit, Group: g, Description: desc(fa, en)}
	}
	v := func(name string, t Type, fa, en string) Def {
		return Def{Name: name, Type: t, Group: g, Variable: true, Description: desc(fa, en)}
	}
	Register(
		cond("meno.stage", TypeEnum, MenopauseStages, "", "مرحلهٔ یائسگی (بدون پاسخ: unsure؛ فقط در حالت یائسگی)",
			"The menopause stage (unanswered: unsure; menopause mode only)."),
		cond("meno.postmenopausal_bleeding", TypeBool, nil, "",
			"خونریزی یا لکه\u200cبینی در ۳۰ روز اخیر در مرحلهٔ یائسگی یا پس از یائسگی (۱۲ ماه بعد از آخرین پریود)",
			"Bleeding or spotting in the last 30 days at stage meno or post (12 months after the last period)."),
		cond("meno.checkups_overdue", TypeInt, nil, "", "تعداد چکاپ\u200cهای یائسگیِ عقب\u200cافتاده در برنامهٔ چکاپ",
			"Menopause checkups of the plan that are overdue."),
		cond("meno.checkups_due", TypeInt, nil, "", "تعداد چکاپ\u200cهای یائسگی که وقتشان رسیده (هرگز انجام\u200cنشده هم)",
			"Menopause checkups of the plan that are due (never done included)."),
		cond("meno.score_rise", TypeInt, nil, "point",
			"افزایش امتیاز تازه\u200cترین پرسش\u200cنامهٔ ماهانه (این ماه یا ماه قبل) نسبت به قبلی؛ بدون آن ۰",
			"Rise of the newest monthly score (this or the previous month) over the one before; 0 without one."),
		cond("meno.hrt_review_in_days", TypeInt, nil, "day",
			"روزهای مانده تا نزدیک\u200cترین بازبینی درمان هورمونی فعال (امروز به بعد)؛ بدون بازبینی −۱",
			"Days to the nearest review of an active HRT item (today or later); -1 without one."),

		v("meno_bleeding_on", TypeDate, "متغیر متن: آخرین روز خونریزی یا لکه\u200cبینی", "Text variable: the last bleeding or spotting day."),
		v("meno_overdue_checkup", TypeEnum, "متغیر متن: عنوان اولین چکاپ عقب\u200cافتاده", "Text variable: the first overdue checkup's title."),
		v("meno_due_checkup", TypeEnum, "متغیر متن: عنوان اولین چکاپی که وقتش رسیده", "Text variable: the first due checkup's title."),
		v("meno_score_total", TypeInt, "متغیر متن: امتیاز تازه\u200cترین پرسش\u200cنامه", "Text variable: the newest score."),
		v("meno_score_previous", TypeInt, "متغیر متن: امتیاز پرسش\u200cنامهٔ قبلی", "Text variable: the previous score."),
		v("meno_score_rise", TypeInt, "متغیر متن: مقدار افزایش امتیاز", "Text variable: the score rise."),
		v("meno_hrt_name", TypeEnum, "متغیر متن: نام درمان هورمونی", "Text variable: the HRT item's name."),
		v("meno_hrt_on", TypeDate, "متغیر متن: روز بازبینی درمان", "Text variable: the treatment review day."),
	)
	RegisterGroup(g, func(ctx context.Context, f *Facts) (Values, error) {
		src, ok := sourceOf[MenopauseSource](f.src)
		if !ok {
			return nil, nil
		}
		in, err := src.MenopauseInput(ctx, f.today)
		if err != nil || in == nil {
			return nil, err
		}
		return MenopauseValues(in), nil
	})
}

// MenopauseValues are the menopause facts and text variables of in.
func MenopauseValues(in *MenopauseInput) Values {
	out := Values{
		"meno.postmenopausal_bleeding": in.Bleeding,
		"meno.checkups_overdue":        float64(in.Overdue),
		"meno.checkups_due":            float64(in.Due),
		"meno.score_rise":              float64(in.ScoreRise),
		"meno.hrt_review_in_days":      float64(in.HRTDays),
	}
	if in.Stage != "" {
		out["meno.stage"] = in.Stage
	}
	if in.Bleeding {
		out["meno_bleeding_on"] = dateOrEmpty(in.BleedingOn)
	}
	if in.Overdue > 0 {
		out["meno_overdue_checkup"] = in.OverdueTitle
	}
	if in.Due > 0 {
		out["meno_due_checkup"] = in.DueTitle
	}
	if in.ScoreRise != 0 {
		out["meno_score_total"], out["meno_score_previous"], out["meno_score_rise"] =
			float64(in.ScoreTotal), float64(in.ScorePrevious), float64(in.ScoreRise)
	}
	if in.HRTDays >= 0 {
		out["meno_hrt_name"], out["meno_hrt_on"] = in.HRTName, dateOrEmpty(in.HRTOn)
	}
	return out
}

func dateOrEmpty(d civildate.Date) string {
	if d.IsZero() {
		return ""
	}
	return d.String()
}

// sourceOf is the first Sources field that implements T (a module's optional extra source: the feed's StoreSource
// implements every interface, a module's own evaluation passes only its source).
func sourceOf[T any](s Sources) (T, bool) {
	for _, v := range []any{s.Account, s.Logs, s.Vitals, s.Pregnancy, s.Postpartum, s.Contraception, s.Engine} {
		if v == nil {
			continue
		}
		if t, ok := v.(T); ok {
			return t, true
		}
	}
	var zero T
	return zero, false
}
