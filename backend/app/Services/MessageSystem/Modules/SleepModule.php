<?php

namespace App\Services\MessageSystem\Modules;

use App\Services\MessageSystem\Contracts\ProvidesMessageContent;
use App\Services\MessageSystem\Core\MessageContext;
use App\Services\MessageSystem\Support\MessageContentRepository;

class SleepModule implements ProvidesMessageContent
{
    public function __construct(
        private readonly string $locale = 'fa',
    ) {}

    /**
     * Get sleep tips based on context
     */
    public function getTips(MessageContext $context): array
    {
        if ($context->isPregnancyMode()) {
            return $this->getPregnancySleep($context);
        }

        return $this->getCycleSleep($context);
    }

    private function content(): MessageContentRepository
    {
        return app(MessageContentRepository::class);
    }

    /**
     * Get cycle-based sleep tips
     */
    private function getCycleSleep(MessageContext $context): array
    {
        $phase = $context->cyclePhase;
        $tips = $this->getCyclePhaseSleep($phase);

        // Add symptom-specific tips
        if (!empty($context->symptoms)) {
            if (in_array('poor_sleep', $context->symptoms)) {
                $fallback = self::slice(self::sleepSymptomSource()['poor_sleep'], $this->locale);
                $p = $this->content()->resolve('sleep_symptom', 'poor_sleep', $this->locale, $fallback);
                $tips['symptom_tip'] = $p['tip'] ?? '';
            }
        }

        return $tips;
    }

    /**
     * Get pregnancy-based sleep tips
     */
    private function getPregnancySleep(MessageContext $context): array
    {
        $trimester = $context->trimester ?? 1;
        return $this->getTrimesterSleep($trimester);
    }

    /**
     * Get cycle phase sleep recommendations (DB-editable, falls back to defaults)
     */
    private function getCyclePhaseSleep(?string $phase): array
    {
        $tips = self::cyclePhaseSleepSource();
        $itemKey = ($phase !== null && array_key_exists($phase, $tips)) ? $phase : 'follicular';
        $fallback = self::slice($tips[$itemKey], $this->locale);

        $p = $this->content()->resolve('sleep_cycle', $itemKey, $this->locale, $fallback);

        return [
            'recommended_hours' => $p['recommended_hours'] ?? '',
            'quality_focus' => $p['quality_focus'] ?? '',
            'tips' => $p['tips'] ?? [],
            'avoid' => $p['avoid'] ?? [],
        ];
    }

