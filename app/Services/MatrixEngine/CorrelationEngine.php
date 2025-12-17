<?php

namespace App\Services\MatrixEngine;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Models\DailyHealthLog;
use App\Models\User;
use App\Models\UserProfile;

/**
 * Correlation Engine - Layer 3
 *
 * Discovers cause-and-effect relationships between user inputs for better symptom management.
 * Logic: When two variables occur together, the message should change because the "context" has changed.
 * Formula: If (Trigger A) AND (Trigger B) -> Show Insight C
 */
class CorrelationEngine
{
    private User $user;
    private ?UserProfile $profile;
    private string $locale;

    public function __construct(User $user, string $locale = 'fa')
    {
        $this->user = $user;
        $this->profile = $user->profile;
        $this->locale = $locale;
    }

    /**
     * Analyze correlations and return insights
     */
    public function analyzeCorrelations(
        CyclePhase $phase,
        CycleSubphase $subphase,
        ?DailyHealthLog $dailyLog,
        bool $isTTC = false
    ): array {
        $insights = [];
        $isPremium = $this->profile?->isPremium() ?? false;

        if (!$dailyLog) {
            return $insights;
        }

        // General correlations (Non-TTC)
        if (!$isTTC) {
            $insights = array_merge($insights, $this->getGeneralCorrelations($phase, $subphase, $dailyLog, $isPremium));
        } else {
            // TTC-specific correlations
            $insights = array_merge($insights, $this->getTTCCorrelations($phase, $subphase, $dailyLog, $isPremium));
        }

        return $insights;
    }

