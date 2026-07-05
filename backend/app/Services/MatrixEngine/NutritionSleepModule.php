<?php

namespace App\Services\MatrixEngine;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Models\DailyHealthLog;
use App\Models\User;
use App\Models\UserProfile;

/**
 * Nutrition & Sleep Module (Life-Sync)
 *
 * Uses sleep data and nutrition input to optimize cycle-based recommendations.
 * Provides phase-specific nutrition and exercise tips.
 */
class NutritionSleepModule
{
    private User $user;
    private ?UserProfile $profile;
    private string $locale;

    public function __construct(User $user, string $locale = 'fa')
    {
        $this->user = $user;
        $this->profile = $user->profile;
        $this->locale = $locale;
    }

    /**
     * Get nutrition and sleep tips based on phase and data
     */
    public function getTips(
        CyclePhase $phase,
        CycleSubphase $subphase,
        ?DailyHealthLog $dailyLog,
        bool $isTTC = false
    ): array {
        $isPremium = $this->profile?->isPremium() ?? false;

        $tips = [
            'nutrition' => $this->getNutritionTips($phase, $subphase, $isPremium, $isTTC),
            'sleep' => $this->getSleepTips($phase, $dailyLog, $isPremium, $isTTC),
            'exercise' => $this->getExerciseTips($phase, $isPremium, $isTTC),
        ];

        // Add smart sleep insights if we have sleep data
        if ($dailyLog) {
            $sleepInsights = $this->analyzeSleepData($phase, $dailyLog, $isTTC);
            if ($sleepInsights) {
                $tips['sleep_insights'] = $sleepInsights;
            }
        }

        return $tips;
    }

    /**
     * Get phase-specific nutrition tips
     */
    private function getNutritionTips(CyclePhase $phase, CycleSubphase $subphase, bool $isPremium, bool $isTTC): array
    {
        $locale = $this->locale;

        return match ($phase) {
            CyclePhase::MENSTRUATION => $this->getMenstruationNutrition($isPremium, $isTTC),
            CyclePhase::FOLLICULAR => $this->getFollicularNutrition($isPremium, $isTTC),
            CyclePhase::OVULATION => $this->getOvulationNutrition($isPremium, $isTTC),
            CyclePhase::LUTEAL => $this->getLutealNutrition($subphase, $isPremium, $isTTC),
        };
    }

    /**
     * Menstruation phase nutrition
     */
    private function getMenstruationNutrition(bool $isPremium, bool $isTTC): array
    {
        $locale = $this->locale;

        $tips = [
            'focus' => $locale === 'fa' ? 'آهن و انرژی' : 'Iron and Energy',
            'main_tip' => $locale === 'fa'
                ? 'خونریزی یعنی دفع آهن. برای جلوگیری از سیاهی رفتن چشم‌هات، پسته، جگر یا قرص آهنت رو فراموش نکن.'
                : 'Bleeding means iron loss. To prevent feeling faint, don\'t forget pistachios, liver, or your iron supplement.',
            'foods_to_eat' => $locale === 'fa'
                ? ['گوشت قرمز', 'جگر', 'اسفناج', 'عدس', 'پسته']
                : ['Red meat', 'Liver', 'Spinach', 'Lentils', 'Pistachios'],
            'foods_to_avoid' => $locale === 'fa'
                ? ['کافئین زیاد', 'غذاهای سرد']
                : ['Too much caffeine', 'Cold foods'],
        ];

        if ($isTTC && $isPremium) {
            $tips['ttc_focus'] = $locale === 'fa'
                ? 'شروع مکمل فولیک اسید، غذای خون‌ساز بخور تا ذخایر آهن برای سیکل بعد پر بشه.'
                : 'Start folic acid supplement, eat blood-building food to replenish iron reserves for next cycle.';
        }

        return $tips;
    }

