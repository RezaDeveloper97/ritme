<?php

namespace App\Services\MessageSystem\Layers;

use App\Models\DailyHealthLog;
use App\Models\User;
use App\Services\MessageSystem\Contracts\ProvidesMessageContent;
use App\Services\MessageSystem\Core\MessageContext;
use App\Services\MessageSystem\Support\MessageContentRepository;

/**
 * Layer 4: Pattern Recognition Engine (Premium Only)
 * Detects long-term patterns across multiple cycles
 */
class PatternLayer implements ProvidesMessageContent
{
    public function __construct(
        private readonly User $user,
        private readonly string $locale = 'fa',
    ) {}

    private function content(): MessageContentRepository
    {
        return app(MessageContentRepository::class);
    }

    /**
     * Analyze patterns based on historical data
     */
    public function analyze(MessageContext $context): array
    {
        $patterns = [];

        // Get historical data
        $recentLogs = $context->recentLogs ?? [];

        if (count($recentLogs) < 14) {
            // Not enough data for pattern detection
            $patterns[] = $this->pattern('insufficient_data', 'info');
            return $patterns;
        }

        // Analyze based on mode
        if ($context->isCycleMode()) {
            $patterns = array_merge($patterns, $this->analyzeCyclePatterns($context, $recentLogs));
        } elseif ($context->isPregnancyMode()) {
            $patterns = array_merge($patterns, $this->analyzePregnancyPatterns($context, $recentLogs));
        }

        // Common patterns
        $patterns = array_merge($patterns, $this->analyzeCommonPatterns($context, $recentLogs));

        return $patterns;
    }

    /**
     * Analyze cycle-specific patterns
     */
    private function analyzeCyclePatterns(MessageContext $context, array $logs): array
    {
        $patterns = [];

        // Analyze PMS pattern
        $pmsPattern = $this->detectPMSPattern($logs);
        if ($pmsPattern) {
            $patterns[] = $pmsPattern;
        }

        // Analyze pain pattern
        $painPattern = $this->detectPainPattern($logs);
        if ($painPattern) {
            $patterns[] = $painPattern;
        }

        // Analyze mood pattern
        $moodPattern = $this->detectMoodPattern($logs);
        if ($moodPattern) {
            $patterns[] = $moodPattern;
        }

        // Cycle length variability
        $cyclePattern = $this->detectCycleLengthPattern($logs);
        if ($cyclePattern) {
            $patterns[] = $cyclePattern;
        }

        // TTC-specific patterns
        if ($context->isTTC()) {
            $ttcPattern = $this->detectTTCPattern($logs);
            if ($ttcPattern) {
                $patterns[] = $ttcPattern;
            }
        }

        return $patterns;
    }

    /**
     * Analyze pregnancy-specific patterns
     */
    private function analyzePregnancyPatterns(MessageContext $context, array $logs): array
    {
        $patterns = [];

        // Symptom progression pattern
        $symptomProgression = $this->detectSymptomProgression($logs, $context->trimester ?? 1);
        if ($symptomProgression) {
            $patterns[] = $symptomProgression;
        }

        return $patterns;
    }

    /**
     * Analyze common patterns (all modes)
     */
    private function analyzeCommonPatterns(MessageContext $context, array $logs): array
    {
        $patterns = [];

        // Sleep quality pattern
        $sleepPattern = $this->detectSleepPattern($logs);
        if ($sleepPattern) {
            $patterns[] = $sleepPattern;
        }

        // Energy level pattern
        $energyPattern = $this->detectEnergyPattern($logs);
        if ($energyPattern) {
            $patterns[] = $energyPattern;
        }

        return $patterns;
    }