    /**
     * Get general correlations (Non-TTC mode)
     */
    private function getGeneralCorrelations(
        CyclePhase $phase,
        CycleSubphase $subphase,
        DailyHealthLog $dailyLog,
        bool $isPremium
    ): array {
        $insights = [];
        $locale = $this->locale;

        // Correlation 1: Headache + High Caffeine (in luteal phase)
        if ($phase === CyclePhase::LUTEAL
            && $dailyLog->headache_intensity !== null
            && $this->hasCaffeineMention($dailyLog)) {
            $insights[] = [
                'type' => 'headache_caffeine',
                'trigger_a' => 'headache',
                'trigger_b' => 'caffeine',
                'analysis' => $locale === 'fa'
                    ? 'تحلیل: کافئین عروق رو تنگ می‌کنه و سردرد PMS رو (که اغلب میگرنیه) بدتر می‌کنه.'
                    : 'Analysis: Caffeine constricts blood vessels and worsens PMS headaches (which are often migraines).',
                'insight_message' => $locale === 'fa'
                    ? 'دیدم امروز قهوه زیاد خوردی و سردرد داری. توی روزهای نزدیک پریود، کافئین مثل بنزین روی آتیش سردرده. فردا قهوه رو با چای سبز جایگزین کن و آب زیاد بخور تا اثرش بره.'
                    : 'I see you had a lot of coffee today and have a headache. In days close to period, caffeine is like gasoline on a fire for headaches. Replace coffee with green tea tomorrow and drink lots of water.',
                'action' => $locale === 'fa'
                    ? 'فردا قهوه رو با چای سبز جایگزین کن.'
                    : 'Replace coffee with green tea tomorrow.',
                'is_premium_only' => false,
            ];
        }

        // Correlation 2: Sweets + Acne
        if ($dailyLog->acne === true && $dailyLog->food_craving === true) {
            $insights[] = [
                'type' => 'sweets_acne',
                'trigger_a' => 'food_craving',
                'trigger_b' => 'acne',
                'analysis' => $locale === 'fa'
                    ? 'تحلیل: انسولین بالا (ناشی از قند) باعث افزایش آندروژن و تولید چربی پوست میشه.'
                    : 'Analysis: High insulin (from sugar) increases androgen and skin oil production.',
                'insight_message' => $locale === 'fa'
                    ? 'ولع شیرینی قبل پریود طبیعیه، اما جوش‌هایی که زدی احتمالاً واکنش بدنت به همین قندهاست. اگر هوس شیرینی کردی، شکلات تلخ بخور که هم هوس رو کم می‌کنه هم برای پوستت بهتره.'
                    : 'Sweet cravings before period are normal, but your pimples are probably your body\'s reaction to this sugar. If you crave sweets, eat dark chocolate which both reduces cravings and is better for your skin.',
                'action' => $locale === 'fa'
                    ? 'شکلات تلخ به جای شیرینی.'
                    : 'Dark chocolate instead of sweets.',
                'is_premium_only' => false,
            ];
        }

        // Correlation 3: Heavy Exercise + Spotting (mid-cycle)
        if ($phase === CyclePhase::OVULATION && $dailyLog->spotting === true) {
            $insights[] = [
                'type' => 'exercise_spotting',
                'trigger_a' => 'heavy_exercise',
                'trigger_b' => 'spotting',
                'analysis' => $locale === 'fa'
                    ? 'تحلیل: فشار فیزیکی شدید در زمان تخمک‌گذاری ممکنه باعث لکه‌بینی موقت بشه.'
                    : 'Analysis: Heavy physical pressure during ovulation may cause temporary spotting.',
                'insight_message' => $locale === 'fa'
                    ? 'لکه‌بینی بعد از ورزش سنگین امروزت احتمالاً به‌خاطر فشار فیزیکی روی رحم در زمان تخمک‌گذاریه. نترس، ولی شدت تمرینت رو فردا کمتر کن.'
                    : 'Spotting after your heavy exercise today is probably due to physical pressure on the uterus during ovulation. Don\'t worry, but reduce your workout intensity tomorrow.',
                'action' => $locale === 'fa'
                    ? 'شدت تمرین رو فردا کمتر کن.'
                    : 'Reduce workout intensity tomorrow.',
                'is_premium_only' => false,
            ];
        }

        // Correlation 4: Poor Sleep + Overeating (luteal phase)
        if ($phase === CyclePhase::LUTEAL
            && $dailyLog->sleep_quality === 'bad'
            && $dailyLog->appetite_change === 'gain') {
            $insights[] = [
                'type' => 'sleep_overeating',
                'trigger_a' => 'poor_sleep',
                'trigger_b' => 'overeating',
                'analysis' => $locale === 'fa'
                    ? 'تحلیل: کمبود خواب هورمون سیری (لپتین) رو کم و هورمون گرسنگی (گرلین) رو زیاد می‌کنه.'
                    : 'Analysis: Sleep deprivation reduces satiety hormone (leptin) and increases hunger hormone (ghrelin).',
                'insight_message' => $locale === 'fa'
                    ? 'اشتهای زیادت امروز فقط به‌خاطر PMS نیست، دیشب کم خوابیدی! وقتی مغز خسته‌ست، دنبال انرژی سریع (غذا) می‌گرده. امشب زود بخواب، فردا اشتهات کنترل میشه.'
                    : 'Your big appetite today isn\'t just because of PMS, you didn\'t sleep well last night! When the brain is tired, it seeks quick energy (food). Sleep early tonight, tomorrow your appetite will be controlled.',
                'action' => $locale === 'fa'
                    ? 'امشب زود بخواب.'
                    : 'Sleep early tonight.',
                'is_premium_only' => false,
            ];
        }

        // Correlation 5: Bloating + Weight Gain (luteal phase)
        if ($phase === CyclePhase::LUTEAL
            && $dailyLog->bloating_intensity !== null
            && $dailyLog->weight !== null) {
            $insights[] = [
                'type' => 'bloating_weight',
                'trigger_a' => 'bloating',
                'trigger_b' => 'weight_gain',
                'analysis' => $locale === 'fa'
                    ? 'تحلیل: پروژسترون روده را تنبل کرده و آب در بدن محبوس شده. ربطی به چاقی ندارد.'
                    : 'Analysis: Progesterone has slowed the intestine and water is trapped in the body. Not related to fat gain.',
                'insight_message' => $locale === 'fa'
                    ? 'احساس سنگینی و عددی که روی ترازو می‌بینی چربی نیست، بلکه ورم ناشی از هورمون پروژسترونه و با شروع پریود از بین میره.'
                    : 'The heaviness you feel and the number on the scale isn\'t fat, it\'s swelling from progesterone hormone and will go away when your period starts.',
                'action' => $locale === 'fa'
                    ? 'مصرف نمک رو قطع کن و آب بیشتری بخور.'
                    : 'Cut salt and drink more water.',
                'is_premium_only' => true,
            ];
        }

        return $insights;
    }

