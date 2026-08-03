<?php

namespace Database\Seeders;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Enums\RecommendationTrigger;
use App\Enums\RecommendationType;
use App\Models\Recommendation;
use Illuminate\Database\Seeder;

/**
 * Seeds the "توصیه‌های امروز" recommendations with the copy that used to live in
 * HealthDataEngine, so an existing install keeps showing exactly the same advice
 * the day the admin page goes live — and admins have real rows to edit.
 *
 * Idempotent and non-destructive: rows are matched on their stable `key` and
 * only ever inserted. Re-running adds recommendations shipped since the last
 * deploy and never overwrites text an admin has since edited (or re-enables a
 * row they deliberately switched off).
 */
class RecommendationSeeder extends Seeder
{
    /** The two luteal sub-phases the engine treated as "PMS may be starting". */
    private const PMS_SUBPHASES = [
        CycleSubphase::LATE_LUTEAL->value,
        CycleSubphase::PMS_POSSIBLE->value,
    ];

    /** The luteal sub-phases before that window. */
    private const EARLY_LUTEAL_SUBPHASES = [
        CycleSubphase::EARLY_LUTEAL->value,
        CycleSubphase::MID_LUTEAL->value,
    ];

    public function run(): void
    {
        foreach ($this->rows() as $index => $row) {
            Recommendation::firstOrCreate(
                ['key' => $row['key']],
                [
                    'type' => $row['type'],
                    'text' => $row['text'],
                    'cycle_phase' => $row['cycle_phase'] ?? null,
                    'cycle_subphases' => $row['cycle_subphases'] ?? null,
                    'symptom_trigger' => $row['symptom_trigger'] ?? null,
                    // Leave gaps so an admin can slot a row in without
                    // renumbering. Derived from the position below, so a row
                    // added mid-list later can land on a value an existing row
                    // already holds — harmless, ties fall back to `id`.
                    'sort_order' => ($index + 1) * 10,
                    'is_active' => true,
                ],
            );
        }
    }

