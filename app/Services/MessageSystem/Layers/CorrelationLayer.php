<?php

namespace App\Services\MessageSystem\Layers;

use App\Models\User;
use App\Services\MessageSystem\Core\MessageContext;

/**
 * Layer 3: Correlation Engine
 * Detects relationships between symptoms and provides insights
 */
class CorrelationLayer
{
    public function __construct(
        private readonly User $user,
        private readonly string $locale = 'fa',
    ) {}

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
            $correlations[] = [
                'type' => 'sleep_mood',
                'insight_message' => $this->locale === 'fa'
                    ? 'کیفیت خواب پایین می‌تونه روی خلق و خو تأثیر بذاره. بهبود خواب ممکنه به بهتر شدن حالت روحی کمک کنه.'
                    : 'Poor sleep quality can affect mood. Improving sleep may help improve mood.',
                'action' => $this->locale === 'fa'
                    ? 'سعی کن امشب زودتر بخوابی و از گوشی قبل خواب استفاده نکنی'
                    : 'Try to sleep earlier tonight and avoid phone before bed',
                'is_premium_only' => false,
            ];
        }

        // Stress + Physical symptoms correlation
        if (in_array('mood_anxious', $symptoms) && (in_array('headache', $symptoms) || in_array('cramps', $symptoms))) {
            $correlations[] = [
                'type' => 'stress_physical',
                'insight_message' => $this->locale === 'fa'
                    ? 'استرس می‌تونه علائم فیزیکی مثل سردرد و کرامپ رو تشدید کنه.'
                    : 'Stress can intensify physical symptoms like headache and cramps.',
                'action' => $this->locale === 'fa'
                    ? 'تکنیک‌های آرامش‌بخش مثل تنفس عمیق رو امتحان کن'
                    : 'Try relaxation techniques like deep breathing',
                'is_premium_only' => false,
            ];
        }

        // PMS window correlations
        if ($context->isPmsWindow) {
            if (in_array('bloating', $symptoms) || in_array('mood_sad', $symptoms)) {
                $correlations[] = [
                    'type' => 'pms_symptoms',
                    'insight_message' => $this->locale === 'fa'
                        ? 'علائم PMS در این دوره طبیعی هستن. چند روز دیگه بهتر میشی.'
                        : 'PMS symptoms are normal in this period. You\'ll feel better in a few days.',
                    'action' => $this->locale === 'fa'
                        ? 'مصرف نمک رو کم کن و ورزش سبک انجام بده'
                        : 'Reduce salt intake and do light exercise',
                    'is_premium_only' => false,
                ];
            }
        }

        // Fertile window + Energy (TTC)
        if ($context->isTTC() && $context->isFertileWindow) {
            $correlations[] = [
                'type' => 'fertile_window_ttc',
                'insight_message' => $this->locale === 'fa'
                    ? 'این بهترین زمان برای باردار شدن است. استروژن در اوج و شانس لقاح بالاست.'
                    : 'This is the best time to conceive. Estrogen is at peak and conception chances are high.',
                'action' => $this->locale === 'fa'
                    ? 'رابطه هر ۲۴-۴۸ ساعت توصیه می‌شه'
                    : 'Intercourse every 24-48 hours is recommended',
                'is_premium_only' => false,
            ];
        }

        // Premium: Detailed phase-symptom correlation
        if ($phase === 'luteal' && in_array('low_energy', $symptoms)) {
            $correlations[] = [
                'type' => 'luteal_energy_drop',
                'insight_message' => $this->locale === 'fa'
                    ? 'کاهش انرژی در فاز لوتئال به دلیل افزایش پروژسترون است. این کاملاً طبیعیه.'
                    : 'Energy drop in luteal phase is due to rising progesterone. This is completely normal.',
                'action' => $this->locale === 'fa'
                    ? 'به بدنت گوش بده و استراحت کافی داشته باش'
                    : 'Listen to your body and get adequate rest',
                'is_premium_only' => true,
            ];
        }

        // Premium: Hormone-skin correlation
        if (in_array('acne', $symptoms) && ($phase === 'luteal' || $phase === 'menstruation')) {
            $correlations[] = [
                'type' => 'hormone_skin',
                'insight_message' => $this->locale === 'fa'
                    ? 'جوش‌های هورمونی در این فاز شایع هستن. با شروع فاز فولیکولار بهتر میشه.'
                    : 'Hormonal acne is common in this phase. It will improve with follicular phase.',
                'action' => $this->locale === 'fa'
                    ? 'پاکسازی ملایم و اجتناب از دست زدن به صورت'
                    : 'Gentle cleansing and avoid touching face',
                'is_premium_only' => true,
            ];
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
            $correlations[] = [
                'type' => 'first_trimester_combo',
                'insight_message' => $this->locale === 'fa'
                    ? 'ترکیب تهوع و خستگی در سه‌ماهه اول بسیار شایع است و نشانه تغییرات هورمونی قوی است.'
                    : 'Combination of nausea and fatigue is very common in first trimester, showing strong hormonal changes.',
                'action' => $this->locale === 'fa'
                    ? 'غذاهای کوچک و مکرر و استراحت کوتاه در طول روز'
                    : 'Small frequent meals and short rests during day',
                'is_premium_only' => false,
            ];
        }

        // Third trimester discomfort
        if ($trimester === 3 && (in_array('backache', $symptoms) || in_array('fatigue', $symptoms))) {
            $correlations[] = [
                'type' => 'third_trimester_discomfort',
                'insight_message' => $this->locale === 'fa'
                    ? 'ناراحتی‌های سه‌ماهه سوم به دلیل رشد نوزاد و آماده‌سازی بدن برای زایمان است.'
                    : 'Third trimester discomforts are due to baby growth and body preparing for delivery.',
                'action' => $this->locale === 'fa'
                    ? 'شنا، یوگای بارداری و استراحت با پاهای بالا'
                    : 'Swimming, prenatal yoga and resting with elevated legs',
                'is_premium_only' => false,
            ];
        }

        // Premium: Mood changes in pregnancy
        if (in_array('mood_sad', $symptoms) || in_array('mood_anxious', $symptoms)) {
            $correlations[] = [
                'type' => 'pregnancy_mood',
                'insight_message' => $this->locale === 'fa'
                    ? 'تغییرات خلقی در بارداری طبیعی هستن اما اگر ادامه‌دار باشن با پزشک صحبت کنید.'
                    : 'Mood changes in pregnancy are normal but if persistent, talk to your doctor.',
                'action' => $this->locale === 'fa'
                    ? 'با همسر یا دوست صحبت کن و در صورت نیاز با پزشک مشورت کن'
                    : 'Talk to partner or friend and consult doctor if needed',
                'is_premium_only' => true,
            ];
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
            $correlations[] = [
                'type' => 'dehydration_indicator',
                'insight_message' => $this->locale === 'fa'
                    ? 'ترکیب سردرد و خستگی می‌تونه نشانه کم‌آبی باشه.'
                    : 'Combination of headache and fatigue could indicate dehydration.',
                'action' => $this->locale === 'fa'
                    ? 'حداقل ۸ لیوان آب در روز بنوش'
                    : 'Drink at least 8 glasses of water daily',
                'is_premium_only' => false,
            ];
        }

        // Sleep deprivation pattern
        if (in_array('poor_sleep', $symptoms) && in_array('low_energy', $symptoms)) {
            $correlations[] = [
                'type' => 'sleep_deprivation',
                'insight_message' => $this->locale === 'fa'
                    ? 'کم‌خوابی مستقیماً روی انرژی روزانه تأثیر می‌گذاره.'
                    : 'Sleep deprivation directly affects daily energy.',
                'action' => $this->locale === 'fa'
                    ? 'یک برنامه خواب منظم تنظیم کن'
                    : 'Set a regular sleep schedule',
                'is_premium_only' => false,
            ];
        }

        return $correlations;
    }
}