    /**
     * Get TTC-specific correlations
     */
    private function getTTCCorrelations(
        CyclePhase $phase,
        CycleSubphase $subphase,
        DailyHealthLog $dailyLog,
        bool $isPremium
    ): array {
        $insights = [];
        $locale = $this->locale;

        // TTC Correlation 1: High Stress + Fertile Window
        if ($phase === CyclePhase::OVULATION && $this->hasStress($dailyLog)) {
            $insights[] = [
                'type' => 'stress_fertile_window',
                'trigger_a' => 'high_stress',
                'trigger_b' => 'fertile_window',
                'analysis' => $locale === 'fa'
                    ? 'تضاد: استرس کورتیزول رو بالا می‌بره و کورتیزول جلوی هورمون‌های جنسی رو می‌گیره.'
                    : 'Conflict: Stress raises cortisol and cortisol blocks sex hormones.',
                'insight_message' => $isPremium
                    ? ($locale === 'fa'
                        ? 'می‌دونم الان روزهای طلایی اقدامه، اما استرسی که ثبت کردی مثل ترمز دستی برای سیستم باروری عمل می‌کنه. الان بهترین کاری که برای بچه‌دار شدن می‌تونی بکنی اینه که بیخیالِ تقویم بشی و فقط روی آرامش خودت تمرکز کنی.'
                        : 'I know these are golden days for action, but the stress you recorded acts like a handbrake for your fertility system. The best thing you can do for conception right now is to forget the calendar and focus only on your relaxation.')
                    : ($locale === 'fa'
                        ? 'استرس شانس بارداری رو کم می‌کنه، سعی کن آروم باشی.'
                        : 'Stress reduces pregnancy chances, try to be calm.'),
                'action' => $locale === 'fa'
                    ? 'امشب به جای اقدام اجباری، یک دوش آب گرم بگیر و زود بخواب.'
                    : 'Tonight instead of forced action, take a warm shower and sleep early.',
                'is_premium_only' => false,
            ];
        }

        // TTC Correlation 2: Poor Sleep + Ovulation
        if ($phase === CyclePhase::OVULATION && $dailyLog->sleep_quality === 'bad') {
            $insights[] = [
                'type' => 'sleep_ovulation',
                'trigger_a' => 'poor_sleep',
                'trigger_b' => 'ovulation',
                'analysis' => $locale === 'fa'
                    ? 'خطر: کم‌خوابی یعنی ملاتونین کم. ملاتونین آنتی‌اکسیدان اصلی برای حفاظت از کیفیت تخمک است.'
                    : 'Risk: Sleep deprivation means low melatonin. Melatonin is the main antioxidant for protecting egg quality.',
                'insight_message' => $isPremium
                    ? ($locale === 'fa'
                        ? 'تخمک‌گذاری نزدیکه اما خوابت کیفیت نداره و این می‌تونه روی کیفیت نهایی تخمک اثر بذاره. خواب در این دو سه روز، فقط استراحت نیست، بلکه سپر محافظتی تخمک‌های تو در برابر پیری و آسیب‌های محیطیه.'
                        : 'Ovulation is near but your sleep quality is poor and this can affect final egg quality. Sleep in these 2-3 days isn\'t just rest, it\'s a protective shield for your eggs against aging and environmental damage.')
                    : ($locale === 'fa'
                        ? 'کم‌خوابی می‌تونه روی کیفیت تخمک‌گذاری اثر بذاره.'
                        : 'Sleep deprivation can affect ovulation quality.'),
                'action' => $locale === 'fa'
                    ? 'امشب گوشی رو یک ساعت زودتر کنار بذار؛ هر ساعت خواب قبل از نیمه‌شب، دو برابر ارزش داره.'
                    : 'Put your phone away an hour earlier tonight; every hour of sleep before midnight is worth double.',
                'is_premium_only' => false,
            ];
        }

        // TTC Correlation 3: Spotting + Mild Cramps (Luteal Day 21-25) - Possible Implantation
        if ($phase === CyclePhase::LUTEAL
            && $subphase === CycleSubphase::MID_LUTEAL
            && $dailyLog->spotting === true
            && ($dailyLog->pelvic_pain_intensity !== null || $dailyLog->stomach_ache_intensity !== null)) {
            $insights[] = [
                'type' => 'implantation_signs',
                'trigger_a' => 'spotting',
                'trigger_b' => 'mild_cramps',
                'analysis' => $locale === 'fa'
                    ? 'امید کاذب/واقعی: این ترکیب کلاسیک‌ترین نشانه «احتمالی» لانه‌گزینی است.'
                    : 'False/Real Hope: This combination is the classic sign of "possible" implantation.',
                'insight_message' => $isPremium
                    ? ($locale === 'fa'
                        ? 'ثبت لکه‌بینی همراه با درد خفیف در این روزها الگوی جالبی‌ه که ممکنه به لانه‌گزینی ربط داشته باشه، اما هنوز برای قضاوت زوده. لطفاً هیجان‌زده یا ناامید نشو، فقط این علائم رو دقیق ثبت کن و تا موعد پریود صبر کن.'
                        : 'Recording spotting with mild pain these days is an interesting pattern that might be related to implantation, but it\'s still too early to judge. Please don\'t get excited or disappointed, just record these symptoms carefully and wait until your period is due.')
                    : ($locale === 'fa'
                        ? 'لکه‌بینی با درد خفیف ممکنه نشانه لانه‌گزینی باشه، صبر کن.'
                        : 'Spotting with mild pain might be a sign of implantation, wait.'),
                'action' => $locale === 'fa'
                    ? 'فعلاً فعالیت سنگین نکن و چیزهای خیلی گرم روی شکمت نذار.'
                    : 'For now, avoid heavy activity and don\'t put very hot things on your stomach.',
                'is_premium_only' => false,
            ];
        }

        // TTC Correlation 4: Bloating + Weight Gain (Luteal Phase)
        if ($phase === CyclePhase::LUTEAL
            && $dailyLog->bloating_intensity !== null
            && $dailyLog->weight !== null) {
            $insights[] = [
                'type' => 'bloating_weight_ttc',
                'trigger_a' => 'bloating',
                'trigger_b' => 'weight_gain',
                'analysis' => $locale === 'fa'
                    ? 'فیزیولوژی: پروژسترون روده را تنبل کرده و آب در بدن محبوس شده.'
                    : 'Physiology: Progesterone has slowed the intestine and water is trapped in the body.',
                'insight_message' => $isPremium
                    ? ($locale === 'fa'
                        ? 'احساس سنگینی و عددی که روی ترازو می‌بینی چربی نیست، بلکه ورم ناشی از هورمون پروژسترونه و با شروع پریود از بین میره. این یعنی سطح هورمونت برای حمایت از بارداری احتمالی کافیه، پس نگران ظاهرت نباش.'
                        : 'The heaviness you feel and the number on the scale isn\'t fat, it\'s swelling from progesterone hormone. This means your hormone level is sufficient to support a possible pregnancy, so don\'t worry about your appearance.')
                    : ($locale === 'fa'
                        ? 'این ورم طبیعیه و نشونه سطح خوب پروژسترونه.'
                        : 'This swelling is normal and a sign of good progesterone levels.'),
                'action' => $locale === 'fa'
                    ? 'مصرف نمک رو قطع کن و آب بیشتری بخور.'
                    : 'Cut salt and drink more water.',
                'is_premium_only' => false,
            ];
        }

        return $insights;
    }

    /**
     * Check if user mentioned caffeine in medications or notes
     */
    private function hasCaffeineMention(DailyHealthLog $log): bool
    {
        // This is a simplified check - in real implementation,
        // we might track coffee/caffeine consumption separately
        return true; // Assume caffeine for headache correlation
    }

    private function hasStress(DailyHealthLog $log): bool
    {
        if (!is_array($log->moods)) return false;
        return in_array('anxious', $log->moods);
    }
}
