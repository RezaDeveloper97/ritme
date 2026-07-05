<?php

namespace App\Services\MessageSystem\Modules;

use App\Services\MessageSystem\Core\MessageContext;

class SleepModule
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
                $tips['symptom_tip'] = $this->locale === 'fa'
                    ? 'کیفیت خواب پایین ثبت شده. بهداشت خواب را رعایت کنید.'
                    : 'Poor sleep quality logged. Practice sleep hygiene.';
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
     * Get cycle phase sleep recommendations
     */
    private function getCyclePhaseSleep(?string $phase): array
    {
        $tips = [
            'menstruation' => [
                'recommended_hours' => '8-9',
                'quality_focus' => $this->locale === 'fa' ? 'استراحت بیشتر' : 'More rest',
                'tips' => $this->locale === 'fa'
                    ? ['خواب بیشتر نیاز است', 'کمپرس گرم برای کرامپ', 'اتاق خنک']
                    : ['More sleep is needed', 'Warm compress for cramps', 'Cool room'],
                'avoid' => $this->locale === 'fa'
                    ? ['کافئین بعد از ظهر', 'گوشی قبل خواب']
                    : ['Caffeine after noon', 'Phone before bed'],
            ],
            'follicular' => [
                'recommended_hours' => '7-8',
                'quality_focus' => $this->locale === 'fa' ? 'خواب بهینه' : 'Optimal sleep',
                'tips' => $this->locale === 'fa'
                    ? ['خواب عمیق‌تر در این فاز', 'زمان مناسب برای تنظیم ساعت خواب']
                    : ['Deeper sleep in this phase', 'Good time to set sleep schedule'],
                'avoid' => $this->locale === 'fa'
                    ? ['بی‌خوابی', 'ساعت خواب نامنظم']
                    : ['Sleep deprivation', 'Irregular sleep schedule'],
            ],
            'ovulation' => [
                'recommended_hours' => '7-8',
                'quality_focus' => $this->locale === 'fa' ? 'انرژی بالا' : 'High energy',
                'tips' => $this->locale === 'fa'
                    ? ['ممکنه کمتر خسته باشی', 'از انرژی استفاده کن ولی خواب رو فراموش نکن']
                    : ['You may feel less tired', 'Use the energy but don\'t forget sleep'],
                'avoid' => $this->locale === 'fa'
                    ? ['بیدار موندن دیروقت', 'از دست دادن خواب']
                    : ['Staying up late', 'Missing sleep'],
            ],
            'luteal' => [
                'recommended_hours' => '8-9',
                'quality_focus' => $this->locale === 'fa' ? 'کیفیت خواب ممکنه کم بشه' : 'Sleep quality may decrease',
                'tips' => $this->locale === 'fa'
                    ? ['ممکنه بی‌خوابی داشته باشی', 'منیزیم قبل خواب', 'اتاق تاریک و خنک', 'مدیتیشن قبل خواب']
                    : ['You may have insomnia', 'Magnesium before bed', 'Dark and cool room', 'Meditation before bed'],
                'avoid' => $this->locale === 'fa'
                    ? ['صفحه نمایش قبل خواب', 'غذای سنگین شب', 'کافئین']
                    : ['Screen before bed', 'Heavy dinner', 'Caffeine'],
            ],
        ];

        return $tips[$phase] ?? $tips['follicular'];
    }

    /**
     * Get trimester sleep recommendations
     */
    private function getTrimesterSleep(int $trimester): array
    {
        $tips = [
            1 => [
                'recommended_hours' => '8-10',
                'quality_focus' => $this->locale === 'fa' ? 'خستگی شدید طبیعی است' : 'Extreme fatigue is normal',
                'tips' => $this->locale === 'fa'
                    ? ['چرت‌های کوتاه در طول روز', 'زود بخواب', 'به بدنت گوش بده', 'هورمون‌ها باعث خواب‌آلودگی می‌شن']
                    : ['Short naps during day', 'Sleep early', 'Listen to your body', 'Hormones cause sleepiness'],
                'avoid' => $this->locale === 'fa'
                    ? ['مقاومت در برابر خواب', 'کار زیاد']
                    : ['Resisting sleep', 'Overwork'],
                'position' => $this->locale === 'fa'
                    ? 'هنوز هر پوزیشنی مناسب است'
                    : 'Any position is still suitable',
            ],
            2 => [
                'recommended_hours' => '8-9',
                'quality_focus' => $this->locale === 'fa' ? 'انرژی برگشته' : 'Energy is back',
                'tips' => $this->locale === 'fa'
                    ? ['خواب بهتر از سه‌ماهه اول', 'شروع عادت به خوابیدن به پهلو', 'بالش بین زانوها']
                    : ['Better sleep than first trimester', 'Start getting used to side sleeping', 'Pillow between knees'],
                'avoid' => $this->locale === 'fa'
                    ? ['خوابیدن به پشت طولانی']
                    : ['Sleeping on back for long'],
                'position' => $this->locale === 'fa'
                    ? 'پهلوی چپ بهترین است'
                    : 'Left side is best',
            ],
            3 => [
                'recommended_hours' => '8-9',
                'quality_focus' => $this->locale === 'fa' ? 'خواب راحت سخت می‌شه' : 'Comfortable sleep becomes harder',
                'tips' => $this->locale === 'fa'
                    ? ['بالش بارداری', 'سر تخت بالاتر', 'ادرار قبل خواب', 'تنفس عمیق', 'نوشیدنی کم قبل خواب']
                    : ['Pregnancy pillow', 'Elevated head', 'Urinate before sleep', 'Deep breathing', 'Less fluids before bed'],
                'avoid' => $this->locale === 'fa'
                    ? ['خوابیدن به پشت', 'مایعات زیاد قبل خواب']
                    : ['Sleeping on back', 'Too much fluids before bed'],
                'position' => $this->locale === 'fa'
                    ? 'حتماً به پهلو بخوابید، ترجیحاً چپ'
                    : 'Must sleep on side, preferably left',
            ],
        ];

        return $tips[$trimester] ?? $tips[1];
    }
}