    /**
     * Detect PMS pattern
     */
    private function detectPMSPattern(array $logs): ?array
    {
        // Count PMS-related symptoms in last 7 days before menstruation
        $pmsSymptoms = ['bloating', 'mood_sad', 'mood_angry', 'headache', 'breast_tenderness'];
        $pmsOccurrences = 0;

        foreach ($logs as $log) {
            foreach ($pmsSymptoms as $symptom) {
                $field = 'has_' . str_replace('mood_', '', $symptom);
                if (isset($log[$field]) && $log[$field]) {
                    $pmsOccurrences++;
                }
            }
        }

        if ($pmsOccurrences > 10) {
            return $this->pattern('recurring_pms', 'info');
        }

        return null;
    }

    /**
     * Detect pain pattern
     */
    private function detectPainPattern(array $logs): ?array
    {
        $painDays = 0;
        $severePainDays = 0;

        foreach ($logs as $log) {
            if (isset($log['has_cramps']) && $log['has_cramps']) {
                $painDays++;
                if (isset($log['cramp_severity']) && $log['cramp_severity'] === 'severe') {
                    $severePainDays++;
                }
            }
        }

        if ($severePainDays >= 3) {
            return $this->pattern('severe_pain', 'warning');
        }

        if ($painDays >= 7) {
            return $this->pattern('frequent_pain', 'info');
        }

        return null;
    }

    /**
     * Detect mood pattern
     */
    private function detectMoodPattern(array $logs): ?array
    {
        $sadDays = 0;
        $anxiousDays = 0;

        foreach ($logs as $log) {
            $mood = $log['mood'] ?? null;
            if (in_array($mood, ['sad', 'depressed'])) {
                $sadDays++;
            }
            if (in_array($mood, ['anxious', 'stressed'])) {
                $anxiousDays++;
            }
        }

        if ($sadDays >= 10) {
            return $this->pattern('low_mood', 'warning');
        }

        if ($anxiousDays >= 7) {
            return $this->pattern('anxiety_pattern', 'info');
        }

        return null;
    }

    /**
     * Detect cycle length pattern
     */
    private function detectCycleLengthPattern(array $logs): ?array
    {
        // This would need period start dates to calculate
        // For now, we'll skip this as it requires more complex analysis
        return null;
    }

    /**
     * Detect TTC-specific patterns
     */
    private function detectTTCPattern(array $logs): ?array
    {
        // Check for consistent timing attempts during fertile window
        // This is a placeholder for more complex TTC analysis
        return $this->pattern('ttc_tracking', 'info');
    }

    /**
     * Detect sleep pattern
     */
    private function detectSleepPattern(array $logs): ?array
    {
        $poorSleepDays = 0;

        foreach ($logs as $log) {
            $quality = $log['sleep_quality'] ?? null;
            if (in_array($quality, ['poor', 'very_poor'])) {
                $poorSleepDays++;
            }
        }

        if ($poorSleepDays >= 10) {
            return $this->pattern('poor_sleep', 'warning');
        }

        return null;
    }

    /**
     * Detect energy pattern
     */
    private function detectEnergyPattern(array $logs): ?array
    {
        $lowEnergyDays = 0;

        foreach ($logs as $log) {
            $energy = $log['energy_level'] ?? null;
            if (in_array($energy, ['low', 'very_low'])) {
                $lowEnergyDays++;
            }
        }

        if ($lowEnergyDays >= 15) {
            return $this->pattern('chronic_fatigue', 'warning');
        }

        return null;
    }

    /**
     * Detect symptom progression in pregnancy
     */
    private function detectSymptomProgression(array $logs, int $trimester): ?array
    {
        // Compare symptom frequency over time
        if (count($logs) < 14) {
            return null;
        }

        $recentLogs = array_slice($logs, 0, 7);
        $olderLogs = array_slice($logs, 7, 7);

        $recentNausea = 0;
        $olderNausea = 0;

        foreach ($recentLogs as $log) {
            if (isset($log['has_nausea']) && $log['has_nausea']) {
                $recentNausea++;
            }
        }

        foreach ($olderLogs as $log) {
            if (isset($log['has_nausea']) && $log['has_nausea']) {
                $olderNausea++;
            }
        }

        if ($trimester === 1 && $recentNausea > $olderNausea + 2) {
            return $this->pattern('increasing_nausea', 'info');
        }

        if ($trimester === 2 && $recentNausea < $olderNausea - 2) {
            return $this->pattern('improving_symptoms', 'info');
        }

        return null;
    }

