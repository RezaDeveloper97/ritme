<?php

namespace App\Services\MatrixEngine;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Enums\OverrideType;
use App\Models\DailyHealthLog;
use App\Models\User;
use App\Models\UserProfile;

/**
 * TTC Matrix Engine - Trying to Conceive Matrix
 *
 * Generates personalized messages for users who are trying to conceive.
 * User Goal: Wants to get pregnant.
 */
class TTCMatrixEngine
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
     * Get TTC matrix message for a specific date and cycle data
     */
    public function getMatrixMessage(
        CyclePhase $phase,
        CycleSubphase $subphase,
        int $cycleDay,
        ?DailyHealthLog $dailyLog,
        bool $isFertileWindow = false
    ): array {
        $overrideType = $this->detectTTCOverrideType($phase, $subphase, $dailyLog);
        $isPremium = $this->profile?->isPremium() ?? false;

        $message = $this->getPhaseMessage($phase, $subphase, $cycleDay, $overrideType, $isPremium, $isFertileWindow);

        return [
            'phase' => $phase->value,
            'subphase' => $subphase->value,
            'cycle_day' => $cycleDay,
            'override_type' => $overrideType->value,
            'is_premium' => $isPremium,
            'is_fertile_window' => $isFertileWindow,
            'ttc_mode' => true,
            'short_message' => $message['short_message'],
            'long_message' => $message['long_message'],
            'action_suggestion' => $message['action'],
            'dos' => $message['dos'],
            'donts' => $message['donts'],
            'ttc_tip' => $message['ttc_tip'] ?? null,
        ];
    }

    /**
     * Detect TTC-specific override type
     */
    private function detectTTCOverrideType(CyclePhase $phase, CycleSubphase $subphase, ?DailyHealthLog $log): OverrideType
    {
        if (!$log) {
            return OverrideType::NORMAL;
        }

        // TTC-specific overrides

        // Period grief (menstruation phase with sad mood)
        if ($phase === CyclePhase::MENSTRUATION && $this->hasSadMood($log)) {
            return OverrideType::TTC_GRIEF;
        }

        // Performance anxiety during fertile window
        if ($phase === CyclePhase::OVULATION && $this->hasStress($log)) {
            return OverrideType::TTC_PERFORMANCE_ANXIETY;
        }

        // Spotting in luteal phase (possible implantation)
        if ($phase === CyclePhase::LUTEAL && $log->spotting === true && $subphase === CycleSubphase::MID_LUTEAL) {
            return OverrideType::IMPLANTATION_SPOTTING;
        }

        // General overrides
        if ($this->hasDischargeChange($log)) {
            return OverrideType::DISCHARGE;
        }

        if ($log->ovarian_pain_intensity !== null) {
            return OverrideType::OVARIAN_PAIN;
        }

        if ($this->hasStress($log)) {
            return OverrideType::STRESS;
        }

        return OverrideType::NORMAL;
    }

    private function hasSadMood(?DailyHealthLog $log): bool
    {
        if (!$log || !is_array($log->moods)) return false;
        return in_array('sad', $log->moods);
    }

    private function hasStress(?DailyHealthLog $log): bool
    {
        if (!$log || !is_array($log->moods)) return false;
        return in_array('anxious', $log->moods);
    }

    private function hasDischargeChange(?DailyHealthLog $log): bool
    {
        if (!$log) return false;
        return $log->discharge_texture !== null || $log->discharge_amount === 'high';
    }

    /**
     * Get TTC phase-specific message
     */
    private function getPhaseMessage(
        CyclePhase $phase,
        CycleSubphase $subphase,
        int $cycleDay,
        OverrideType $overrideType,
        bool $isPremium,
        bool $isFertileWindow
    ): array {
        return match ($phase) {
            CyclePhase::MENSTRUATION => $this->getMenstruationTTCMessage($overrideType, $isPremium),
            CyclePhase::FOLLICULAR => $this->getFollicularTTCMessage($overrideType, $isPremium),
            CyclePhase::OVULATION => $this->getOvulationTTCMessage($overrideType, $isPremium, $isFertileWindow),
            CyclePhase::LUTEAL => $this->getLutealTTCMessage($subphase, $overrideType, $isPremium),
        };
    }

    /**
     * Phase 1: Menstruation TTC - "Reset"
     */
    private function getMenstruationTTCMessage(OverrideType $overrideType, bool $isPremium): array
    {
        $locale = $this->locale;

        if ($overrideType === OverrideType::TTC_GRIEF) {
            $longMessage = $locale === 'fa'
                ? 'ناراحتی طبیعی است. فردا روز بهتری است.'
                : 'It\'s natural to be sad. Tomorrow will be a better day.';

            if ($isPremium) {
                $longMessage = $locale === 'fa'
                    ? 'حق داری ناراحت باشی. اما این پریود یعنی بدن سالم است و برای تلاش جدید آماده می‌شود. ریست هورمونی انجام شد و فرصت جدیدی پیش روته.'
                    : 'You have the right to be sad. But this period means your body is healthy and getting ready for a new try. Hormonal reset is done and a new opportunity is ahead.';
            }

            return [
                'short_message' => $locale === 'fa'
                    ? 'ناراحتی طبیعیه، فردا بهتره.'
                    : 'Being sad is natural, tomorrow is better.',
                'long_message' => $longMessage,
                'action' => $locale === 'fa'
                    ? 'تجسم مثبت بارداری در سیکل جدید داشته باش.'
                    : 'Have positive visualization of pregnancy in the new cycle.',
                'dos' => $locale === 'fa'
                    ? ['استراحت', 'تجسم مثبت', 'صحبت با همسر']
                    : ['Rest', 'Positive visualization', 'Talk to partner'],
                'donts' => $locale === 'fa'
                    ? ['سرزنش خود', 'مقایسه با دیگران']
                    : ['Self-blame', 'Comparing with others'],
                'ttc_tip' => $locale === 'fa'
                    ? 'رحم دارد محیط را برای مهمان جدید آماده می‌کند.'
                    : 'The uterus is preparing the environment for a new guest.',
            ];
        }

        // Normal TTC menstruation message
        return [
            'short_message' => $locale === 'fa'
                ? 'سیکل جدید شروع شد. استراحت کن.'
                : 'New cycle started. Rest.',
            'long_message' => $isPremium
                ? ($locale === 'fa'
                    ? 'ریست هورمونی انجام شد. فرصت جدید! رحم دارد محیط را برای مهمان جدید آماده می‌کند. این ریزش یعنی سلامتی سیستم.'
                    : 'Hormonal reset done. New opportunity! The uterus is preparing the environment for a new guest. This shedding means system health.')
                : ($locale === 'fa'
                    ? 'بدن در حال پاکسازی است. انرژی کم طبیعی است.'
                    : 'Body is cleansing. Low energy is natural.'),
            'action' => $isPremium
                ? ($locale === 'fa'
                    ? 'شروع مکمل فولیک اسید، غذای خون‌ساز بخور.'
                    : 'Start folic acid supplement, eat blood-building food.')
                : ($locale === 'fa'
                    ? 'فیلم آرام ببین و استراحت کن.'
                    : 'Watch a calm movie and rest.'),
            'dos' => $locale === 'fa'
                ? ['استراحت', 'آب گرم', 'شروع مکمل فولیک اسید', 'غذای خون‌ساز']
                : ['Rest', 'Hot water', 'Start folic acid supplement', 'Blood-building food'],
            'donts' => $locale === 'fa'
                ? ['استرس زیاد', 'سرزنش خود برای پریود شدن']
                : ['Too much stress', 'Self-blame for getting period'],
            'ttc_tip' => $locale === 'fa'
                ? 'این زمان خوبی برای شروع یا ادامه مصرف فولیک اسید است.'
                : 'This is a good time to start or continue taking folic acid.',
        ];
    }

    /**
     * Phase 2: Follicular TTC - "Build-Up"
     */
    private function getFollicularTTCMessage(OverrideType $overrideType, bool $isPremium): array
    {
        $locale = $this->locale;

        if ($overrideType === OverrideType::DISCHARGE) {
            $longMessage = $locale === 'fa'
                ? 'ترشحات تغییر کرده، طبیعی است.'
                : 'Discharge has changed, it\'s normal.';

            if ($isPremium) {
                $longMessage = $locale === 'fa'
                    ? 'ترشحات شفاف نشانه افزایش استروژن و آمادگی رحم است. بدنت داره برای روزهای طلایی آماده میشه.'
                    : 'Clear discharge is a sign of rising estrogen and uterine readiness. Your body is preparing for the golden days.';
            }

            return [
                'short_message' => $locale === 'fa'
                    ? 'ترشحات تغییر کرده، بدنت آماده میشه.'
                    : 'Discharge changed, your body is getting ready.',
                'long_message' => $longMessage,
                'action' => $locale === 'fa'
                    ? 'ترشحات رو هر روز چک کن و ثبت کن.'
                    : 'Check and record your discharge every day.',
                'dos' => $locale === 'fa'
                    ? ['ثبت ترشحات', 'تغذیه سالم', 'خواب کافی']
                    : ['Record discharge', 'Healthy nutrition', 'Enough sleep'],
                'donts' => $locale === 'fa'
                    ? ['الکل', 'سیگار']
                    : ['Alcohol', 'Smoking'],
            ];
        }

        // Normal TTC follicular message
        return [
            'short_message' => $locale === 'fa'
                ? 'بدنت داره آماده میشه!'
                : 'Your body is getting ready!',
            'long_message' => $isPremium
                ? ($locale === 'fa'
                    ? 'کارخانه تخمک‌سازی فعال شد! کیفیت تغذیه این روزها مستقیم روی DNA تخمک اثر دارد.'
                    : 'Egg factory activated! Nutrition quality these days directly affects egg DNA.')
                : ($locale === 'fa'
                    ? 'استروژن در حال بالا رفتن است. بدنت داره آماده میشه.'
                    : 'Estrogen is rising. Your body is getting ready.'),
            'action' => $isPremium
                ? ($locale === 'fa'
                    ? 'مصرف آنتی‌اکسیدان (آجیل، سبزیجات) و خرید کیت تخمک‌گذاری.'
                    : 'Consume antioxidants (nuts, vegetables) and buy an ovulation kit.')
                : ($locale === 'fa'
                    ? 'پیاده‌روی منظم داشته باش.'
                    : 'Have regular walks.'),
            'dos' => $locale === 'fa'
                ? ['تغذیه سالم', 'خواب کافی', 'مصرف آنتی‌اکسیدان', 'آجیل و سبزیجات']
                : ['Healthy nutrition', 'Enough sleep', 'Consume antioxidants', 'Nuts and vegetables'],
            'donts' => $locale === 'fa'
                ? ['الکل', 'سیگار', 'شکر و غذاهای فرآوری شده']
                : ['Alcohol', 'Smoking', 'Sugar and processed foods'],
            'ttc_tip' => $locale === 'fa'
                ? 'این هفته وقت خرید کیت تخمک‌گذاری است!'
                : 'This week is the time to buy an ovulation kit!',
        ];
    }

    /**
     * Phase 3: Ovulation TTC - "Game Time"
     */
    private function getOvulationTTCMessage(OverrideType $overrideType, bool $isPremium, bool $isFertileWindow): array
    {
        $locale = $this->locale;

        if ($overrideType === OverrideType::TTC_PERFORMANCE_ANXIETY) {
            $longMessage = $locale === 'fa'
                ? 'آروم باش. استرس شانس رو کم می‌کنه.'
                : 'Calm down. Stress reduces chances.';

            if ($isPremium) {
                $longMessage = $locale === 'fa'
                    ? 'استرس باعث ترشح هورمون‌هایی میشه که تخمک‌گذاری رو مختل می‌کنن. فقط لذت ببر و بیخیال نتیجه باش.'
                    : 'Stress causes the release of hormones that disrupt ovulation. Just enjoy and forget about the outcome.';
            }

            return [
                'short_message' => $locale === 'fa'
                    ? 'آروم باش، استرس کمکی نمی‌کنه.'
                    : 'Calm down, stress doesn\'t help.',
                'long_message' => $longMessage,
                'action' => $locale === 'fa'
                    ? 'فضای عاشقانه ایجاد کن و فقط لذت ببر.'
                    : 'Create a romantic atmosphere and just enjoy.',
                'dos' => $locale === 'fa'
                    ? ['آرامش', 'فضای عاشقانه', 'لذت بردن']
                    : ['Relaxation', 'Romantic atmosphere', 'Enjoying'],
                'donts' => $locale === 'fa'
                    ? ['استرس عملکرد', 'تمرکز روی نتیجه']
                    : ['Performance anxiety', 'Focusing on outcome'],
            ];
        }

        if ($overrideType === OverrideType::OVARIAN_PAIN && $isPremium) {
            return [
                'short_message' => $locale === 'fa'
                    ? 'این درد یعنی تخمک‌گذاری الان داره میفته!'
                    : 'This pain means ovulation is happening now!',
                'long_message' => $locale === 'fa'
                    ? 'این درد تیز یعنی دقیقاً همین الان تخمک‌گذاری اتفاق افتاد یا داره میفته. این بهترین سیگنال بدنه که بهت میگه زمان‌بندی درسته، پس نگران نباش و اقدام داشته باش.'
                    : 'This sharp pain means ovulation is happening right now. This is the best signal from your body telling you the timing is right, so don\'t worry and take action.',
                'action' => $locale === 'fa'
                    ? 'الان بهترین زمان برای اقدامه!'
                    : 'Now is the best time for action!',
                'dos' => $locale === 'fa'
                    ? ['اقدام', 'ثبت زمان درد']
                    : ['Action', 'Record pain timing'],
                'donts' => $locale === 'fa'
                    ? ['نگرانی', 'از دست دادن فرصت']
                    : ['Worry', 'Missing the opportunity'],
            ];
        }

        // Normal TTC ovulation message
        return [
            'short_message' => $locale === 'fa'
                ? 'روزهای طلایی اقدام!'
                : 'Golden days for action!',
            'long_message' => $isPremium
                ? ($locale === 'fa'
                    ? 'پنجره باروری باز شد! بیشترین شانس بارداری در روزهای قبل از تخمک‌گذاری است. رابطه یک روز در میان توصیه میشه.'
                    : 'Fertile window is open! Highest pregnancy chance is in the days before ovulation. Intercourse every other day is recommended.')
                : ($locale === 'fa'
                    ? 'بهترین زمان برای اقدام است. رابطه منظم داشته باش.'
                    : 'Best time for action. Have regular intercourse.'),
            'action' => $isPremium
                ? ($locale === 'fa'
                    ? 'رابطه یک روز در میان، پاها را بعد رابطه بالا نگه دار.'
                    : 'Intercourse every other day, keep legs elevated after.')
                : ($locale === 'fa'
                    ? 'فضای عاشقانه ایجاد کن.'
                    : 'Create a romantic atmosphere.'),
            'dos' => $locale === 'fa'
                ? ['رابطه منظم', 'رابطه یک روز در میان', 'فضای عاشقانه']
                : ['Regular intercourse', 'Intercourse every other day', 'Romantic atmosphere'],
            'donts' => $locale === 'fa'
                ? ['استرس عملکرد', 'استفاده از لوبریکانت شیمیایی']
                : ['Performance stress', 'Using chemical lubricants'],
            'ttc_tip' => $locale === 'fa'
                ? 'اسپرم تا ۵ روز زنده می‌مونه ولی تخمک فقط ۲۴ ساعت وقت داره!'
                : 'Sperm survives up to 5 days but the egg only has 24 hours!',
        ];
    }

    /**
     * Phase 4: Luteal TTC - "Two Week Wait"
     */
    private function getLutealTTCMessage(CycleSubphase $subphase, OverrideType $overrideType, bool $isPremium): array
    {
        $locale = $this->locale;

        if ($overrideType === OverrideType::IMPLANTATION_SPOTTING) {
            $longMessage = $locale === 'fa'
                ? 'لکه‌بینی ممکنه طبیعی باشه.'
                : 'Spotting might be normal.';

            if ($isPremium) {
                $longMessage = $locale === 'fa'
                    ? 'لکه‌بینی صورتی ممکنه نشونه لانه‌گزینی باشه (اما قطعی نیست). هیجان‌زده نشو، فقط ثبت کن و صبر کن.'
                    : 'Pink spotting might be a sign of implantation (but not certain). Don\'t get excited, just record and wait.';
            }

            return [
                'short_message' => $locale === 'fa'
                    ? 'لکه‌بینی دیدی؟ ممکنه نشونه خوبی باشه!'
                    : 'Saw spotting? Might be a good sign!',
                'long_message' => $longMessage,
                'action' => $locale === 'fa'
                    ? 'فعالیت سنگین نکن و چیزهای خیلی گرم روی شکمت نذار.'
                    : 'Avoid heavy activity and don\'t put very hot things on your stomach.',
                'dos' => $locale === 'fa'
                    ? ['ثبت علائم', 'صبر کردن', 'گرمای ملایم']
                    : ['Record symptoms', 'Be patient', 'Gentle warmth'],
                'donts' => $locale === 'fa'
                    ? ['فعالیت سنگین', 'کیسه آب جوش روی شکم']
                    : ['Heavy activity', 'Hot water bottle on stomach'],
            ];
        }

        // PMS symptoms that could also be pregnancy signs
        if ($subphase === CycleSubphase::LATE_LUTEAL && $isPremium) {
            return [
                'short_message' => $locale === 'fa'
                    ? 'دوران انتظار، صبور باش.'
                    : 'Waiting period, be patient.',
                'long_message' => $locale === 'fa'
                    ? 'علائم بارداری و علائم قبل از پریود (PMS) کاملاً شبیه هم هستن چون هر دو ناشی از پروژسترونن. این علائم رو به عنوان نشانه قطعی تفسیر نکن.'
                    : 'Pregnancy symptoms and PMS symptoms are completely similar because both are caused by progesterone. Don\'t interpret these symptoms as a definite sign.',
                'action' => $locale === 'fa'
                    ? 'زندگی عادی ادامه بده و تا روز تأخیر پریود صبر کن.'
                    : 'Continue normal life and wait until your period is late.',
                'dos' => $locale === 'fa'
                    ? ['زندگی عادی', 'غذای گرم', 'خواب کافی', 'پاها را گرم نگه دار']
                    : ['Normal life', 'Warm food', 'Enough sleep', 'Keep feet warm'],
                'donts' => $locale === 'fa'
                    ? ['تست زودرس', 'جستجوی وسواسی علائم']
                    : ['Early testing', 'Obsessive symptom searching'],
                'ttc_tip' => $locale === 'fa'
                    ? 'تست منفی الان هیچ اعتباری نداره، تا روز تأخیر پریود صبر کن.'
                    : 'A negative test now has no validity, wait until your period is late.',
            ];
        }

        // Normal TTC luteal message
        return [
            'short_message' => $locale === 'fa'
                ? 'دوران انتظار شروع شد.'
                : 'Waiting period started.',
            'long_message' => $isPremium
                ? ($locale === 'fa'
                    ? 'فاز لانه‌گزینی آغاز شد. لانه‌گزینی بین روزهای ۶-۱۰ بعد تخمک‌گذاری اتفاق می‌افتد.'
                    : 'Implantation phase started. Implantation occurs between days 6-10 after ovulation.')
                : ($locale === 'fa'
                    ? 'حالا نوبت بدن است. صبر کن.'
                    : 'Now it\'s the body\'s turn. Wait.'),
            'action' => $isPremium
                ? ($locale === 'fa'
                    ? 'غذای گرم، خواب کافی، پاها را گرم نگه دار.'
                    : 'Warm food, enough sleep, keep feet warm.')
                : ($locale === 'fa'
                    ? 'خودت رو سرگرم کن و زندگی عادی ادامه بده.'
                    : 'Keep yourself busy and continue normal life.'),
            'dos' => $locale === 'fa'
                ? ['زندگی عادی', 'غذای گرم', 'خواب کافی']
                : ['Normal life', 'Warm food', 'Enough sleep'],
            'donts' => $locale === 'fa'
                ? ['تست زودرس', 'جستجوی وسواسی علائم']
                : ['Early testing', 'Obsessive symptom searching'],
            'ttc_tip' => $locale === 'fa'
                ? 'صبر سخت‌ترین بخش اقدام به بارداریه، خودت رو سرگرم کن!'
                : 'Patience is the hardest part of TTC, keep yourself busy!',
        ];
    }
}
