<?php

namespace App\Services\MessageSystem\Modules;

use App\Services\MessageSystem\Contracts\ProvidesMessageContent;
use App\Services\MessageSystem\Core\MessageContext;
use App\Services\MessageSystem\Support\MessageContentRepository;

class ExerciseModule implements ProvidesMessageContent
{
    public function __construct(
        private readonly string $locale = 'fa',
    ) {}

    /**
     * Get exercise tips based on context
     */
    public function getTips(MessageContext $context): array
    {
        if ($context->isPregnancyMode()) {
            return $this->getPregnancyExercise($context);
        }

        return $this->getCycleExercise($context);
    }

    private function content(): MessageContentRepository
    {
        return app(MessageContentRepository::class);
    }

    /**
     * Get cycle-based exercise tips
     */
    private function getCycleExercise(MessageContext $context): array
    {
        $phase = $context->cyclePhase;
        $tips = $this->getCyclePhaseExercise($phase);

        // Add symptom-specific modifications
        if (!empty($context->symptoms)) {
            if (in_array('cramps', $context->symptoms) || in_array('pain', $context->symptoms)) {
                $tips['symptom_modification'] = $this->exerciseSymptomTip('pain');
            }
            if (in_array('low_energy', $context->symptoms) || in_array('fatigue', $context->symptoms)) {
                $tips['symptom_modification'] = $this->exerciseSymptomTip('fatigue');
            }
        }

        // TTC specific
        if ($context->isTTC()) {
            $fallback = self::slice(self::ttcExerciseSource(), $this->locale);
            $p = $this->content()->resolve('exercise_ttc', 'note', $this->locale, $fallback);
            $tips['ttc_note'] = $p['tip'] ?? '';
        }

        return $tips;
    }

    /**
     * Get pregnancy-based exercise tips
     */
    private function getPregnancyExercise(MessageContext $context): array
    {
        $trimester = $context->trimester ?? 1;
        return $this->getTrimesterExercise($trimester);
    }

    /**
     * Resolve a single symptom exercise modification (DB-editable, falls back to defaults)
     */
    private function exerciseSymptomTip(string $key): string
    {
        $entries = self::symptomExerciseSource();
        $fallback = self::slice($entries[$key], $this->locale);
        $p = $this->content()->resolve('exercise_symptom', $key, $this->locale, $fallback);

        return $p['tip'] ?? '';
    }

    /**
     * Get cycle phase exercise recommendations (DB-editable, falls back to defaults)
     */
    private function getCyclePhaseExercise(?string $phase): array
    {
        $tips = self::cyclePhaseExerciseSource();
        $itemKey = ($phase !== null && array_key_exists($phase, $tips)) ? $phase : 'follicular';
        $fallback = self::slice($tips[$itemKey], $this->locale);

        $p = $this->content()->resolve('exercise_cycle', $itemKey, $this->locale, $fallback);

        return [
            'intensity' => $p['intensity'] ?? '',
            'recommended' => $p['recommended'] ?? [],
            'avoid' => $p['avoid'] ?? [],
            'tip' => $p['tip'] ?? '',
            'duration' => $p['duration'] ?? '',
        ];
    }

