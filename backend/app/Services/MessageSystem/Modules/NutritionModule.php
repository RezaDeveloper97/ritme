<?php

namespace App\Services\MessageSystem\Modules;

use App\Services\MessageSystem\Contracts\ProvidesMessageContent;
use App\Services\MessageSystem\Core\MessageContext;
use App\Services\MessageSystem\Support\MessageContentRepository;

class NutritionModule implements ProvidesMessageContent
{
    public function __construct(
        private readonly string $locale = 'fa',
    ) {}

    /**
     * Get nutrition tips based on context
     */
    public function getTips(MessageContext $context): array
    {
        if ($context->isPregnancyMode()) {
            return $this->getPregnancyNutrition($context);
        }

        return $this->getCycleNutrition($context);
    }

    private function content(): MessageContentRepository
    {
        return app(MessageContentRepository::class);
    }

    /**
     * Get cycle-based nutrition tips
     */
    private function getCycleNutrition(MessageContext $context): array
    {
        $phase = $context->cyclePhase;
        $isTTC = $context->isTTC();

        $nutrition = $this->getCyclePhaseNutrition($phase);

        // Add TTC-specific recommendations
        if ($isTTC) {
            $nutrition['ttc_additions'] = $this->getTTCNutrition($phase);
        }

        // Override based on symptoms
        if (!empty($context->symptoms)) {
            $symptomNutrition = $this->getSymptomBasedNutrition($context->symptoms);
            if (!empty($symptomNutrition)) {
                $nutrition['symptom_specific'] = $symptomNutrition;
            }
        }

        return $nutrition;
    }

    /**
     * Get pregnancy-based nutrition tips
     */
    private function getPregnancyNutrition(MessageContext $context): array
    {
        $trimester = $context->trimester ?? 1;
        $week = $context->pregnancyWeek;

        return $this->getTrimesterNutrition($trimester, $week);
    }

    /**
     * Get cycle phase nutrition (DB-editable, falls back to defaults)
     */
    private function getCyclePhaseNutrition(?string $phase): array
    {
        $tips = self::cyclePhaseNutritionSource();
        $itemKey = ($phase !== null && array_key_exists($phase, $tips)) ? $phase : 'follicular';
        $fallback = self::slice($tips[$itemKey], $this->locale);

        $p = $this->content()->resolve('nutrition_cycle', $itemKey, $this->locale, $fallback);

        return [
            'focus' => $p['focus'] ?? '',
            'foods' => $p['foods'] ?? [],
            'avoid' => $p['avoid'] ?? [],
            'tip' => $p['tip'] ?? '',
        ];
    }

    /**
     * Get TTC-specific nutrition (DB-editable, falls back to defaults)
     */
    private function getTTCNutrition(?string $phase): array
    {
        $fallback = self::slice(self::ttcNutritionSource(), $this->locale);
        $p = $this->content()->resolve('nutrition_ttc', 'base', $this->locale, $fallback);

        $base = [
            'essential' => $p['essential'] ?? [],
            'foods' => $p['foods'] ?? [],
            'avoid' => $p['avoid'] ?? [],
        ];

        if ($phase === 'ovulation') {
            $base['fertile_window_tip'] = $p['fertile_window_tip'] ?? '';
        }

        return $base;
    }

    /**
     * Get symptom-based nutrition (DB-editable, falls back to defaults)
     */
    private function getSymptomBasedNutrition(array $symptoms): array
    {
        $tips = [];

        if (in_array('cramps', $symptoms)) {
            $tips['for_cramps'] = $this->symptomTip('cramps');
        }

        if (in_array('bloating', $symptoms)) {
            $tips['for_bloating'] = $this->symptomTip('bloating');
        }

        if (in_array('fatigue', $symptoms) || in_array('low_energy', $symptoms)) {
            $tips['for_fatigue'] = $this->symptomTip('fatigue');
        }

        if (in_array('headache', $symptoms)) {
            $tips['for_headache'] = $this->symptomTip('headache');
        }

        return $tips;
    }

    /**
     * Resolve a single symptom nutrition tip (DB-editable, falls back to defaults)
     */
    private function symptomTip(string $key): string
    {
        $entries = self::symptomNutritionSource();
        $fallback = self::slice($entries[$key], $this->locale);
        $p = $this->content()->resolve('nutrition_symptom', $key, $this->locale, $fallback);

        return $p['tip'] ?? '';
    }