    /**
     * @return array<int, array<string, mixed>>
     */
    private function rows(): array
    {
        return [
            // ── Menstruation ───────────────────────────────────────
            [
                'key' => 'menstruation.hydration',
                'type' => RecommendationType::HYDRATION->value,
                'cycle_phase' => CyclePhase::MENSTRUATION->value,
                'text' => [
                    'fa' => 'آب کافی بنوشید و هیدراته بمانید.',
                    'en' => 'Stay hydrated and drink plenty of water.',
                ],
            ],
            [
                'key' => 'menstruation.warmth',
                'type' => RecommendationType::WARMTH->value,
                'cycle_phase' => CyclePhase::MENSTRUATION->value,
                'text' => [
                    'fa' => 'کمپرس گرم روی شکم قرار دهید تا دردها کاهش یابد.',
                    'en' => 'Apply a warm compress to your lower abdomen to relieve cramps.',
                ],
            ],
            [
                'key' => 'menstruation.rest',
                'type' => RecommendationType::REST->value,
                'cycle_phase' => CyclePhase::MENSTRUATION->value,
                'text' => [
                    'fa' => 'استراحت کافی داشته باشید و از فعالیت‌های سنگین اجتناب کنید.',
                    'en' => 'Get enough rest and avoid strenuous activities.',
                ],
            ],
            [
                'key' => 'menstruation.nutrition',
                'type' => RecommendationType::NUTRITION->value,
                'cycle_phase' => CyclePhase::MENSTRUATION->value,
                'text' => [
                    'fa' => 'غذاهای غنی از آهن مثل اسفناج و گوشت قرمز بخورید.',
                    'en' => 'Eat iron-rich foods like spinach and red meat.',
                ],
            ],

            // ── Follicular ─────────────────────────────────────────
            [
                'key' => 'follicular.energy',
                'type' => RecommendationType::ENERGY->value,
                'cycle_phase' => CyclePhase::FOLLICULAR->value,
                'text' => [
                    'fa' => 'سطح انرژی شما در حال افزایش است. زمان مناسبی برای پروژه‌های جدید!',
                    'en' => 'Your energy levels are rising. Great time for new projects!',
                ],
            ],
            [
                'key' => 'follicular.exercise',
                'type' => RecommendationType::EXERCISE->value,
                'cycle_phase' => CyclePhase::FOLLICULAR->value,
                'text' => [
                    'fa' => 'زمان مناسبی برای ورزش‌های سنگین است.',
                    'en' => 'Perfect time for high-intensity workouts.',
                ],
            ],
            [
                'key' => 'follicular.nutrition',
                'type' => RecommendationType::NUTRITION->value,
                'cycle_phase' => CyclePhase::FOLLICULAR->value,
                'text' => [
                    'fa' => 'پروتئین و سبزیجات تازه بخورید تا ذخیره انرژی‌تان دوباره پر شود.',
                    'en' => 'Add protein and fresh vegetables to rebuild your energy stores.',
                ],
            ],
            [
                'key' => 'follicular.mental_health',
                'type' => RecommendationType::MENTAL_HEALTH->value,
                'cycle_phase' => CyclePhase::FOLLICULAR->value,
                'text' => [
                    'fa' => 'فرصت خوبی برای برنامه‌ریزی و یادگیری چیزهای تازه است.',
                    'en' => 'A good window for planning and learning something new.',
                ],
            ],

            // ── Ovulation ──────────────────────────────────────────
            [
                'key' => 'ovulation.fertility',
                'type' => RecommendationType::FERTILITY->value,
                'cycle_phase' => CyclePhase::OVULATION->value,
                'text' => [
                    'fa' => 'روزهای اوج باروری. اگر قصد بارداری دارید، بهترین زمان است.',
                    'en' => 'Peak fertility days. If trying to conceive, this is the best time.',
                ],
            ],
            [
                'key' => 'ovulation.energy',
                'type' => RecommendationType::ENERGY->value,
                'cycle_phase' => CyclePhase::OVULATION->value,
                'text' => [
                    'fa' => 'ممکن است احساس اعتماد به نفس و اجتماعی بودن بیشتری داشته باشید.',
                    'en' => 'You may feel more confident and social.',
                ],
            ],
            [
                'key' => 'ovulation.hydration',
                'type' => RecommendationType::HYDRATION->value,
                'cycle_phase' => CyclePhase::OVULATION->value,
                'text' => [
                    'fa' => 'مرتب آب بنوشید؛ هیدراته بودن به کیفیت ترشحات کمک می‌کند.',
                    'en' => 'Drink water regularly — hydration supports cervical fluid.',
                ],
            ],
            [
                'key' => 'ovulation.sleep',
                'type' => RecommendationType::SLEEP->value,
                'cycle_phase' => CyclePhase::OVULATION->value,
                'text' => [
                    'fa' => 'برنامه خواب منظمی داشته باشید تا تعادل هورمونی حفظ شود.',
                    'en' => 'Keep a steady sleep schedule to support hormone balance.',
                ],
            ],

            // ── Luteal, before the PMS window ──────────────────────
            [
                'key' => 'luteal.nutrition',
                'type' => RecommendationType::NUTRITION->value,
                'cycle_phase' => CyclePhase::LUTEAL->value,
                'cycle_subphases' => self::EARLY_LUTEAL_SUBPHASES,
                'text' => [
                    'fa' => 'روی غذاهای غنی از منیزیم و ویتامین B6 تمرکز کنید.',
                    'en' => 'Focus on foods rich in magnesium and vitamin B6.',
                ],
            ],
            [
                'key' => 'luteal.exercise',
                'type' => RecommendationType::EXERCISE->value,
                'cycle_phase' => CyclePhase::LUTEAL->value,
                'cycle_subphases' => self::EARLY_LUTEAL_SUBPHASES,
                'text' => [
                    'fa' => 'به حرکات ملایم‌تر مثل یوگا، پیلاتس یا پیاده‌روی رو بیاورید.',
                    'en' => 'Switch to gentler movement like yoga, pilates or walking.',
                ],
            ],
            [
                'key' => 'luteal.sleep',
                'type' => RecommendationType::SLEEP->value,
                'cycle_phase' => CyclePhase::LUTEAL->value,
                'cycle_subphases' => self::EARLY_LUTEAL_SUBPHASES,
                'text' => [
                    'fa' => 'کمی زودتر بخوابید؛ در این فاز انرژی کم‌کم افت می‌کند.',
                    'en' => 'Go to bed a little earlier — energy dips in this phase.',
                ],
            ],
            [
                'key' => 'luteal.mental_health',
                'type' => RecommendationType::MENTAL_HEALTH->value,
                'cycle_phase' => CyclePhase::LUTEAL->value,
                'cycle_subphases' => self::EARLY_LUTEAL_SUBPHASES,
                'text' => [
                    'fa' => 'زمان خوبی برای تمام‌کردن کارهای نیمه‌تمام است تا شروع کارهای جدید.',
                    'en' => 'A good time to finish what you started rather than begin new things.',
                ],
            ],

            // ── Late luteal / possible PMS ─────────────────────────
            [
                'key' => 'luteal.pms',
                'type' => RecommendationType::PMS->value,
                'cycle_phase' => CyclePhase::LUTEAL->value,
                'cycle_subphases' => self::PMS_SUBPHASES,
                'text' => [
                    'fa' => 'علائم PMS ممکن است ظاهر شوند. مراقبت از خود و آرامش را تمرین کنید.',
                    'en' => 'PMS symptoms may appear. Practice self-care and relaxation.',
                ],
            ],
            [
                'key' => 'luteal.pms_nutrition',
                'type' => RecommendationType::NUTRITION->value,
                'cycle_phase' => CyclePhase::LUTEAL->value,
                'cycle_subphases' => self::PMS_SUBPHASES,
                'text' => [
                    'fa' => 'مصرف نمک و کافئین را کاهش دهید تا نفخ کم شود.',
                    'en' => 'Reduce salt and caffeine intake to minimize bloating.',
                ],
            ],
            [
                'key' => 'luteal.pms_mood',
                'type' => RecommendationType::MOOD->value,
                'cycle_phase' => CyclePhase::LUTEAL->value,
                'cycle_subphases' => self::PMS_SUBPHASES,
                'text' => [
                    'fa' => 'استراحت کنید و اگر احساس تحریک‌پذیری دارید تنفس عمیق تمرین کنید.',
                    'en' => 'Take breaks and practice deep breathing if feeling irritable.',
                ],
            ],

            // ── Symptom-triggered (any phase) ──────────────────────
            [
                'key' => 'symptom.headache',
                'type' => RecommendationType::PAIN_RELIEF->value,
                'symptom_trigger' => RecommendationTrigger::HEADACHE->value,
                'text' => [
                    'fa' => 'برای سردرد: در اتاق تاریک استراحت کنید و آب بنوشید.',
                    'en' => 'For headaches: rest in a dark room and stay hydrated.',
                ],
            ],
            [
                'key' => 'symptom.cramps',
                'type' => RecommendationType::PAIN_RELIEF->value,
                'symptom_trigger' => RecommendationTrigger::CRAMPS->value,
                'text' => [
                    'fa' => 'برای کرامپ: حمام گرم یا پد گرم‌کننده امتحان کنید.',
                    'en' => 'For cramps: try a warm bath or heating pad.',
                ],
            ],
            [
                'key' => 'symptom.poor_sleep',
                'type' => RecommendationType::SLEEP->value,
                'symptom_trigger' => RecommendationTrigger::POOR_SLEEP->value,
                'text' => [
                    'fa' => 'سعی کنید برنامه خواب منظم داشته باشید و قبل از خواب از صفحه نمایش استفاده نکنید.',
                    'en' => 'Try to maintain a regular sleep schedule and avoid screens before bed.',
                ],
            ],
            [
                'key' => 'symptom.low_mood',
                'type' => RecommendationType::MENTAL_HEALTH->value,
                'symptom_trigger' => RecommendationTrigger::LOW_MOOD->value,
                'text' => [
                    'fa' => 'وقتی برای فعالیت‌های مورد علاقه‌تان بگذارید. ورزش سبک یا مدیتیشن را در نظر بگیرید.',
                    'en' => 'Take time for activities you enjoy. Consider light exercise or meditation.',
                ],
            ],
            [
                'key' => 'symptom.bloating',
                'type' => RecommendationType::DIGESTION->value,
                'symptom_trigger' => RecommendationTrigger::BLOATING->value,
                'text' => [
                    'fa' => 'مصرف نمک را کم کنید و وعده‌های کوچک و مکرر بخورید.',
                    'en' => 'Reduce salt intake and eat smaller, frequent meals.',
                ],
            ],
            [
                'key' => 'symptom.fatigue',
                'type' => RecommendationType::ENERGY->value,
                'symptom_trigger' => RecommendationTrigger::FATIGUE->value,
                'text' => [
                    'fa' => 'به بدنتان گوش دهید. استراحت‌های کوتاه داشته باشید و استراحت را اولویت قرار دهید.',
                    'en' => 'Listen to your body. Take short breaks and prioritize rest.',
                ],
            ],
        ];
    }
}