    /**
     * Follicular phase nutrition
     */
    private function getFollicularNutrition(bool $isPremium, bool $isTTC): array
    {
        $locale = $this->locale;

        $tips = [
            'focus' => $locale === 'fa' ? 'کیفیت تخمک' : 'Egg Quality',
            'main_tip' => $locale === 'fa'
                ? 'این هفته آجیل خام، آووکادو و روغن زیتون بیشتر بخور. چربی‌های سالم سوخت اصلی برای رشد یک تخمک باکیفیت هستن.'
                : 'This week eat more raw nuts, avocado and olive oil. Healthy fats are the main fuel for growing a quality egg.',
            'foods_to_eat' => $locale === 'fa'
                ? ['آجیل خام', 'آووکادو', 'روغن زیتون', 'سبزیجات برگ سبز', 'پروتئین']
                : ['Raw nuts', 'Avocado', 'Olive oil', 'Leafy greens', 'Protein'],
            'foods_to_avoid' => $locale === 'fa'
                ? ['شکر و شیرینی‌جات', 'غذاهای فرآوری شده']
                : ['Sugar and sweets', 'Processed foods'],
        ];

        if ($isTTC && $isPremium) {
            $tips['ttc_focus'] = $locale === 'fa'
                ? 'مصرف آنتی‌اکسیدان (آجیل، سبزیجات) برای محافظت از DNA تخمک.'
                : 'Consume antioxidants (nuts, vegetables) to protect egg DNA.';
        }

        return $tips;
    }

    /**
     * Ovulation phase nutrition
     */
    private function getOvulationNutrition(bool $isPremium, bool $isTTC): array
    {
        $locale = $this->locale;

        $tips = [
            'focus' => $locale === 'fa' ? 'انرژی و آب' : 'Energy and Hydration',
            'main_tip' => $locale === 'fa'
                ? 'برای اینکه ترشحات دهانه رحم کیفیت بهتری داشته باشه و به حرکت اسپرم کمک کنه، نوشیدن آب رو دو برابر کن.'
                : 'To improve cervical mucus quality and help sperm movement, double your water intake.',
            'foods_to_eat' => $locale === 'fa'
                ? ['آب زیاد', 'میوه‌های آبدار', 'سبزیجات تازه']
                : ['Lots of water', 'Juicy fruits', 'Fresh vegetables'],
            'foods_to_avoid' => $locale === 'fa'
                ? ['الکل', 'کافئین زیاد']
                : ['Alcohol', 'Too much caffeine'],
        ];

        if ($isTTC && $isPremium) {
            $tips['ttc_focus'] = $locale === 'fa'
                ? 'آب‌رسانی برای بهبود کیفیت ترشحات و کمک به حرکت اسپرم.'
                : 'Hydration to improve discharge quality and help sperm movement.';
        }

        return $tips;
    }

    /**
     * Luteal phase nutrition
     */
    private function getLutealNutrition(CycleSubphase $subphase, bool $isPremium, bool $isTTC): array
    {
        $locale = $this->locale;

        $tips = [
            'focus' => $locale === 'fa' ? 'ضد التهاب و ضد نفخ' : 'Anti-inflammatory and Anti-bloat',
            'main_tip' => $locale === 'fa'
                ? 'احساس ورم داری؟ نمک رو حذف کن و موز (پتاسیم) بخور. این ورم چربی نیست، آبه.'
                : 'Feeling bloated? Cut salt and eat bananas (potassium). This bloating isn\'t fat, it\'s water.',
            'foods_to_eat' => $locale === 'fa'
                ? ['موز', 'سبزیجات پخته', 'ماهی', 'غذاهای کم نمک']
                : ['Bananas', 'Cooked vegetables', 'Fish', 'Low-salt foods'],
            'foods_to_avoid' => $locale === 'fa'
                ? ['نمک زیاد', 'غذاهای شور', 'کافئین']
                : ['Too much salt', 'Salty foods', 'Caffeine'],
        ];

        if ($isTTC && $isPremium) {
            $tips['ttc_focus'] = $locale === 'fa'
                ? 'آناناس (به‌خاطر بروملائین) و غذاهای گرم مثل سوپ رو دریاب. التهاب کم = لانه‌گزینی راحت‌تر.'
                : 'Try pineapple (for bromelain) and warm foods like soup. Less inflammation = easier implantation.';
            $tips['foods_to_eat'][] = $locale === 'fa' ? 'آناناس' : 'Pineapple';
            $tips['foods_to_eat'][] = $locale === 'fa' ? 'سوپ گرم' : 'Warm soup';
        }

        return $tips;
    }

