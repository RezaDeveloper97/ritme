<?php

namespace App\Services\MessageSystem\Layers;

use App\Models\User;
use App\Services\MessageSystem\Contracts\ProvidesMessageContent;
use App\Services\MessageSystem\Core\MessageContext;
use App\Services\MessageSystem\Support\MessageContentRepository;

/**
 * Layer 3: Correlation Engine
 * Detects relationships between symptoms and provides insights
 */
class CorrelationLayer implements ProvidesMessageContent
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
     * Analyze correlations based on context
     */
    public function analyze(MessageContext $context): array
    {
        $correlations = [];

        if (!$context->symptoms || empty($context->symptoms)) {
            return $correlations;
        }

        // Analyze based on mode
        if ($context->isCycleMode()) {
            $correlations = array_merge($correlations, $this->analyzeCycleCorrelations($context));
        } elseif ($context->isPregnancyMode()) {
            $correlations = array_merge($correlations, $this->analyzePregnancyCorrelations($context));
        }

        // Common correlations (applicable to all modes)
        $correlations = array_merge($correlations, $this->analyzeCommonCorrelations($context));

        return $correlations;
    }

    /**
     * Analyze cycle-specific correlations
     */
    private function analyzeCycleCorrelations(MessageContext $context): array
    {
        $correlations = [];
        $symptoms = $context->symptoms;
        $phase = $context->cyclePhase;

        // Sleep + Mood correlation
        if (in_array('poor_sleep', $symptoms) && (in_array('mood_sad', $symptoms) || in_array('mood_angry', $symptoms))) {
            $correlations[] = $this->correlation('correlation_cycle', 'sleep_mood', false);
        }

        // Stress + Physical symptoms correlation
        if (in_array('mood_anxious', $symptoms) && (in_array('headache', $symptoms) || in_array('cramps', $symptoms))) {
            $correlations[] = $this->correlation('correlation_cycle', 'stress_physical', false);
        }

        // PMS window correlations
        if ($context->isPmsWindow) {
            if (in_array('bloating', $symptoms) || in_array('mood_sad', $symptoms)) {
                $correlations[] = $this->correlation('correlation_cycle', 'pms_symptoms', false);
            }
        }

        // Fertile window + Energy (TTC)
        if ($context->isTTC() && $context->isFertileWindow) {
            $correlations[] = $this->correlation('correlation_cycle', 'fertile_window_ttc', false);
        }

        // Premium: Detailed phase-symptom correlation
        if ($phase === 'luteal' && in_array('low_energy', $symptoms)) {
            $correlations[] = $this->correlation('correlation_cycle', 'luteal_energy_drop', true);
        }

        // Premium: Hormone-skin correlation
        if (in_array('acne', $symptoms) && ($phase === 'luteal' || $phase === 'menstruation')) {
            $correlations[] = $this->correlation('correlation_cycle', 'hormone_skin', true);
        }

        return $correlations;
    }

    /**
     * Analyze pregnancy-specific correlations
     */
    private function analyzePregnancyCorrelations(MessageContext $context): array
    {
        $correlations = [];
        $symptoms = $context->symptoms;
        $trimester = $context->trimester;

        // First trimester nausea + fatigue
        if ($trimester === 1 && in_array('nausea', $symptoms) && in_array('fatigue', $symptoms)) {
            $correlations[] = $this->correlation('correlation_pregnancy', 'first_trimester_combo', false);
        }

        // Third trimester discomfort
        if ($trimester === 3 && (in_array('backache', $symptoms) || in_array('fatigue', $symptoms))) {
            $correlations[] = $this->correlation('correlation_pregnancy', 'third_trimester_discomfort', false);
        }

        // Premium: Mood changes in pregnancy
        if (in_array('mood_sad', $symptoms) || in_array('mood_anxious', $symptoms)) {
            $correlations[] = $this->correlation('correlation_pregnancy', 'pregnancy_mood', true);
        }

        return $correlations;
    }

    /**
     * Analyze common correlations (all modes)
     */
    private function analyzeCommonCorrelations(MessageContext $context): array
    {
        $correlations = [];
        $symptoms = $context->symptoms;

        // Dehydration indicators
        if (in_array('headache', $symptoms) && in_array('fatigue', $symptoms)) {
            $correlations[] = $this->correlation('correlation_common', 'dehydration_indicator', false);
        }

        // Sleep deprivation pattern
        if (in_array('poor_sleep', $symptoms) && in_array('low_energy', $symptoms)) {
            $correlations[] = $this->correlation('correlation_common', 'sleep_deprivation', false);
        }

        return $correlations;
    }

    /**
     * Build a single correlation entry. Detection logic, the stable id/type and
     * the is_premium_only flag stay in code; only insight_message and action are
     * editable content (DB-editable, falls back to the seeded literal).
     */
    private function correlation(string $group, string $type, bool $isPremiumOnly): array
    {
        $entry = self::correlationContent()[$group][$type];
        $fallback = self::slice($entry, $this->locale);
        $p = $this->content()->resolve($group, $type, $this->locale, $fallback);

        return [
            'type' => $type,
            'insight_message' => $p['insight_message'] ?? '',
            'action' => $p['action'] ?? '',
            'is_premium_only' => $isPremiumOnly,
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
     * @inheritDoc — seed/fallback source for all correlation insight text.
     */
    public static function contentDefaults(): array
    {
        $out = [];
        foreach (self::correlationContent() as $group => $entries) {
            foreach ($entries as $key => $entry) {
                foreach (['fa', 'en'] as $loc) {
                    $out[$group][$key][$loc] = self::slice($entry, $loc);
                }
            }
        }
        return $out;
    }

    /**
     * Correlation insight text database (seed/fallback), grouped by mode and
     * keyed by the correlation's stable type.
     */
    private static function correlationContent(): array
    {
        return [
            'correlation_cycle' => [
                'sleep_mood' => [
                    'insight_message' => [
                        'fa' => 'کیفیت خواب پایین می‌تونه روی خلق و خو تأثیر بذاره. بهبود خواب ممکنه به بهتر شدن حالت روحی کمک کنه.',
                        'en' => 'Poor sleep quality can affect mood. Improving sleep may help improve mood.',
                    ],
                    'action' => [
                        'fa' => 'سعی کن امشب زودتر بخوابی و از گوشی قبل خواب استفاده نکنی',
                        'en' => 'Try to sleep earlier tonight and avoid phone before bed',
                    ],
                ],
                'stress_physical' => [
                    'insight_message' => [
                        'fa' => 'استرس می‌تونه علائم فیزیکی مثل سردرد و کرامپ رو تشدید کنه.',
                        'en' => 'Stress can intensify physical symptoms like headache and cramps.',
                    ],
                    'action' => [
                        'fa' => 'تکنیک‌های آرامش‌بخش مثل تنفس عمیق رو امتحان کن',
                        'en' => 'Try relaxation techniques like deep breathing',
                    ],
                ],
                'pms_symptoms' => [
                    'insight_message' => [
                        'fa' => 'علائم PMS در این دوره طبیعی هستن. چند روز دیگه بهتر میشی.',
                        'en' => 'PMS symptoms are normal in this period. You\'ll feel better in a few days.',
                    ],
                    'action' => [
                        'fa' => 'مصرف نمک رو کم کن و ورزش سبک انجام بده',
                        'en' => 'Reduce salt intake and do light exercise',
                    ],
                ],
                'fertile_window_ttc' => [
                    'insight_message' => [
                        'fa' => 'این بهترین زمان برای باردار شدن است. استروژن در اوج و شانس لقاح بالاست.',
                        'en' => 'This is the best time to conceive. Estrogen is at peak and conception chances are high.',
                    ],
                    'action' => [
                        'fa' => 'رابطه هر ۲۴-۴۸ ساعت توصیه می‌شه',
                        'en' => 'Intercourse every 24-48 hours is recommended',
                    ],
                ],
                'luteal_energy_drop' => [
                    'insight_message' => [
                        'fa' => 'کاهش انرژی در فاز لوتئال به دلیل افزایش پروژسترون است. این کاملاً طبیعیه.',
                        'en' => 'Energy drop in luteal phase is due to rising progesterone. This is completely normal.',
                    ],
                    'action' => [
                        'fa' => 'به بدنت گوش بده و استراحت کافی داشته باش',
                        'en' => 'Listen to your body and get adequate rest',
                    ],
                ],
                'hormone_skin' => [
                    'insight_message' => [
                        'fa' => 'جوش‌های هورمونی در این فاز شایع هستن. با شروع فاز فولیکولار بهتر میشه.',
                        'en' => 'Hormonal acne is common in this phase. It will improve with follicular phase.',
                    ],
                    'action' => [
                        'fa' => 'پاکسازی ملایم و اجتناب از دست زدن به صورت',
                        'en' => 'Gentle cleansing and avoid touching face',
                    ],
                ],
            ],
            'correlation_pregnancy' => [
                'first_trimester_combo' => [
                    'insight_message' => [
                        'fa' => 'ترکیب تهوع و خستگی در سه‌ماهه اول بسیار شایع است و نشانه تغییرات هورمونی قوی است.',
                        'en' => 'Combination of nausea and fatigue is very common in first trimester, showing strong hormonal changes.',
                    ],
                    'action' => [
                        'fa' => 'غذاهای کوچک و مکرر و استراحت کوتاه در طول روز',
                        'en' => 'Small frequent meals and short rests during day',
                    ],
                ],
                'third_trimester_discomfort' => [
                    'insight_message' => [
                        'fa' => 'ناراحتی‌های سه‌ماهه سوم به دلیل رشد نوزاد و آماده‌سازی بدن برای زایمان است.',
                        'en' => 'Third trimester discomforts are due to baby growth and body preparing for delivery.',
                    ],
                    'action' => [
                        'fa' => 'شنا، یوگای بارداری و استراحت با پاهای بالا',
                        'en' => 'Swimming, prenatal yoga and resting with elevated legs',
                    ],
                ],
                'pregnancy_mood' => [
                    'insight_message' => [
                        'fa' => 'تغییرات خلقی در بارداری طبیعی هستن اما اگر ادامه‌دار باشن با پزشک صحبت کنید.',
                        'en' => 'Mood changes in pregnancy are normal but if persistent, talk to your doctor.',
                    ],
                    'action' => [
                        'fa' => 'با همسر یا دوست صحبت کن و در صورت نیاز با پزشک مشورت کن',
                        'en' => 'Talk to partner or friend and consult doctor if needed',
                    ],
                ],
            ],
            'correlation_common' => [
                'dehydration_indicator' => [
                    'insight_message' => [
                        'fa' => 'ترکیب سردرد و خستگی می‌تونه نشانه کم‌آبی باشه.',
                        'en' => 'Combination of headache and fatigue could indicate dehydration.',
                    ],
                    'action' => [
                        'fa' => 'حداقل ۸ لیوان آب در روز بنوش',
                        'en' => 'Drink at least 8 glasses of water daily',
                    ],
                ],
                'sleep_deprivation' => [
                    'insight_message' => [
                        'fa' => 'کم‌خوابی مستقیماً روی انرژی روزانه تأثیر می‌گذاره.',
                        'en' => 'Sleep deprivation directly affects daily energy.',
                    ],
                    'action' => [
                        'fa' => 'یک برنامه خواب منظم تنظیم کن',
                        'en' => 'Set a regular sleep schedule',
                    ],
                ],
            ],
        ];
    }
}