    /**
     * Get trimester sleep recommendations (DB-editable, falls back to defaults)
     */
    private function getTrimesterSleep(int $trimester): array
    {
        $tips = self::trimesterSleepSource();
        $key = array_key_exists($trimester, $tips) ? $trimester : 1;
        $itemKey = (string) $key;
        $fallback = self::slice($tips[$key], $this->locale);

        $p = $this->content()->resolve('sleep_trimester', $itemKey, $this->locale, $fallback);

        return [
            'recommended_hours' => $p['recommended_hours'] ?? '',
            'quality_focus' => $p['quality_focus'] ?? '',
            'tips' => $p['tips'] ?? [],
            'avoid' => $p['avoid'] ?? [],
            'position' => $p['position'] ?? '',
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
     * @inheritDoc — seed/fallback source for all sleep message text.
     */
    public static function contentDefaults(): array
    {
        $out = [];
        foreach (self::cyclePhaseSleepSource() as $key => $entry) {
            foreach (['fa', 'en'] as $loc) {
                $out['sleep_cycle'][$key][$loc] = self::slice($entry, $loc);
            }
        }
        foreach (self::sleepSymptomSource() as $key => $entry) {
            foreach (['fa', 'en'] as $loc) {
                $out['sleep_symptom'][$key][$loc] = self::slice($entry, $loc);
            }
        }
        foreach (self::trimesterSleepSource() as $key => $entry) {
            foreach (['fa', 'en'] as $loc) {
                $out['sleep_trimester'][(string) $key][$loc] = self::slice($entry, $loc);
            }
        }
        return $out;
    }

    /**
     * Cycle phase sleep database (seed/fallback)
     */
    private static function cyclePhaseSleepSource(): array
    {
        return [
            'menstruation' => [
                'recommended_hours' => '8-9',
                'quality_focus' => ['fa' => 'استراحت بیشتر', 'en' => 'More rest'],
                'tips' => [
                    'fa' => ['خواب بیشتر نیاز است', 'کمپرس گرم برای کرامپ', 'اتاق خنک'],
                    'en' => ['More sleep is needed', 'Warm compress for cramps', 'Cool room'],
                ],
                'avoid' => [
                    'fa' => ['کافئین بعد از ظهر', 'گوشی قبل خواب'],
                    'en' => ['Caffeine after noon', 'Phone before bed'],
                ],
            ],
            'follicular' => [
                'recommended_hours' => '7-8',
                'quality_focus' => ['fa' => 'خواب بهینه', 'en' => 'Optimal sleep'],
                'tips' => [
                    'fa' => ['خواب عمیق‌تر در این فاز', 'زمان مناسب برای تنظیم ساعت خواب'],
                    'en' => ['Deeper sleep in this phase', 'Good time to set sleep schedule'],
                ],
                'avoid' => [
                    'fa' => ['بی‌خوابی', 'ساعت خواب نامنظم'],
                    'en' => ['Sleep deprivation', 'Irregular sleep schedule'],
                ],
            ],
            'ovulation' => [
                'recommended_hours' => '7-8',
                'quality_focus' => ['fa' => 'انرژی بالا', 'en' => 'High energy'],
                'tips' => [
                    'fa' => ['ممکنه کمتر خسته باشی', 'از انرژی استفاده کن ولی خواب رو فراموش نکن'],
                    'en' => ['You may feel less tired', 'Use the energy but don\'t forget sleep'],
                ],
                'avoid' => [
                    'fa' => ['بیدار موندن دیروقت', 'از دست دادن خواب'],
                    'en' => ['Staying up late', 'Missing sleep'],
                ],
            ],
            'luteal' => [
                'recommended_hours' => '8-9',
                'quality_focus' => ['fa' => 'کیفیت خواب ممکنه کم بشه', 'en' => 'Sleep quality may decrease'],
                'tips' => [
                    'fa' => ['ممکنه بی‌خوابی داشته باشی', 'منیزیم قبل خواب', 'اتاق تاریک و خنک', 'مدیتیشن قبل خواب'],
                    'en' => ['You may have insomnia', 'Magnesium before bed', 'Dark and cool room', 'Meditation before bed'],
                ],
                'avoid' => [
                    'fa' => ['صفحه نمایش قبل خواب', 'غذای سنگین شب', 'کافئین'],
                    'en' => ['Screen before bed', 'Heavy dinner', 'Caffeine'],
                ],
            ],
        ];
    }

    /**
     * Symptom-based sleep database (seed/fallback)
     */
    private static function sleepSymptomSource(): array
    {
        return [
            'poor_sleep' => [
                'tip' => [
                    'fa' => 'کیفیت خواب پایین ثبت شده. بهداشت خواب را رعایت کنید.',
                    'en' => 'Poor sleep quality logged. Practice sleep hygiene.',
                ],
            ],
        ];
    }

    /**
     * Trimester sleep database (seed/fallback)
     */
    private static function trimesterSleepSource(): array
    {
        return [
            1 => [
                'recommended_hours' => '8-10',
                'quality_focus' => ['fa' => 'خستگی شدید طبیعی است', 'en' => 'Extreme fatigue is normal'],
                'tips' => [
                    'fa' => ['چرت‌های کوتاه در طول روز', 'زود بخواب', 'به بدنت گوش بده', 'هورمون‌ها باعث خواب‌آلودگی می‌شن'],
                    'en' => ['Short naps during day', 'Sleep early', 'Listen to your body', 'Hormones cause sleepiness'],
                ],
                'avoid' => [
                    'fa' => ['مقاومت در برابر خواب', 'کار زیاد'],
                    'en' => ['Resisting sleep', 'Overwork'],
                ],
                'position' => ['fa' => 'هنوز هر پوزیشنی مناسب است', 'en' => 'Any position is still suitable'],
            ],
            2 => [
                'recommended_hours' => '8-9',
                'quality_focus' => ['fa' => 'انرژی برگشته', 'en' => 'Energy is back'],
                'tips' => [
                    'fa' => ['خواب بهتر از سه‌ماهه اول', 'شروع عادت به خوابیدن به پهلو', 'بالش بین زانوها'],
                    'en' => ['Better sleep than first trimester', 'Start getting used to side sleeping', 'Pillow between knees'],
                ],
                'avoid' => [
                    'fa' => ['خوابیدن به پشت طولانی'],
                    'en' => ['Sleeping on back for long'],
                ],
                'position' => ['fa' => 'پهلوی چپ بهترین است', 'en' => 'Left side is best'],
            ],
            3 => [
                'recommended_hours' => '8-9',
                'quality_focus' => ['fa' => 'خواب راحت سخت می‌شه', 'en' => 'Comfortable sleep becomes harder'],
                'tips' => [
                    'fa' => ['بالش بارداری', 'سر تخت بالاتر', 'ادرار قبل خواب', 'تنفس عمیق', 'نوشیدنی کم قبل خواب'],
                    'en' => ['Pregnancy pillow', 'Elevated head', 'Urinate before sleep', 'Deep breathing', 'Less fluids before bed'],
                ],
                'avoid' => [
                    'fa' => ['خوابیدن به پشت', 'مایعات زیاد قبل خواب'],
                    'en' => ['Sleeping on back', 'Too much fluids before bed'],
                ],
                'position' => ['fa' => 'حتماً به پهلو بخوابید، ترجیحاً چپ', 'en' => 'Must sleep on side, preferably left'],
            ],
        ];
    }
}