    /**
     * Get phase-specific sleep tips
     */
    private function getSleepTips(CyclePhase $phase, ?DailyHealthLog $dailyLog, bool $isPremium, bool $isTTC): array
    {
        $locale = $this->locale;

        $baseTips = match ($phase) {
            CyclePhase::MENSTRUATION => [
                'recommendation' => $locale === 'fa'
                    ? 'در این فاز بدن به استراحت بیشتری نیاز داره. سعی کن زودتر بخوابی.'
                    : 'In this phase, the body needs more rest. Try to sleep earlier.',
                'hours' => '8-9',
            ],
            CyclePhase::FOLLICULAR => [
                'recommendation' => $locale === 'fa'
                    ? 'انرژی بالاست ولی خواب کافی برای رشد فولیکول‌ها ضروریه.'
                    : 'Energy is high but enough sleep is essential for follicle growth.',
                'hours' => '7-8',
            ],
            CyclePhase::OVULATION => [
                'recommendation' => $locale === 'fa'
                    ? 'خواب در این دو سه روز سپر محافظتی تخمک‌هاست.'
                    : 'Sleep in these 2-3 days is a protective shield for eggs.',
                'hours' => '7-8',
            ],
            CyclePhase::LUTEAL => [
                'recommendation' => $locale === 'fa'
                    ? 'بدنت داغ‌تر شده (افزایش دمای پایه). اتاق رو خنک‌تر کن و پتو نازک‌تر استفاده کن.'
                    : 'Your body is warmer (increased basal temperature). Keep the room cooler and use a lighter blanket.',
                'hours' => '8-9',
            ],
        };

        if ($isTTC && $isPremium && $phase === CyclePhase::FOLLICULAR) {
            $baseTips['ttc_tip'] = $locale === 'fa'
                ? 'رشد فولیکول‌ها شب‌ها و در خواب عمیق اتفاق میفته. کم‌خوابی سرعت رشدشون رو کم می‌کنه.'
                : 'Follicle growth happens at night during deep sleep. Sleep deprivation slows their growth.';
        }

        if ($isTTC && $isPremium && $phase === CyclePhase::LUTEAL) {
            $baseTips['ttc_tip'] = $locale === 'fa'
                ? 'بی‌نظمی خواب می‌تونه دمای پایه بدن (BBT) رو تغییر بده. ساعت خوابت رو ثابت نگه دار.'
                : 'Irregular sleep can change basal body temperature (BBT). Keep your sleep schedule consistent.';
        }

        return $baseTips;
    }

    /**
     * Get phase-specific exercise tips
     */
    private function getExerciseTips(CyclePhase $phase, bool $isPremium, bool $isTTC): array
    {
        $locale = $this->locale;

        return match ($phase) {
            CyclePhase::MENSTRUATION => [
                'intensity' => $locale === 'fa' ? 'سبک' : 'Light',
                'recommendation' => $locale === 'fa'
                    ? 'ورزش سبک مثل یوگا یا پیاده‌روی آهسته. از ورزش سنگین اجتناب کن.'
                    : 'Light exercise like yoga or slow walking. Avoid heavy exercise.',
                'suggested' => $locale === 'fa'
                    ? ['یوگا', 'پیاده‌روی آهسته', 'کشش']
                    : ['Yoga', 'Slow walking', 'Stretching'],
                'avoid' => $locale === 'fa'
                    ? ['ورزش سنگین', 'وزنه‌برداری سنگین']
                    : ['Heavy exercise', 'Heavy weightlifting'],
            ],
            CyclePhase::FOLLICULAR => [
                'intensity' => $locale === 'fa' ? 'متوسط تا بالا' : 'Medium to High',
                'recommendation' => $locale === 'fa'
                    ? 'بدنت الان آماده‌ی چربی‌سوزیه. بهترین زمان برای تمرینات اینتروال (HIIT) همین هفته‌ست.'
                    : 'Your body is ready for fat burning now. Best time for HIIT workouts is this week.',
                'suggested' => $locale === 'fa'
                    ? ['HIIT', 'دویدن', 'ایروبیک', 'وزنه‌برداری']
                    : ['HIIT', 'Running', 'Aerobics', 'Weightlifting'],
                'avoid' => $locale === 'fa'
                    ? []
                    : [],
            ],
            CyclePhase::OVULATION => [
                'intensity' => $locale === 'fa' ? 'بالا' : 'High',
                'recommendation' => $locale === 'fa'
                    ? 'اوج انرژی! بهترین زمان برای رکوردشکنی و تمرینات سخت.'
                    : 'Peak energy! Best time for breaking records and hard workouts.',
                'suggested' => $locale === 'fa'
                    ? ['تمرینات قدرتی', 'کلاس‌های گروهی', 'ورزش‌های رقابتی']
                    : ['Strength training', 'Group classes', 'Competitive sports'],
                'avoid' => $locale === 'fa'
                    ? []
                    : [],
            ],
            CyclePhase::LUTEAL => [
                'intensity' => $locale === 'fa' ? 'متوسط تا سبک' : 'Medium to Light',
                'recommendation' => $locale === 'fa'
                    ? 'انرژی کم میشه. ورزش‌های ملایم‌تر مثل پیلاتس یا شنا توصیه میشه.'
                    : 'Energy decreases. Gentler exercises like Pilates or swimming are recommended.',
                'suggested' => $locale === 'fa'
                    ? ['پیلاتس', 'شنا', 'پیاده‌روی', 'یوگا ریلکس']
                    : ['Pilates', 'Swimming', 'Walking', 'Relaxing yoga'],
                'avoid' => $locale === 'fa'
                    ? ['تمرینات خیلی سنگین در هفته آخر']
                    : ['Very heavy workouts in the last week'],
            ],
        };
    }