    /**
     * Build a single pattern entry. Detection thresholds, the stable
     * pattern_type and alert_level stay in code; only message and recommendation
     * are editable content (DB-editable, falls back to the seeded literal).
     */
    private function pattern(string $patternType, string $alertLevel): array
    {
        $entry = self::patternContent()[$patternType];
        $fallback = self::slice($entry, $this->locale);
        $p = $this->content()->resolve('pattern', $patternType, $this->locale, $fallback);

        return [
            'pattern_type' => $patternType,
            'alert_level' => $alertLevel,
            'message' => $p['message'] ?? '',
            'recommendation' => $p['recommendation'] ?? '',
        ];
    }

    /**
     * Reduce a bilingual entry (every field is ['fa'=>..,'en'=>..]) to one locale.
     */
    private static function slice(array $entry, string $locale): array
    {
        $out = [];
        foreach ($entry as $field => $val) {
            $out[$field] = is_array($val) ? ($val[$locale] ?? $val['fa'] ?? null) : $val;
        }
        return $out;
    }

    /**
     * @inheritDoc — seed/fallback source for all pattern message text.
     */
    public static function contentDefaults(): array
    {
        $out = [];
        foreach (self::patternContent() as $key => $entry) {
            foreach (['fa', 'en'] as $loc) {
                $out['pattern'][$key][$loc] = self::slice($entry, $loc);
            }
        }
        return $out;
    }

