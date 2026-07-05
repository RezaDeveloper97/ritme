<?php

namespace App\Services\MessageSystem\Modules;

use App\Services\MessageSystem\Core\MessageContext;

class ExerciseModule
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
                $tips['symptom_modification'] = $this->locale === 'fa'
                    ? 'با توجه به درد، ورزش‌های کششی ملایم توصیه می‌شود'
                    : 'Due to pain, gentle stretching exercises are recommended';
            }
            if (in_array('low_energy', $context->symptoms) || in_array('fatigue', $context->symptoms)) {
                $tips['symptom_modification'] = $this->locale === 'fa'
                    ? 'با توجه به خستگی، پیاده‌روی سبک کافی است'
                    : 'Due to fatigue, light walking is enough';
            }
        }

        // TTC specific
        if ($context->isTTC()) {
            $tips['ttc_note'] = $this->locale === 'fa'
                ? 'ورزش متوسط برای باروری مفید است. از ورزش‌های خیلی شدید اجتناب کنید.'
                : 'Moderate exercise is good for fertility. Avoid very intense workouts.';
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
     * Get cycle phase exercise recommendations
     */
    private function getCyclePhaseExercise(?string $phase): array
    {
        $tips = [
            'menstruation' => [
                'intensity' => $this->locale === 'fa' ? 'کم تا متوسط' : 'Low to moderate',
                'recommended' => $this->locale === 'fa'
                    ? ['پیاده‌روی', 'یوگای ریستوراتیو', 'کشش ملایم', 'شنا']
                    : ['Walking', 'Restorative yoga', 'Gentle stretching', 'Swimming'],
                'avoid' => $this->locale === 'fa'
                    ? ['HIIT شدید', 'وزنه سنگین', 'وارونگی در یوگا']
                    : ['Intense HIIT', 'Heavy weights', 'Inversions in yoga'],
                'tip' => $this->locale === 'fa'
                    ? 'به بدنت گوش بده. اگر انرژی نداری، استراحت کن.'
                    : 'Listen to your body. If you have no energy, rest.',
                'duration' => '20-30',
            ],
            'follicular' => [
                'intensity' => $this->locale === 'fa' ? 'متوسط تا بالا' : 'Moderate to high',
                'recommended' => $this->locale === 'fa'
                    ? ['دویدن', 'HIIT', 'وزنه‌برداری', 'دوچرخه', 'رقص']
                    : ['Running', 'HIIT', 'Weight training', 'Cycling', 'Dancing'],
                'avoid' => $this->locale === 'fa'
                    ? ['بی‌فعالیتی']
                    : ['Inactivity'],
                'tip' => $this->locale === 'fa'
                    ? 'انرژی در اوج است! بهترین زمان برای چالش‌های جدید و رکوردشکنی.'
                    : 'Energy is at peak! Best time for new challenges and personal records.',
                'duration' => '45-60',
            ],
            'ovulation' => [
                'intensity' => $this->locale === 'fa' ? 'بالا' : 'High',
                'recommended' => $this->locale === 'fa'
                    ? ['تمرینات قدرتی', 'HIIT', 'ورزش‌های گروهی', 'کراس‌فیت']
                    : ['Strength training', 'HIIT', 'Group sports', 'CrossFit'],
                'avoid' => $this->locale === 'fa'
                    ? ['هیچی! همه چیز خوبه']
                    : ['Nothing! Everything is good'],
                'tip' => $this->locale === 'fa'
                    ? 'استروژن بالاست، عضلات قوی‌تر و ریکاوری سریع‌تر.'
                    : 'Estrogen is high, muscles are stronger and recovery is faster.',
                'duration' => '45-60',
            ],
            'luteal' => [
                'intensity' => $this->locale === 'fa' ? 'متوسط (کم می‌شه)' : 'Moderate (decreasing)',
                'recommended' => $this->locale === 'fa'
                    ? ['پیلاتس', 'یوگا', 'پیاده‌روی', 'وزنه سبک', 'شنا']
                    : ['Pilates', 'Yoga', 'Walking', 'Light weights', 'Swimming'],
                'avoid' => $this->locale === 'fa'
                    ? ['ورزش‌های خیلی شدید', 'تمرینات طولانی']
                    : ['Very intense workouts', 'Long training sessions'],
                'tip' => $this->locale === 'fa'
                    ? 'انرژی کم‌کم کم می‌شه. ورزش ملایم به کاهش علائم PMS کمک می‌کنه.'
                    : 'Energy is gradually decreasing. Light exercise helps reduce PMS symptoms.',
                'duration' => '30-45',
            ],
        ];

        return $tips[$phase] ?? $tips['follicular'];
    }

    /**
     * Get trimester exercise recommendations
     */
    private function getTrimesterExercise(int $trimester): array
    {
        $tips = [
            1 => [
                'intensity' => $this->locale === 'fa' ? 'کم تا متوسط' : 'Low to moderate',
                'recommended' => $this->locale === 'fa'
                    ? ['پیاده‌روی', 'شنا', 'یوگای بارداری', 'تمرینات کگل']
                    : ['Walking', 'Swimming', 'Prenatal yoga', 'Kegel exercises'],
                'avoid' => $this->locale === 'fa'
                    ? ['ورزش‌های پرخطر', 'ورزش‌های تماسی', 'وزنه سنگین', 'غوص']
                    : ['High-risk sports', 'Contact sports', 'Heavy weights', 'Scuba diving'],
                'tip' => $this->locale === 'fa'
                    ? 'اگر قبلاً ورزش می‌کردید، می‌توانید با شدت کمتر ادامه دهید. با پزشک مشورت کنید.'
                    : 'If you were exercising before, you can continue with less intensity. Consult your doctor.',
                'duration' => '20-30',
                'precautions' => $this->locale === 'fa'
                    ? ['گرم‌کردن حتماً', 'آب فراوان', 'اجتناب از گرمازدگی']
                    : ['Always warm up', 'Plenty of water', 'Avoid overheating'],
            ],
            2 => [
                'intensity' => $this->locale === 'fa' ? 'متوسط' : 'Moderate',
                'recommended' => $this->locale === 'fa'
                    ? ['شنا', 'پیاده‌روی', 'یوگای بارداری', 'پیلاتس بارداری', 'دوچرخه ثابت']
                    : ['Swimming', 'Walking', 'Prenatal yoga', 'Prenatal pilates', 'Stationary bike'],
                'avoid' => $this->locale === 'fa'
                    ? ['دراز کشیدن به پشت طولانی', 'حرکات پرشی', 'ورزش در گرما']
                    : ['Lying on back for long', 'Jumping movements', 'Exercise in heat'],
                'tip' => $this->locale === 'fa'
                    ? 'دوران طلایی! انرژی برگشته. از ورزش لذت ببرید اما محتاط باشید.'
                    : 'Golden period! Energy is back. Enjoy exercise but be cautious.',
                'duration' => '30-45',
                'precautions' => $this->locale === 'fa'
                    ? ['مرکز ثقل تغییر کرده', 'تعادل ممکنه کم شده باشه']
                    : ['Center of gravity has changed', 'Balance may be reduced'],
            ],
            3 => [
                'intensity' => $this->locale === 'fa' ? 'کم' : 'Low',
                'recommended' => $this->locale === 'fa'
                    ? ['پیاده‌روی', 'شنا', 'یوگای ملایم', 'کشش', 'تمرینات تنفسی', 'کگل']
                    : ['Walking', 'Swimming', 'Gentle yoga', 'Stretching', 'Breathing exercises', 'Kegel'],
                'avoid' => $this->locale === 'fa'
                    ? ['هر چیزی که ریسک افتادن داره', 'ورزش‌های شدید', 'بلند کردن وزنه']
                    : ['Anything with falling risk', 'Intense exercise', 'Lifting weights'],
                'tip' => $this->locale === 'fa'
                    ? 'تمرکز روی آمادگی زایمان. تمرینات تنفسی و کگل مهم هستند.'
                    : 'Focus on birth preparation. Breathing exercises and Kegel are important.',
                'duration' => '20-30',
                'precautions' => $this->locale === 'fa'
                    ? ['شکم بزرگ تعادل را تغییر داده', 'مفاصل شل‌تر شده‌اند']
                    : ['Large belly has changed balance', 'Joints have loosened'],
            ],
        ];

        return $tips[$trimester] ?? $tips[1];
    }
}