    /**
     * Analyze sleep data and provide insights
     */
    private function analyzeSleepData(CyclePhase $phase, DailyHealthLog $dailyLog, bool $isTTC): ?array
    {
        $locale = $this->locale;

        if (!$dailyLog->sleep_quality && !$dailyLog->sleep_duration) {
            return null;
        }

        $insights = [];

        // Poor sleep in follicular phase (affects egg growth)
        if ($phase === CyclePhase::FOLLICULAR && $dailyLog->sleep_quality === 'bad') {
            $insights[] = [
                'type' => 'follicular_sleep',
                'message' => $isTTC
                    ? ($locale === 'fa'
                        ? 'رشد فولیکول‌ها شب‌ها و در خواب عمیق اتفاق میفته. کم‌خوابی دیشب ممکنه سرعت رشدشون رو کم کنه، امروز سعی کن جبران کنی.'
                        : 'Follicle growth happens at night during deep sleep. Last night\'s poor sleep might slow their growth, try to make up for it today.')
                    : ($locale === 'fa'
                        ? 'کم‌خوابی می‌تونه انرژی این فاز رو کم کنه، سعی کن امشب زودتر بخوابی.'
                        : 'Poor sleep can reduce the energy of this phase, try to sleep earlier tonight.'),
            ];
        }

        // Poor sleep near ovulation (affects egg quality)
        if ($phase === CyclePhase::OVULATION && $dailyLog->sleep_quality === 'bad') {
            $insights[] = [
                'type' => 'ovulation_sleep',
                'message' => $isTTC
                    ? ($locale === 'fa'
                        ? 'تخمک‌گذاری نزدیکه اما خوابت کیفیت نداره. امشب گوشی رو یک ساعت زودتر کنار بذار.'
                        : 'Ovulation is near but your sleep quality is poor. Put your phone away an hour earlier tonight.')
                    : ($locale === 'fa'
                        ? 'خواب کافی برای حفظ انرژی این روزها مهمه.'
                        : 'Enough sleep is important to maintain energy these days.'),
            ];
        }

        // Poor sleep in luteal phase (affects BBT tracking)
        if ($phase === CyclePhase::LUTEAL && $dailyLog->sleep_quality === 'bad') {
            $insights[] = [
                'type' => 'luteal_sleep',
                'message' => $isTTC
                    ? ($locale === 'fa'
                        ? 'بی‌نظمی خواب می‌تونه دمای پایه بدن رو تغییر بده و تفسیر علائم رو سخت کنه.'
                        : 'Irregular sleep can change basal body temperature and make interpreting symptoms difficult.')
                    : ($locale === 'fa'
                        ? 'در فاز لوتئال بدن گرم‌تره، اتاق رو خنک نگه دار.'
                        : 'In the luteal phase the body is warmer, keep the room cool.'),
            ];
        }

        return !empty($insights) ? $insights : null;
    }
}
