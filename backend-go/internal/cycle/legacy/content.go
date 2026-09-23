package legacy

import (
	"slices"
	"strconv"

	"github.com/ritme/backend-go/internal/cycle/recommendation"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

// TextFlags is generateTextFlags (HealthDataEngine.php:699): the bilingual today-page flags.
// finalProbability is the unrounded 0..0.35 value; the message shows round(p*100, 1) in PHP's
// float-to-string form ("12.5", "12").
func TextFlags(phase enums.CyclePhase, subphase enums.CycleSubphase, isFertile, isPms, isPeriodTomorrow bool,
	finalProbability float64, variability enums.CycleVariability,
) []TextFlag {
	var flags []TextFlag
	if isFertile {
		flags = append(flags, TextFlag{
			Key: "fertility_status",
			EN:  "You are in your fertile window. Pregnancy chance is higher.",
			FA:  "شما در پنجره باروری هستید. احتمال بارداری بیشتر است.",
		})
	}

	percent := phpround.String(phpround.Round(finalProbability*100, 1))
	flags = append(flags, TextFlag{
		Key: "probability_message",
		EN:  "Today's pregnancy probability is " + percent + "%.",
		FA:  "احتمال بارداری امروز " + percent + "٪ است.",
	})

	if isPms {
		flags = append(flags, TextFlag{
			Key: "pms_warning",
			EN:  "You are in the PMS window. You may experience mood changes and physical symptoms.",
			FA:  "شما در دوره PMS هستید. ممکن است تغییرات خلقی و علائم جسمی را تجربه کنید.",
		})
	}

	if isPeriodTomorrow {
		u := strconv.Itoa(variability.UncertaintyRange())
		flags = append(flags, TextFlag{
			Key: "period_prediction",
			EN:  "Your period may start soon (±" + u + " days based on your cycle regularity).",
			FA:  "پریود شما ممکن است به زودی شروع شود (±" + u + " روز بر اساس نظم سیکل شما).",
		})
	}

	flags = append(flags, TextFlag{
		Key: "phase_info",
		EN:  "Current phase: " + phase.Label("en") + " (" + subphase.Label("en") + ")",
		FA:  "فاز فعلی: " + phase.Label("fa") + " (" + subphase.Label("fa") + ")",
	})
	return flags
}

func tip(typ, en, fa string) recommendation.Tip {
	return recommendation.Tip{Type: typ, EN: en, FA: fa}
}

// phaseTips is getPhaseBasedTips (HealthDataEngine.php:780), the fallback for an install whose
// recommendations table has never been seeded.
func phaseTips(phase enums.CyclePhase, subphase enums.CycleSubphase) []recommendation.Tip {
	switch phase {
	case enums.CyclePhaseMenstruation:
		return []recommendation.Tip{
			tip("hydration", "Stay hydrated and drink plenty of water.", "آب کافی بنوشید و هیدراته بمانید."),
			tip("warmth", "Apply a warm compress to your lower abdomen to relieve cramps.", "کمپرس گرم روی شکم قرار دهید تا دردها کاهش یابد."),
			tip("rest", "Get enough rest and avoid strenuous activities.", "استراحت کافی داشته باشید و از فعالیت\u200cهای سنگین اجتناب کنید."),
			tip("nutrition", "Eat iron-rich foods like spinach and red meat.", "غذاهای غنی از آهن مثل اسفناج و گوشت قرمز بخورید."),
		}
	case enums.CyclePhaseFollicular:
		return []recommendation.Tip{
			tip("energy", "Your energy levels are rising. Great time for new projects!", "سطح انرژی شما در حال افزایش است. زمان مناسبی برای پروژه\u200cهای جدید!"),
			tip("exercise", "Perfect time for high-intensity workouts.", "زمان مناسبی برای ورزش\u200cهای سنگین است."),
			tip("nutrition", "Add protein and fresh vegetables to rebuild your energy stores.", "پروتئین و سبزیجات تازه بخورید تا ذخیره انرژی\u200cتان دوباره پر شود."),
			tip("mental_health", "A good window for planning and learning something new.", "فرصت خوبی برای برنامه\u200cریزی و یادگیری چیزهای تازه است."),
		}
	case enums.CyclePhaseOvulation:
		return []recommendation.Tip{
			tip("fertility", "Peak fertility days. If trying to conceive, this is the best time.", "روزهای اوج باروری. اگر قصد بارداری دارید، بهترین زمان است."),
			tip("energy", "You may feel more confident and social.", "ممکن است احساس اعتماد به نفس و اجتماعی بودن بیشتری داشته باشید."),
			tip("hydration", "Drink water regularly — hydration supports cervical fluid.", "مرتب آب بنوشید؛ هیدراته بودن به کیفیت ترشحات کمک می\u200cکند."),
			tip("sleep", "Keep a steady sleep schedule to support hormone balance.", "برنامه خواب منظمی داشته باشید تا تعادل هورمونی حفظ شود."),
		}
	case enums.CyclePhaseLuteal:
		if subphase == enums.CycleSubphaseLateLuteal || subphase == enums.CycleSubphasePmsPossible {
			return []recommendation.Tip{
				tip("pms", "PMS symptoms may appear. Practice self-care and relaxation.", "علائم PMS ممکن است ظاهر شوند. مراقبت از خود و آرامش را تمرین کنید."),
				tip("nutrition", "Reduce salt and caffeine intake to minimize bloating.", "مصرف نمک و کافئین را کاهش دهید تا نفخ کم شود."),
				tip("mood", "Take breaks and practice deep breathing if feeling irritable.", "استراحت کنید و اگر احساس تحریک\u200cپذیری دارید تنفس عمیق تمرین کنید."),
			}
		}
		return []recommendation.Tip{
			tip("nutrition", "Focus on foods rich in magnesium and vitamin B6.", "روی غذاهای غنی از منیزیم و ویتامین B6 تمرکز کنید."),
			tip("exercise", "Switch to gentler movement like yoga, pilates or walking.", "به حرکات ملایم\u200cتر مثل یوگا، پیلاتس یا پیاده\u200cروی رو بیاورید."),
			tip("sleep", "Go to bed a little earlier — energy dips in this phase.", "کمی زودتر بخوابید؛ در این فاز انرژی کم\u200cکم افت می\u200cکند."),
			tip("mental_health", "A good time to finish what you started rather than begin new things.", "زمان خوبی برای تمام\u200cکردن کارهای نیمه\u200cتمام است تا شروع کارهای جدید."),
		}
	}
	return []recommendation.Tip{}
}

// symptomTips is getSymptomBasedTips (HealthDataEngine.php:902).
func symptomTips(log *DailyLog) []recommendation.Tip {
	var tips []recommendation.Tip
	if log.HeadacheIntensity != nil {
		tips = append(tips, tip("pain_relief", "For headaches: rest in a dark room and stay hydrated.", "برای سردرد: در اتاق تاریک استراحت کنید و آب بنوشید."))
	}
	if log.PelvicPainIntensity != nil || log.StomachAcheIntensity != nil {
		tips = append(tips, tip("pain_relief", "For cramps: try a warm bath or heating pad.", "برای کرامپ: حمام گرم یا پد گرم\u200cکننده امتحان کنید."))
	}
	if eq(log.SleepQuality, "bad") {
		tips = append(tips, tip("sleep", "Try to maintain a regular sleep schedule and avoid screens before bed.", "سعی کنید برنامه خواب منظم داشته باشید و قبل از خواب از صفحه نمایش استفاده نکنید."))
	}
	if slices.Contains(log.Moods, "anxious") || slices.Contains(log.Moods, "sad") {
		tips = append(tips, tip("mental_health", "Take time for activities you enjoy. Consider light exercise or meditation.", "وقتی برای فعالیت\u200cهای مورد علاقه\u200cتان بگذارید. ورزش سبک یا مدیتیشن را در نظر بگیرید."))
	}
	if log.BloatingIntensity != nil {
		tips = append(tips, tip("digestion", "Reduce salt intake and eat smaller, frequent meals.", "مصرف نمک را کم کنید و وعده\u200cهای کوچک و مکرر بخورید."))
	}
	if isTrue(log.Fatigue) {
		tips = append(tips, tip("energy", "Listen to your body. Take short breaks and prioritize rest.", "به بدنتان گوش دهید. استراحت\u200cهای کوتاه داشته باشید و استراحت را اولویت قرار دهید."))
	}
	return tips
}