    /**
     * Get trimester-based nutrition (DB-editable, falls back to defaults)
     */
    private function getTrimesterNutrition(int $trimester, ?int $week): array
    {
        $tips = self::trimesterNutritionSource();
        $key = array_key_exists($trimester, $tips) ? $trimester : 1;
        $itemKey = (string) $key;
        $fallback = self::slice($tips[$key], $this->locale);

        $p = $this->content()->resolve('nutrition_trimester', $itemKey, $this->locale, $fallback);

        return [
            'focus' => $p['focus'] ?? '',
            'essential' => $p['essential'] ?? [],
            'foods' => $p['foods'] ?? [],
            'avoid' => $p['avoid'] ?? [],
            'tip' => $p['tip'] ?? '',
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
     * @inheritDoc — seed/fallback source for all nutrition message text.
     */
    public static function contentDefaults(): array
    {
        $out = [];
        foreach (self::cyclePhaseNutritionSource() as $key => $entry) {
            foreach (['fa', 'en'] as $loc) {
                $out['nutrition_cycle'][$key][$loc] = self::slice($entry, $loc);
            }
        }
        foreach (['fa', 'en'] as $loc) {
            $out['nutrition_ttc']['base'][$loc] = self::slice(self::ttcNutritionSource(), $loc);
        }
        foreach (self::symptomNutritionSource() as $key => $entry) {
            foreach (['fa', 'en'] as $loc) {
                $out['nutrition_symptom'][$key][$loc] = self::slice($entry, $loc);
            }
        }
        foreach (self::trimesterNutritionSource() as $key => $entry) {
            foreach (['fa', 'en'] as $loc) {
                $out['nutrition_trimester'][(string) $key][$loc] = self::slice($entry, $loc);
            }
        }
        return $out;
    }

    /**
     * Cycle phase nutrition database (seed/fallback)
     */
    private static function cyclePhaseNutritionSource(): array
    {
        return [
            'menstruation' => [
                'focus' => ['fa' => 'جبران آهن و ضدالتهاب', 'en' => 'Iron replenishment and anti-inflammatory'],
                'foods' => [
                    'fa' => ['گوشت قرمز', 'اسفناج', 'عدس', 'ماهی', 'شکلات تلخ'],
                    'en' => ['Red meat', 'Spinach', 'Lentils', 'Fish', 'Dark chocolate'],
                ],
                'avoid' => [
                    'fa' => ['غذاهای شور', 'کافئین زیاد', 'الکل'],
                    'en' => ['Salty foods', 'Too much caffeine', 'Alcohol'],
                ],
                'tip' => ['fa' => 'ویتامین C همراه آهن جذب را افزایش می‌دهد', 'en' => 'Vitamin C with iron increases absorption'],
            ],
            'follicular' => [
                'focus' => ['fa' => 'پروتئین و سبزیجات تازه', 'en' => 'Protein and fresh vegetables'],
                'foods' => [
                    'fa' => ['تخم‌مرغ', 'مرغ', 'کینوا', 'سبزیجات برگ‌سبز', 'آووکادو'],
                    'en' => ['Eggs', 'Chicken', 'Quinoa', 'Leafy greens', 'Avocado'],
                ],
                'avoid' => [
                    'fa' => ['فست‌فود', 'غذاهای فرآوری شده'],
                    'en' => ['Fast food', 'Processed foods'],
                ],
                'tip' => ['fa' => 'بدن آماده رشد است، پروتئین کافی بخور', 'en' => 'Body is ready for growth, eat enough protein'],
            ],
            'ovulation' => [
                'focus' => ['fa' => 'آنتی‌اکسیدان و چربی سالم', 'en' => 'Antioxidants and healthy fats'],
                'foods' => [
                    'fa' => ['توت‌ها', 'آجیل', 'روغن زیتون', 'ماهی سالمون', 'سبزیجات رنگی'],
                    'en' => ['Berries', 'Nuts', 'Olive oil', 'Salmon', 'Colorful vegetables'],
                ],
                'avoid' => [
                    'fa' => ['قند زیاد', 'چربی ترانس'],
                    'en' => ['Too much sugar', 'Trans fats'],
                ],
                'tip' => ['fa' => 'استروژن در اوج است، از آنتی‌اکسیدان‌ها استفاده کن', 'en' => 'Estrogen is at peak, use antioxidants'],
            ],
            'luteal' => [
                'focus' => ['fa' => 'کربوهیدرات پیچیده و منیزیم', 'en' => 'Complex carbs and magnesium'],
                'foods' => [
                    'fa' => ['جو', 'نخود', 'موز', 'بادام', 'شکلات تلخ'],
                    'en' => ['Oats', 'Chickpeas', 'Banana', 'Almonds', 'Dark chocolate'],
                ],
                'avoid' => [
                    'fa' => ['نمک زیاد', 'شکر', 'کافئین'],
                    'en' => ['Too much salt', 'Sugar', 'Caffeine'],
                ],
                'tip' => ['fa' => 'منیزیم به کاهش علائم PMS کمک می‌کند', 'en' => 'Magnesium helps reduce PMS symptoms'],
            ],
        ];
    }

    /**
     * TTC nutrition database (seed/fallback)
     */
    private static function ttcNutritionSource(): array
    {
        return [
            'essential' => [
                'fa' => ['فولیک اسید ۴۰۰mcg روزانه', 'اسیدهای چرب امگا-۳', 'آنتی‌اکسیدان‌ها'],
                'en' => ['Folic acid 400mcg daily', 'Omega-3 fatty acids', 'Antioxidants'],
            ],
            'foods' => [
                'fa' => ['سبزیجات برگ‌سبز تیره', 'ماهی‌های چرب', 'تخم‌مرغ', 'آجیل و دانه‌ها'],
                'en' => ['Dark leafy greens', 'Fatty fish', 'Eggs', 'Nuts and seeds'],
            ],
            'avoid' => [
                'fa' => ['الکل', 'کافئین بیش از ۲۰۰mg', 'ماهی با جیوه بالا'],
                'en' => ['Alcohol', 'Caffeine over 200mg', 'High mercury fish'],
            ],
            'fertile_window_tip' => [
                'fa' => 'هیدراتاسیون کافی برای مخاط دهانه رحم مهم است',
                'en' => 'Adequate hydration is important for cervical mucus',
            ],
        ];
    }

    /**
     * Symptom-based nutrition database (seed/fallback)
     */
    private static function symptomNutritionSource(): array
    {
        return [
            'cramps' => [
                'tip' => [
                    'fa' => 'منیزیم (موز، بادام)، زنجبیل و چای بابونه',
                    'en' => 'Magnesium (banana, almonds), ginger and chamomile tea',
                ],
            ],
            'bloating' => [
                'tip' => [
                    'fa' => 'کاهش نمک، چای نعناع، خیار و هندوانه',
                    'en' => 'Reduce salt, peppermint tea, cucumber and watermelon',
                ],
            ],
            'fatigue' => [
                'tip' => [
                    'fa' => 'آهن (گوشت، اسفناج)، ویتامین B12، آب کافی',
                    'en' => 'Iron (meat, spinach), vitamin B12, enough water',
                ],
            ],
            'headache' => [
                'tip' => [
                    'fa' => 'آب فراوان، منیزیم، اجتناب از قند و کافئین زیاد',
                    'en' => 'Plenty of water, magnesium, avoid sugar and too much caffeine',
                ],
            ],
        ];
    }

    /**
     * Trimester nutrition database (seed/fallback)
     */
    private static function trimesterNutritionSource(): array
    {
        return [
            1 => [
                'focus' => ['fa' => 'فولیک اسید و مدیریت تهوع', 'en' => 'Folic acid and nausea management'],
                'essential' => [
                    'fa' => ['فولیک اسید ۶۰۰mcg', 'ویتامین D', 'آهن'],
                    'en' => ['Folic acid 600mcg', 'Vitamin D', 'Iron'],
                ],
                'foods' => [
                    'fa' => ['غذاهای کوچک و مکرر', 'کراکر خشک', 'زنجبیل', 'لیمو', 'پروتئین'],
                    'en' => ['Small frequent meals', 'Dry crackers', 'Ginger', 'Lemon', 'Protein'],
                ],
                'avoid' => [
                    'fa' => ['ماهی خام', 'گوشت نپخته', 'پنیرهای نرم', 'الکل', 'کافئین زیاد'],
                    'en' => ['Raw fish', 'Undercooked meat', 'Soft cheeses', 'Alcohol', 'Too much caffeine'],
                ],
                'tip' => ['fa' => 'برای تهوع، قبل از بلند شدن از تخت کراکر بخورید', 'en' => 'For nausea, eat crackers before getting out of bed'],
            ],
            2 => [
                'focus' => ['fa' => 'کلسیم، آهن و پروتئین', 'en' => 'Calcium, iron and protein'],
                'essential' => [
                    'fa' => ['کلسیم ۱۰۰۰mg', 'آهن ۲۷mg', 'پروتئین ۷۵g'],
                    'en' => ['Calcium 1000mg', 'Iron 27mg', 'Protein 75g'],
                ],
                'foods' => [
                    'fa' => ['لبنیات', 'گوشت قرمز', 'مرغ', 'ماهی', 'حبوبات', 'میوه و سبزی'],
                    'en' => ['Dairy', 'Red meat', 'Chicken', 'Fish', 'Legumes', 'Fruits and vegetables'],
                ],
                'avoid' => [
                    'fa' => ['غذاهای فرآوری شده', 'قند زیاد', 'نمک زیاد'],
                    'en' => ['Processed foods', 'Too much sugar', 'Too much salt'],
                ],
                'tip' => ['fa' => 'نوزاد در حال رشد سریع است، کالری و پروتئین بیشتر نیاز دارید', 'en' => 'Baby is growing fast, you need more calories and protein'],
            ],
            3 => [
                'focus' => ['fa' => 'امگا-۳ و آماده‌سازی شیردهی', 'en' => 'Omega-3 and breastfeeding prep'],
                'essential' => [
                    'fa' => ['امگا-۳ (DHA)', 'کلسیم', 'ویتامین K'],
                    'en' => ['Omega-3 (DHA)', 'Calcium', 'Vitamin K'],
                ],
                'foods' => [
                    'fa' => ['ماهی سالمون', 'گردو', 'سبزیجات برگ‌سبز', 'تخم‌مرغ'],
                    'en' => ['Salmon', 'Walnuts', 'Leafy greens', 'Eggs'],
                ],
                'avoid' => [
                    'fa' => ['غذاهای زیاد حجم', 'نمک زیاد (ورم)'],
                    'en' => ['Large volume meals', 'Too much salt (swelling)'],
                ],
                'tip' => ['fa' => 'غذاهای کوچک‌تر و مکرر برای فضای کم معده', 'en' => 'Smaller and more frequent meals for limited stomach space'],
            ],
        ];
    }
}
