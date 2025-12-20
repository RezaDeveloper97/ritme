<?php

namespace App\Services\MessageSystem\Modules;

use App\Services\MessageSystem\Core\MessageContext;

class NutritionModule
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
     * Get cycle phase nutrition
     */
    private function getCyclePhaseNutrition(?string $phase): array
    {
        $tips = [
            'menstruation' => [
                'focus' => $this->locale === 'fa' ? 'جبران آهن و ضدالتهاب' : 'Iron replenishment and anti-inflammatory',
                'foods' => $this->locale === 'fa'
                    ? ['گوشت قرمز', 'اسفناج', 'عدس', 'ماهی', 'شکلات تلخ']
                    : ['Red meat', 'Spinach', 'Lentils', 'Fish', 'Dark chocolate'],
                'avoid' => $this->locale === 'fa'
                    ? ['غذاهای شور', 'کافئین زیاد', 'الکل']
                    : ['Salty foods', 'Too much caffeine', 'Alcohol'],
                'tip' => $this->locale === 'fa'
                    ? 'ویتامین C همراه آهن جذب را افزایش می‌دهد'
                    : 'Vitamin C with iron increases absorption',
            ],
            'follicular' => [
                'focus' => $this->locale === 'fa' ? 'پروتئین و سبزیجات تازه' : 'Protein and fresh vegetables',
                'foods' => $this->locale === 'fa'
                    ? ['تخم‌مرغ', 'مرغ', 'کینوا', 'سبزیجات برگ‌سبز', 'آووکادو']
                    : ['Eggs', 'Chicken', 'Quinoa', 'Leafy greens', 'Avocado'],
                'avoid' => $this->locale === 'fa'
                    ? ['فست‌فود', 'غذاهای فرآوری شده']
                    : ['Fast food', 'Processed foods'],
                'tip' => $this->locale === 'fa'
                    ? 'بدن آماده رشد است، پروتئین کافی بخور'
                    : 'Body is ready for growth, eat enough protein',
            ],
            'ovulation' => [
                'focus' => $this->locale === 'fa' ? 'آنتی‌اکسیدان و چربی سالم' : 'Antioxidants and healthy fats',
                'foods' => $this->locale === 'fa'
                    ? ['توت‌ها', 'آجیل', 'روغن زیتون', 'ماهی سالمون', 'سبزیجات رنگی']
                    : ['Berries', 'Nuts', 'Olive oil', 'Salmon', 'Colorful vegetables'],
                'avoid' => $this->locale === 'fa'
                    ? ['قند زیاد', 'چربی ترانس']
                    : ['Too much sugar', 'Trans fats'],
                'tip' => $this->locale === 'fa'
                    ? 'استروژن در اوج است، از آنتی‌اکسیدان‌ها استفاده کن'
                    : 'Estrogen is at peak, use antioxidants',
            ],
            'luteal' => [
                'focus' => $this->locale === 'fa' ? 'کربوهیدرات پیچیده و منیزیم' : 'Complex carbs and magnesium',
                'foods' => $this->locale === 'fa'
                    ? ['جو', 'نخود', 'موز', 'بادام', 'شکلات تلخ']
                    : ['Oats', 'Chickpeas', 'Banana', 'Almonds', 'Dark chocolate'],
                'avoid' => $this->locale === 'fa'
                    ? ['نمک زیاد', 'شکر', 'کافئین']
                    : ['Too much salt', 'Sugar', 'Caffeine'],
                'tip' => $this->locale === 'fa'
                    ? 'منیزیم به کاهش علائم PMS کمک می‌کند'
                    : 'Magnesium helps reduce PMS symptoms',
            ],
        ];

        return $tips[$phase] ?? $tips['follicular'];
    }

    /**
     * Get TTC-specific nutrition
     */
    private function getTTCNutrition(?string $phase): array
    {
        $base = [
            'essential' => $this->locale === 'fa'
                ? ['فولیک اسید ۴۰۰mcg روزانه', 'اسیدهای چرب امگا-۳', 'آنتی‌اکسیدان‌ها']
                : ['Folic acid 400mcg daily', 'Omega-3 fatty acids', 'Antioxidants'],
            'foods' => $this->locale === 'fa'
                ? ['سبزیجات برگ‌سبز تیره', 'ماهی‌های چرب', 'تخم‌مرغ', 'آجیل و دانه‌ها']
                : ['Dark leafy greens', 'Fatty fish', 'Eggs', 'Nuts and seeds'],
            'avoid' => $this->locale === 'fa'
                ? ['الکل', 'کافئین بیش از ۲۰۰mg', 'ماهی با جیوه بالا']
                : ['Alcohol', 'Caffeine over 200mg', 'High mercury fish'],
        ];

        if ($phase === 'ovulation') {
            $base['fertile_window_tip'] = $this->locale === 'fa'
                ? 'هیدراتاسیون کافی برای مخاط دهانه رحم مهم است'
                : 'Adequate hydration is important for cervical mucus';
        }

        return $base;
    }

    /**
     * Get symptom-based nutrition
     */
    private function getSymptomBasedNutrition(array $symptoms): array
    {
        $tips = [];

        if (in_array('cramps', $symptoms)) {
            $tips['for_cramps'] = $this->locale === 'fa'
                ? 'منیزیم (موز، بادام)، زنجبیل و چای بابونه'
                : 'Magnesium (banana, almonds), ginger and chamomile tea';
        }

        if (in_array('bloating', $symptoms)) {
            $tips['for_bloating'] = $this->locale === 'fa'
                ? 'کاهش نمک، چای نعناع، خیار و هندوانه'
                : 'Reduce salt, peppermint tea, cucumber and watermelon';
        }

        if (in_array('fatigue', $symptoms) || in_array('low_energy', $symptoms)) {
            $tips['for_fatigue'] = $this->locale === 'fa'
                ? 'آهن (گوشت، اسفناج)، ویتامین B12، آب کافی'
                : 'Iron (meat, spinach), vitamin B12, enough water';
        }

        if (in_array('headache', $symptoms)) {
            $tips['for_headache'] = $this->locale === 'fa'
                ? 'آب فراوان، منیزیم، اجتناب از قند و کافئین زیاد'
                : 'Plenty of water, magnesium, avoid sugar and too much caffeine';
        }

        return $tips;
    }

    /**
     * Get trimester-based nutrition
     */
    private function getTrimesterNutrition(int $trimester, ?int $week): array
    {
        $tips = [
            1 => [
                'focus' => $this->locale === 'fa' ? 'فولیک اسید و مدیریت تهوع' : 'Folic acid and nausea management',
                'essential' => $this->locale === 'fa'
                    ? ['فولیک اسید ۶۰۰mcg', 'ویتامین D', 'آهن']
                    : ['Folic acid 600mcg', 'Vitamin D', 'Iron'],
                'foods' => $this->locale === 'fa'
                    ? ['غذاهای کوچک و مکرر', 'کراکر خشک', 'زنجبیل', 'لیمو', 'پروتئین']
                    : ['Small frequent meals', 'Dry crackers', 'Ginger', 'Lemon', 'Protein'],
                'avoid' => $this->locale === 'fa'
                    ? ['ماهی خام', 'گوشت نپخته', 'پنیرهای نرم', 'الکل', 'کافئین زیاد']
                    : ['Raw fish', 'Undercooked meat', 'Soft cheeses', 'Alcohol', 'Too much caffeine'],
                'tip' => $this->locale === 'fa'
                    ? 'برای تهوع، قبل از بلند شدن از تخت کراکر بخورید'
                    : 'For nausea, eat crackers before getting out of bed',
            ],
            2 => [
                'focus' => $this->locale === 'fa' ? 'کلسیم، آهن و پروتئین' : 'Calcium, iron and protein',
                'essential' => $this->locale === 'fa'
                    ? ['کلسیم ۱۰۰۰mg', 'آهن ۲۷mg', 'پروتئین ۷۵g']
                    : ['Calcium 1000mg', 'Iron 27mg', 'Protein 75g'],
                'foods' => $this->locale === 'fa'
                    ? ['لبنیات', 'گوشت قرمز', 'مرغ', 'ماهی', 'حبوبات', 'میوه و سبزی']
                    : ['Dairy', 'Red meat', 'Chicken', 'Fish', 'Legumes', 'Fruits and vegetables'],
                'avoid' => $this->locale === 'fa'
                    ? ['غذاهای فرآوری شده', 'قند زیاد', 'نمک زیاد']
                    : ['Processed foods', 'Too much sugar', 'Too much salt'],
                'tip' => $this->locale === 'fa'
                    ? 'نوزاد در حال رشد سریع است، کالری و پروتئین بیشتر نیاز دارید'
                    : 'Baby is growing fast, you need more calories and protein',
            ],
            3 => [
                'focus' => $this->locale === 'fa' ? 'امگا-۳ و آماده‌سازی شیردهی' : 'Omega-3 and breastfeeding prep',
                'essential' => $this->locale === 'fa'
                    ? ['امگا-۳ (DHA)', 'کلسیم', 'ویتامین K']
                    : ['Omega-3 (DHA)', 'Calcium', 'Vitamin K'],
                'foods' => $this->locale === 'fa'
                    ? ['ماهی سالمون', 'گردو', 'سبزیجات برگ‌سبز', 'تخم‌مرغ']
                    : ['Salmon', 'Walnuts', 'Leafy greens', 'Eggs'],
                'avoid' => $this->locale === 'fa'
                    ? ['غذاهای زیاد حجم', 'نمک زیاد (ورم)']
                    : ['Large volume meals', 'Too much salt (swelling)'],
                'tip' => $this->locale === 'fa'
                    ? 'غذاهای کوچک‌تر و مکرر برای فضای کم معده'
                    : 'Smaller and more frequent meals for limited stomach space',
            ],
        ];

        return $tips[$trimester] ?? $tips[1];
    }
}