    /**
     * Pattern message text database (seed/fallback), keyed by pattern_type.
     */
    private static function patternContent(): array
    {
        return [
            'insufficient_data' => [
                'message' => [
                    'fa' => 'برای تشخیص الگوها، حداقل ۲ هفته داده نیاز است. به ثبت روزانه ادامه دهید.',
                    'en' => 'At least 2 weeks of data is needed for pattern detection. Continue daily logging.',
                ],
                'recommendation' => [
                    'fa' => 'هر روز علائم خود را ثبت کنید',
                    'en' => 'Log your symptoms every day',
                ],
            ],
            'recurring_pms' => [
                'message' => [
                    'fa' => 'الگوی تکراری PMS در داده‌های شما مشاهده شده. می‌توانید با آمادگی قبلی علائم را مدیریت کنید.',
                    'en' => 'Recurring PMS pattern detected in your data. You can manage symptoms with prior preparation.',
                ],
                'recommendation' => [
                    'fa' => 'یک هفته قبل از پریود، مصرف نمک و کافئین را کم کنید',
                    'en' => 'Reduce salt and caffeine intake a week before period',
                ],
            ],
            'severe_pain' => [
                'message' => [
                    'fa' => 'دردهای شدید مکرر مشاهده شده. در صورت ادامه، با پزشک مشورت کنید.',
                    'en' => 'Recurring severe pain detected. Consult a doctor if it continues.',
                ],
                'recommendation' => [
                    'fa' => 'پیشنهاد می‌شود با متخصص زنان صحبت کنید',
                    'en' => 'It is recommended to talk to a gynecologist',
                ],
            ],
            'frequent_pain' => [
                'message' => [
                    'fa' => 'درد در بیش از ۲۵٪ روزها گزارش شده. راهکارهای مدیریت درد را امتحان کنید.',
                    'en' => 'Pain reported on more than 25% of days. Try pain management strategies.',
                ],
                'recommendation' => [
                    'fa' => 'یوگا، کمپرس گرم و تغذیه ضدالتهابی می‌تواند کمک کند',
                    'en' => 'Yoga, warm compress and anti-inflammatory diet can help',
                ],
            ],
            'low_mood' => [
                'message' => [
                    'fa' => 'خلق پایین در بیش از یک سوم روزها مشاهده شده. اگر این حالت ادامه‌دار است، با متخصص صحبت کنید.',
                    'en' => 'Low mood detected on more than a third of days. If persistent, talk to a specialist.',
                ],
                'recommendation' => [
                    'fa' => 'ورزش منظم، نور آفتاب و در صورت نیاز مشاوره روانشناسی',
                    'en' => 'Regular exercise, sunlight and psychological counseling if needed',
                ],
            ],
            'anxiety_pattern' => [
                'message' => [
                    'fa' => 'اضطراب مکرر مشاهده شده. تکنیک‌های مدیریت استرس می‌تواند کمک کند.',
                    'en' => 'Recurring anxiety detected. Stress management techniques can help.',
                ],
                'recommendation' => [
                    'fa' => 'مدیتیشن، تنفس عمیق و کاهش مصرف کافئین',
                    'en' => 'Meditation, deep breathing and reducing caffeine',
                ],
            ],
            'ttc_tracking' => [
                'message' => [
                    'fa' => 'ثبت منظم داده‌ها به بهبود پیش‌بینی تخمک‌گذاری کمک می‌کند.',
                    'en' => 'Regular data logging helps improve ovulation prediction.',
                ],
                'recommendation' => [
                    'fa' => 'علاوه بر علائم، دمای پایه بدن را هم ثبت کنید',
                    'en' => 'In addition to symptoms, also log basal body temperature',
                ],
            ],
            'poor_sleep' => [
                'message' => [
                    'fa' => 'کیفیت خواب پایین در بیش از یک سوم روزها. خواب ضعیف روی سلامت کلی تأثیر می‌گذارد.',
                    'en' => 'Poor sleep quality on more than a third of days. Poor sleep affects overall health.',
                ],
                'recommendation' => [
                    'fa' => 'بهداشت خواب را رعایت کنید: ساعت خواب منظم، اتاق تاریک، بدون گوشی قبل خواب',
                    'en' => 'Practice sleep hygiene: regular sleep schedule, dark room, no phone before bed',
                ],
            ],
            'chronic_fatigue' => [
                'message' => [
                    'fa' => 'انرژی پایین مداوم مشاهده شده. این می‌تواند نشانه کم‌خونی یا مشکلات تیروئید باشد.',
                    'en' => 'Persistent low energy detected. This could indicate anemia or thyroid issues.',
                ],
                'recommendation' => [
                    'fa' => 'آزمایش خون برای بررسی آهن و تیروئید توصیه می‌شود',
                    'en' => 'Blood test for iron and thyroid is recommended',
                ],
            ],
            'increasing_nausea' => [
                'message' => [
                    'fa' => 'تهوع در حال افزایش است. این در هفته‌های ۸-۱۲ طبیعی است و معمولاً بعداً کم می‌شود.',
                    'en' => 'Nausea is increasing. This is normal in weeks 8-12 and usually decreases later.',
                ],
                'recommendation' => [
                    'fa' => 'غذاهای کوچک و مکرر، زنجبیل و ویتامین B6 می‌تواند کمک کند',
                    'en' => 'Small frequent meals, ginger and vitamin B6 can help',
                ],
            ],
            'improving_symptoms' => [
                'message' => [
                    'fa' => 'علائم در حال بهبود هستند! این نشانه ورود به دوران طلایی بارداری است.',
                    'en' => 'Symptoms are improving! This is a sign of entering the golden period of pregnancy.',
                ],
                'recommendation' => [
                    'fa' => 'از این دوره برای ورزش ملایم و فعالیت‌های لذت‌بخش استفاده کنید',
                    'en' => 'Use this period for gentle exercise and enjoyable activities',
                ],
            ],
        ];
    }
}