    /**
     * Get trimester exercise recommendations (DB-editable, falls back to defaults)
     */
    private function getTrimesterExercise(int $trimester): array
    {
        $tips = self::trimesterExerciseSource();
        $key = array_key_exists($trimester, $tips) ? $trimester : 1;
        $itemKey = (string) $key;
        $fallback = self::slice($tips[$key], $this->locale);

        $p = $this->content()->resolve('exercise_trimester', $itemKey, $this->locale, $fallback);

        return [
            'intensity' => $p['intensity'] ?? '',
            'recommended' => $p['recommended'] ?? [],
            'avoid' => $p['avoid'] ?? [],
            'tip' => $p['tip'] ?? '',
            'duration' => $p['duration'] ?? '',
            'precautions' => $p['precautions'] ?? [],
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
     * @inheritDoc — seed/fallback source for all exercise message text.
     */
    public static function contentDefaults(): array
    {
        $out = [];
        foreach (self::cyclePhaseExerciseSource() as $key => $entry) {
            foreach (['fa', 'en'] as $loc) {
                $out['exercise_cycle'][$key][$loc] = self::slice($entry, $loc);
            }
        }
        foreach (self::symptomExerciseSource() as $key => $entry) {
            foreach (['fa', 'en'] as $loc) {
                $out['exercise_symptom'][$key][$loc] = self::slice($entry, $loc);
            }
        }
        foreach (['fa', 'en'] as $loc) {
            $out['exercise_ttc']['note'][$loc] = self::slice(self::ttcExerciseSource(), $loc);
        }
        foreach (self::trimesterExerciseSource() as $key => $entry) {
            foreach (['fa', 'en'] as $loc) {
                $out['exercise_trimester'][(string) $key][$loc] = self::slice($entry, $loc);
            }
        }
        return $out;
    }

    /**
     * Cycle phase exercise database (seed/fallback)
     */
    private static function cyclePhaseExerciseSource(): array
    {
        return [
            'menstruation' => [
                'intensity' => ['fa' => 'کم تا متوسط', 'en' => 'Low to moderate'],
                'recommended' => [
                    'fa' => ['پیاده‌روی', 'یوگای ریستوراتیو', 'کشش ملایم', 'شنا'],
                    'en' => ['Walking', 'Restorative yoga', 'Gentle stretching', 'Swimming'],
                ],
                'avoid' => [
                    'fa' => ['HIIT شدید', 'وزنه سنگین', 'وارونگی در یوگا'],
                    'en' => ['Intense HIIT', 'Heavy weights', 'Inversions in yoga'],
                ],
                'tip' => ['fa' => 'به بدنت گوش بده. اگر انرژی نداری، استراحت کن.', 'en' => 'Listen to your body. If you have no energy, rest.'],
                'duration' => '20-30',
            ],
            'follicular' => [
                'intensity' => ['fa' => 'متوسط تا بالا', 'en' => 'Moderate to high'],
                'recommended' => [
                    'fa' => ['دویدن', 'HIIT', 'وزنه‌برداری', 'دوچرخه', 'رقص'],
                    'en' => ['Running', 'HIIT', 'Weight training', 'Cycling', 'Dancing'],
                ],
                'avoid' => [
                    'fa' => ['بی‌فعالیتی'],
                    'en' => ['Inactivity'],
                ],
                'tip' => ['fa' => 'انرژی در اوج است! بهترین زمان برای چالش‌های جدید و رکوردشکنی.', 'en' => 'Energy is at peak! Best time for new challenges and personal records.'],
                'duration' => '45-60',
            ],
            'ovulation' => [
                'intensity' => ['fa' => 'بالا', 'en' => 'High'],
                'recommended' => [
                    'fa' => ['تمرینات قدرتی', 'HIIT', 'ورزش‌های گروهی', 'کراس‌فیت'],
                    'en' => ['Strength training', 'HIIT', 'Group sports', 'CrossFit'],
                ],
                'avoid' => [
                    'fa' => ['هیچی! همه چیز خوبه'],
                    'en' => ['Nothing! Everything is good'],
                ],
                'tip' => ['fa' => 'استروژن بالاست، عضلات قوی‌تر و ریکاوری سریع‌تر.', 'en' => 'Estrogen is high, muscles are stronger and recovery is faster.'],
                'duration' => '45-60',
            ],
            'luteal' => [
                'intensity' => ['fa' => 'متوسط (کم می‌شه)', 'en' => 'Moderate (decreasing)'],
                'recommended' => [
                    'fa' => ['پیلاتس', 'یوگا', 'پیاده‌روی', 'وزنه سبک', 'شنا'],
                    'en' => ['Pilates', 'Yoga', 'Walking', 'Light weights', 'Swimming'],
                ],
                'avoid' => [
                    'fa' => ['ورزش‌های خیلی شدید', 'تمرینات طولانی'],
                    'en' => ['Very intense workouts', 'Long training sessions'],
                ],
                'tip' => ['fa' => 'انرژی کم‌کم کم می‌شه. ورزش ملایم به کاهش علائم PMS کمک می‌کنه.', 'en' => 'Energy is gradually decreasing. Light exercise helps reduce PMS symptoms.'],
                'duration' => '30-45',
            ],
        ];
    }

    /**
     * Symptom-based exercise modification database (seed/fallback)
     */
    private static function symptomExerciseSource(): array
    {
        return [
            'pain' => [
                'tip' => [
                    'fa' => 'با توجه به درد، ورزش‌های کششی ملایم توصیه می‌شود',
                    'en' => 'Due to pain, gentle stretching exercises are recommended',
                ],
            ],
            'fatigue' => [
                'tip' => [
                    'fa' => 'با توجه به خستگی، پیاده‌روی سبک کافی است',
                    'en' => 'Due to fatigue, light walking is enough',
                ],
            ],
        ];
    }

    /**
     * TTC exercise note database (seed/fallback)
     */
    private static function ttcExerciseSource(): array
    {
        return [
            'tip' => [
                'fa' => 'ورزش متوسط برای باروری مفید است. از ورزش‌های خیلی شدید اجتناب کنید.',
                'en' => 'Moderate exercise is good for fertility. Avoid very intense workouts.',
            ],
        ];
    }

    /**
     * Trimester exercise database (seed/fallback)
     */
    private static function trimesterExerciseSource(): array
    {
        return [
            1 => [
                'intensity' => ['fa' => 'کم تا متوسط', 'en' => 'Low to moderate'],
                'recommended' => [
                    'fa' => ['پیاده‌روی', 'شنا', 'یوگای بارداری', 'تمرینات کگل'],
                    'en' => ['Walking', 'Swimming', 'Prenatal yoga', 'Kegel exercises'],
                ],
                'avoid' => [
                    'fa' => ['ورزش‌های پرخطر', 'ورزش‌های تماسی', 'وزنه سنگین', 'غوص'],
                    'en' => ['High-risk sports', 'Contact sports', 'Heavy weights', 'Scuba diving'],
                ],
                'tip' => ['fa' => 'اگر قبلاً ورزش می‌کردید، می‌توانید با شدت کمتر ادامه دهید. با پزشک مشورت کنید.', 'en' => 'If you were exercising before, you can continue with less intensity. Consult your doctor.'],
                'duration' => '20-30',
                'precautions' => [
                    'fa' => ['گرم‌کردن حتماً', 'آب فراوان', 'اجتناب از گرمازدگی'],
                    'en' => ['Always warm up', 'Plenty of water', 'Avoid overheating'],
                ],
            ],
            2 => [
                'intensity' => ['fa' => 'متوسط', 'en' => 'Moderate'],
                'recommended' => [
                    'fa' => ['شنا', 'پیاده‌روی', 'یوگای بارداری', 'پیلاتس بارداری', 'دوچرخه ثابت'],
                    'en' => ['Swimming', 'Walking', 'Prenatal yoga', 'Prenatal pilates', 'Stationary bike'],
                ],
                'avoid' => [
                    'fa' => ['دراز کشیدن به پشت طولانی', 'حرکات پرشی', 'ورزش در گرما'],
                    'en' => ['Lying on back for long', 'Jumping movements', 'Exercise in heat'],
                ],
                'tip' => ['fa' => 'دوران طلایی! انرژی برگشته. از ورزش لذت ببرید اما محتاط باشید.', 'en' => 'Golden period! Energy is back. Enjoy exercise but be cautious.'],
                'duration' => '30-45',
                'precautions' => [
                    'fa' => ['مرکز ثقل تغییر کرده', 'تعادل ممکنه کم شده باشه'],
                    'en' => ['Center of gravity has changed', 'Balance may be reduced'],
                ],
            ],
            3 => [
                'intensity' => ['fa' => 'کم', 'en' => 'Low'],
                'recommended' => [
                    'fa' => ['پیاده‌روی', 'شنا', 'یوگای ملایم', 'کشش', 'تمرینات تنفسی', 'کگل'],
                    'en' => ['Walking', 'Swimming', 'Gentle yoga', 'Stretching', 'Breathing exercises', 'Kegel'],
                ],
                'avoid' => [
                    'fa' => ['هر چیزی که ریسک افتادن داره', 'ورزش‌های شدید', 'بلند کردن وزنه'],
                    'en' => ['Anything with falling risk', 'Intense exercise', 'Lifting weights'],
                ],
                'tip' => ['fa' => 'تمرکز روی آمادگی زایمان. تمرینات تنفسی و کگل مهم هستند.', 'en' => 'Focus on birth preparation. Breathing exercises and Kegel are important.'],
                'duration' => '20-30',
                'precautions' => [
                    'fa' => ['شکم بزرگ تعادل را تغییر داده', 'مفاصل شل‌تر شده‌اند'],
                    'en' => ['Large belly has changed balance', 'Joints have loosened'],
                ],
            ],
        ];
    }
}
