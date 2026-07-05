<?php

namespace App\Services\MessageSystem\Layers;

use App\Models\DailyHealthLog;
use App\Models\User;
use App\Services\MessageSystem\Core\MessageContext;

/**
 * Layer 4: Pattern Recognition Engine (Premium Only)
 * Detects long-term patterns across multiple cycles
 */
class PatternLayer
{
    public function __construct(
        private readonly User $user,
        private readonly string $locale = 'fa',
    ) {}

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
            $patterns[] = [
                'pattern_type' => 'insufficient_data',
                'alert_level' => 'info',
                'message' => $this->locale === 'fa'
                    ? 'برای تشخیص الگوها، حداقل ۲ هفته داده نیاز است. به ثبت روزانه ادامه دهید.'
                    : 'At least 2 weeks of data is needed for pattern detection. Continue daily logging.',
                'recommendation' => $this->locale === 'fa'
                    ? 'هر روز علائم خود را ثبت کنید'
                    : 'Log your symptoms every day',
            ];
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
            return [
                'pattern_type' => 'recurring_pms',
                'alert_level' => 'info',
                'message' => $this->locale === 'fa'
                    ? 'الگوی تکراری PMS در داده‌های شما مشاهده شده. می‌توانید با آمادگی قبلی علائم را مدیریت کنید.'
                    : 'Recurring PMS pattern detected in your data. You can manage symptoms with prior preparation.',
                'recommendation' => $this->locale === 'fa'
                    ? 'یک هفته قبل از پریود، مصرف نمک و کافئین را کم کنید'
                    : 'Reduce salt and caffeine intake a week before period',
            ];
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
            return [
                'pattern_type' => 'severe_pain',
                'alert_level' => 'warning',
                'message' => $this->locale === 'fa'
                    ? 'دردهای شدید مکرر مشاهده شده. در صورت ادامه، با پزشک مشورت کنید.'
                    : 'Recurring severe pain detected. Consult a doctor if it continues.',
                'recommendation' => $this->locale === 'fa'
                    ? 'پیشنهاد می‌شود با متخصص زنان صحبت کنید'
                    : 'It is recommended to talk to a gynecologist',
            ];
        }

        if ($painDays >= 7) {
            return [
                'pattern_type' => 'frequent_pain',
                'alert_level' => 'info',
                'message' => $this->locale === 'fa'
                    ? 'درد در بیش از ۲۵٪ روزها گزارش شده. راهکارهای مدیریت درد را امتحان کنید.'
                    : 'Pain reported on more than 25% of days. Try pain management strategies.',
                'recommendation' => $this->locale === 'fa'
                    ? 'یوگا، کمپرس گرم و تغذیه ضدالتهابی می‌تواند کمک کند'
                    : 'Yoga, warm compress and anti-inflammatory diet can help',
            ];
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
            return [
                'pattern_type' => 'low_mood',
                'alert_level' => 'warning',
                'message' => $this->locale === 'fa'
                    ? 'خلق پایین در بیش از یک سوم روزها مشاهده شده. اگر این حالت ادامه‌دار است، با متخصص صحبت کنید.'
                    : 'Low mood detected on more than a third of days. If persistent, talk to a specialist.',
                'recommendation' => $this->locale === 'fa'
                    ? 'ورزش منظم، نور آفتاب و در صورت نیاز مشاوره روانشناسی'
                    : 'Regular exercise, sunlight and psychological counseling if needed',
            ];
        }

        if ($anxiousDays >= 7) {
            return [
                'pattern_type' => 'anxiety_pattern',
                'alert_level' => 'info',
                'message' => $this->locale === 'fa'
                    ? 'اضطراب مکرر مشاهده شده. تکنیک‌های مدیریت استرس می‌تواند کمک کند.'
                    : 'Recurring anxiety detected. Stress management techniques can help.',
                'recommendation' => $this->locale === 'fa'
                    ? 'مدیتیشن، تنفس عمیق و کاهش مصرف کافئین'
                    : 'Meditation, deep breathing and reducing caffeine',
            ];
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
        return [
            'pattern_type' => 'ttc_tracking',
            'alert_level' => 'info',
            'message' => $this->locale === 'fa'
                ? 'ثبت منظم داده‌ها به بهبود پیش‌بینی تخمک‌گذاری کمک می‌کند.'
                : 'Regular data logging helps improve ovulation prediction.',
            'recommendation' => $this->locale === 'fa'
                ? 'علاوه بر علائم، دمای پایه بدن را هم ثبت کنید'
                : 'In addition to symptoms, also log basal body temperature',
        ];
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
            return [
                'pattern_type' => 'poor_sleep',
                'alert_level' => 'warning',
                'message' => $this->locale === 'fa'
                    ? 'کیفیت خواب پایین در بیش از یک سوم روزها. خواب ضعیف روی سلامت کلی تأثیر می‌گذارد.'
                    : 'Poor sleep quality on more than a third of days. Poor sleep affects overall health.',
                'recommendation' => $this->locale === 'fa'
                    ? 'بهداشت خواب را رعایت کنید: ساعت خواب منظم، اتاق تاریک، بدون گوشی قبل خواب'
                    : 'Practice sleep hygiene: regular sleep schedule, dark room, no phone before bed',
            ];
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
            return [
                'pattern_type' => 'chronic_fatigue',
                'alert_level' => 'warning',
                'message' => $this->locale === 'fa'
                    ? 'انرژی پایین مداوم مشاهده شده. این می‌تواند نشانه کم‌خونی یا مشکلات تیروئید باشد.'
                    : 'Persistent low energy detected. This could indicate anemia or thyroid issues.',
                'recommendation' => $this->locale === 'fa'
                    ? 'آزمایش خون برای بررسی آهن و تیروئید توصیه می‌شود'
                    : 'Blood test for iron and thyroid is recommended',
            ];
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
            return [
                'pattern_type' => 'increasing_nausea',
                'alert_level' => 'info',
                'message' => $this->locale === 'fa'
                    ? 'تهوع در حال افزایش است. این در هفته‌های ۸-۱۲ طبیعی است و معمولاً بعداً کم می‌شود.'
                    : 'Nausea is increasing. This is normal in weeks 8-12 and usually decreases later.',
                'recommendation' => $this->locale === 'fa'
                    ? 'غذاهای کوچک و مکرر، زنجبیل و ویتامین B6 می‌تواند کمک کند'
                    : 'Small frequent meals, ginger and vitamin B6 can help',
            ];
        }

        if ($trimester === 2 && $recentNausea < $olderNausea - 2) {
            return [
                'pattern_type' => 'improving_symptoms',
                'alert_level' => 'info',
                'message' => $this->locale === 'fa'
                    ? 'علائم در حال بهبود هستند! این نشانه ورود به دوران طلایی بارداری است.'
                    : 'Symptoms are improving! This is a sign of entering the golden period of pregnancy.',
                'recommendation' => $this->locale === 'fa'
                    ? 'از این دوره برای ورزش ملایم و فعالیت‌های لذت‌بخش استفاده کنید'
                    : 'Use this period for gentle exercise and enjoyable activities',
            ];
        }

        return null;
    }
}
