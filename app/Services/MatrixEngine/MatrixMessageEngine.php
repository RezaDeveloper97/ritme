<?php

namespace App\Services\MatrixEngine;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Enums\OverrideType;
use App\Models\DailyHealthLog;
use App\Models\User;
use App\Models\UserProfile;
use Carbon\Carbon;

/**
 * Matrix Message Engine - Layer 1 & 2
 *
 * Generates personalized messages based on cycle phase, day, and symptom overrides.
 * Logic: Default message is based on phase energy, unless a symptom (like pain) overrides it.
 */
class MatrixMessageEngine
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
     * Get matrix message for a specific date and cycle data
     */
    public function getMatrixMessage(
        CyclePhase $phase,
        CycleSubphase $subphase,
        int $cycleDay,
        ?DailyHealthLog $dailyLog
    ): array {
        // Detect override type based on symptoms
        $overrideType = $this->detectOverrideType($phase, $dailyLog);
        $isPremium = $this->profile?->isPremium() ?? false;

        // Get phase-specific message
        $message = $this->getPhaseMessage($phase, $subphase, $cycleDay, $overrideType, $isPremium);

        return [
            'phase' => $phase->value,
            'subphase' => $subphase->value,
            'cycle_day' => $cycleDay,
            'override_type' => $overrideType->value,
            'is_premium' => $isPremium,
            'short_message' => $message['short_message'],
            'long_message' => $message['long_message'],
            'action_suggestion' => $message['action'],
            'dos' => $message['dos'],
            'donts' => $message['donts'],
            'fun_fact' => $message['fun_fact'] ?? null,
        ];
    }

    /**
     * Detect override type based on symptoms
     */
    private function detectOverrideType(CyclePhase $phase, ?DailyHealthLog $log): OverrideType
    {
        if (!$log) {
            return OverrideType::NORMAL;
        }

        // Priority order: Pain > Mood > Physical symptoms

        // Check for severe pain (cramps)
        if ($this->hasSeverePain($log)) {
            return OverrideType::PAIN;
        }

        // Check for heavy flow with clots
        if ($this->hasHeavyFlow($log)) {
            return OverrideType::HEAVY_FLOW;
        }

        // Check for sad mood
        if ($this->hasSadMood($log)) {
            return OverrideType::MOOD_SAD;
        }

        // Check for angry mood (in luteal phase)
        if ($phase === CyclePhase::LUTEAL && $this->hasAngryMood($log)) {
            return OverrideType::MOOD_ANGRY;
        }

        // Check for acne
        if ($log->acne === true) {
            return OverrideType::SKIN_ACNE;
        }

        // Check for discharge changes
        if ($this->hasDischargeChange($log)) {
            return OverrideType::DISCHARGE;
        }

        // Check for headache
        if ($log->headache_intensity !== null) {
            return OverrideType::HEADACHE;
        }

        // Check for ovarian pain (Mittelschmerz)
        if ($log->ovarian_pain_intensity !== null) {
            return OverrideType::OVARIAN_PAIN;
        }

        // Check for spotting in luteal phase
        if ($phase === CyclePhase::LUTEAL && $log->spotting === true) {
            return OverrideType::SPOTTING;
        }

        // Check for stress/anxiety
        if ($this->hasStress($log)) {
            return OverrideType::STRESS;
        }

        return OverrideType::NORMAL;
    }

    private function hasSeverePain(?DailyHealthLog $log): bool
    {
        if (!$log) return false;

        return $log->stomach_ache_intensity === 'high'
            || $log->pelvic_pain_intensity === 'high'
            || $log->back_pain_intensity === 'high';
    }

    private function hasHeavyFlow(?DailyHealthLog $log): bool
    {
        if (!$log) return false;

        return ($log->bleeding_intensity === 'very_high' || $log->bleeding_intensity === 'high')
            && $log->has_clots === true;
    }

    private function hasSadMood(?DailyHealthLog $log): bool
    {
        if (!$log || !is_array($log->moods)) return false;

        return in_array('sad', $log->moods);
    }

    private function hasAngryMood(?DailyHealthLog $log): bool
    {
        if (!$log || !is_array($log->moods)) return false;

        return in_array('angry', $log->moods) || in_array('frustrated', $log->moods);
    }

    private function hasDischargeChange(?DailyHealthLog $log): bool
    {
        if (!$log) return false;

        return $log->discharge_texture !== null
            || $log->discharge_amount === 'high';
    }

    private function hasStress(?DailyHealthLog $log): bool
    {
        if (!$log || !is_array($log->moods)) return false;

        return in_array('anxious', $log->moods);
    }

    /**
     * Get phase-specific message with override handling
     */
    private function getPhaseMessage(
        CyclePhase $phase,
        CycleSubphase $subphase,
        int $cycleDay,
        OverrideType $overrideType,
        bool $isPremium
    ): array {
        return match ($phase) {
            CyclePhase::MENSTRUATION => $this->getMenstruationMessage($cycleDay, $overrideType, $isPremium),
            CyclePhase::FOLLICULAR => $this->getFollicularMessage($cycleDay, $overrideType, $isPremium),
            CyclePhase::OVULATION => $this->getOvulationMessage($cycleDay, $overrideType, $isPremium),
            CyclePhase::LUTEAL => $this->getLutealMessage($cycleDay, $subphase, $overrideType, $isPremium),
        };
    }

    /**
     * Phase 1: Menstruation (The Winter - Days 1-5)
     */
    private function getMenstruationMessage(int $cycleDay, OverrideType $overrideType, bool $isPremium): array
    {
        $locale = $this->locale;

        // Override messages take priority
        if ($overrideType === OverrideType::PAIN) {
            return [
                'short_message' => $locale === 'fa'
                    ? 'درد داری، اول درد رو مدیریت کن.'
                    : 'You\'re in pain. Manage the pain first.',
                'long_message' => $locale === 'fa'
                    ? 'می‌دونم کلافه‌ای و درد داری. الان وقت نصیحت نیست. اولویت اولت مدیریت درده. اگر مسکن خوردی و هنوز درد داری، پوزیشن «جنین» (جمع شدن در خود) می‌تونه فشار روی عضلات شکم رو کم کنه.'
                    : 'I know you\'re frustrated and in pain. Now is not the time for advice. Your first priority is pain management. If you\'ve taken painkillers and still have pain, the "fetal" position can reduce pressure on your abdominal muscles.',
                'action' => $locale === 'fa'
                    ? 'کیسه آب گرم روی شکم + جوراب گرم بپوش. گرما بهترین دوست رحم توئه.'
                    : 'Put a hot water bottle on your stomach + wear warm socks. Heat is your uterus\'s best friend.',
                'dos' => $locale === 'fa'
                    ? ['استراحت', 'کمپرس گرم', 'مسکن در صورت نیاز']
                    : ['Rest', 'Heat compress', 'Painkillers if needed'],
                'donts' => $locale === 'fa'
                    ? ['ورزش سنگین', 'کافئین زیاد']
                    : ['Heavy exercise', 'Too much caffeine'],
            ];
        }

        if ($overrideType === OverrideType::MOOD_SAD && $isPremium) {
            return [
                'short_message' => $locale === 'fa'
                    ? 'غم امروزت واقعی نیست، شیمی مغزته.'
                    : 'Your sadness today isn\'t real, it\'s brain chemistry.',
                'long_message' => $locale === 'fa'
                    ? 'هورمون‌ها افت کردن و اینکه بی‌دلیل دلت گرفته کاملاً طبیعیه. این غم واقعی نیست، شیمیِ مغزته که داره بازی درمیاره. به خودت سخت نگیر و اجازه بده امروز بگذره.'
                    : 'Hormones have dropped and feeling sad for no reason is completely normal. This sadness isn\'t real, it\'s your brain chemistry playing tricks. Don\'t be hard on yourself and let today pass.',
                'action' => $locale === 'fa'
                    ? 'یک فیلم کمدی یا چیزی که دوست داری ببین، دوپامین مصنوعی بساز!'
                    : 'Watch a comedy or something you like, create artificial dopamine!',
                'dos' => $locale === 'fa'
                    ? ['فیلم کمدی', 'مدیتیشن', 'صحبت با دوست']
                    : ['Comedy movie', 'Meditation', 'Talk to a friend'],
                'donts' => $locale === 'fa'
                    ? ['تصمیم‌گیری مهم', 'شروع پروژه جدید']
                    : ['Important decisions', 'Starting new projects'],
            ];
        }

        if ($overrideType === OverrideType::HEAVY_FLOW) {
            $longMessage = $locale === 'fa'
                ? 'خونریزی سنگین می‌تونه باعث ضعف بشه پس امروز حتماً استراحت کن و مایعات بدنت رو جایگزین کن.'
                : 'Heavy bleeding can cause weakness, so make sure to rest today and replace your body fluids.';

            if ($isPremium) {
                $longMessage = $locale === 'fa'
                    ? 'وجود لخته یعنی رحم داره با قدرت پاکسازی می‌کنه اما برای اقدام به بارداری نباید بذاری ذخایر آهنت خالی بشه. از همین امروز مصرف منابع آهن حیوانی مثل جگر یا گوشت قرمز رو جدی بگیر تا کیفیت تخمک ماه بعد حفظ بشه.'
                    : 'Clots mean your uterus is cleaning powerfully, but for conception you shouldn\'t let your iron reserves deplete. From today, take animal iron sources like liver or red meat seriously to maintain egg quality next month.';
            }

            return [
                'short_message' => $locale === 'fa'
                    ? 'خونریزی سنگین داری، استراحت کن.'
                    : 'You have heavy bleeding, rest.',
                'long_message' => $longMessage,
                'action' => $locale === 'fa'
                    ? 'آب و مایعات زیاد بخور و غذاهای غنی از آهن مصرف کن.'
                    : 'Drink lots of water and fluids, consume iron-rich foods.',
                'dos' => $locale === 'fa'
                    ? ['استراحت', 'آب زیاد', 'غذای غنی از آهن']
                    : ['Rest', 'Lots of water', 'Iron-rich food'],
                'donts' => $locale === 'fa'
                    ? ['ورزش سنگین', 'ایستادن طولانی']
                    : ['Heavy exercise', 'Standing for long'],
            ];
        }

        // Normal menstruation message
        return [
            'short_message' => $locale === 'fa'
                ? 'پریود شروع شد، مود ذخیره انرژی باش.'
                : 'Period started, be in energy-saving mode.',
            'long_message' => $locale === 'fa'
                ? 'پریود شروع شد و بدنت داره خونه‌تکونی می‌کنه. انرژی کمِ امروزت تنبلی نیست، نیاز بیولوژیک به استراحته. امروز رو مود «ذخیره انرژی» باش و کارهای سنگین رو بذار برای هفته بعد.'
                : 'Period started and your body is doing spring cleaning. Low energy today isn\'t laziness, it\'s a biological need for rest. Be in "energy saving" mode today and leave heavy tasks for next week.',
            'action' => $locale === 'fa'
                ? 'چای دارچین یا زنجبیل دم کن؛ هم گرمت می‌کنه هم جریان خون رو روان‌تر می‌کنه.'
                : 'Make cinnamon or ginger tea; it warms you and smooths blood flow.',
            'dos' => $locale === 'fa'
                ? ['استراحت', 'آب گرم', 'غذای سبک', 'ماساژ ملایم شکم', 'دمنوش‌های گرم']
                : ['Rest', 'Hot water', 'Light food', 'Gentle belly massage', 'Hot herbal teas'],
            'donts' => $locale === 'fa'
                ? ['ورزش سنگین', 'کافئین زیاد', 'تصمیم‌گیری مهم', 'شروع پروژه جدید']
                : ['Heavy exercise', 'Too much caffeine', 'Important decisions', 'Starting new projects'],
            'fun_fact' => $locale === 'fa'
                ? 'دانستنی: در این فاز، همه هورمون‌ها پایین هستند و بدن در حال بازسازی است.'
                : 'Fun fact: In this phase, all hormones are low and the body is regenerating.',
        ];
    }

    /**
     * Phase 2: Follicular (The Spring - Days 6-11)
     */
    private function getFollicularMessage(int $cycleDay, OverrideType $overrideType, bool $isPremium): array
    {
        $locale = $this->locale;

        if ($overrideType === OverrideType::SKIN_ACNE && $isPremium) {
            return [
                'short_message' => $locale === 'fa'
                    ? 'جوش‌هات با بالا رفتن استروژن زودتر خوب میشن.'
                    : 'Your acne will heal faster as estrogen rises.',
                'long_message' => $locale === 'fa'
                    ? 'تو فاز خوبی هستی ولی انگار پوستت هنوز درگیر التهاب پریود قبلیه. با بالا رفتن استروژن، کلاژن‌سازی بهتر میشه، پس نگران نباش، این جوش‌ها زودتر از همیشه خوب میشن.'
                    : 'You\'re in a good phase but your skin seems still dealing with inflammation from the previous period. As estrogen rises, collagen production improves, so don\'t worry, these pimples will heal faster than usual.',
                'action' => $locale === 'fa'
                    ? 'الان وقت لایه‌برداری ملایمه، چون پوستت توی فاز بازسازیه.'
                    : 'Now is the time for gentle exfoliation, as your skin is in the regeneration phase.',
                'dos' => $locale === 'fa'
                    ? ['لایه‌برداری ملایم', 'آب‌رسانی به پوست', 'خواب کافی']
                    : ['Gentle exfoliation', 'Skin hydration', 'Enough sleep'],
                'donts' => $locale === 'fa'
                    ? ['فشار دادن جوش‌ها', 'محصولات سنگین']
                    : ['Popping pimples', 'Heavy products'],
            ];
        }

        if ($overrideType === OverrideType::DISCHARGE) {
            $longMessage = $locale === 'fa'
                ? 'تغییر ترشحات در این روزها طبیعیه و نشون میده بدنت داره بیدار میشه.'
                : 'Discharge changes these days are normal and show your body is waking up.';

            if ($isPremium) {
                $longMessage = $locale === 'fa'
                    ? 'اگر احساس رطوبت بیشتری می‌کنی یعنی سطح استروژن داره بالا میره و رحم داره محیط رو برای اسپرم دوستانه می‌کنه. این نشونه عالیه که داریم به روزهای طلایی نزدیک می‌شیم.'
                    : 'If you feel more moisture, it means estrogen levels are rising and the uterus is making the environment sperm-friendly. This is a great sign that we\'re approaching the golden days.';
            }

            return [
                'short_message' => $locale === 'fa'
                    ? 'تغییر ترشحات طبیعیه، بدنت داره بیدار میشه.'
                    : 'Discharge changes are normal, your body is waking up.',
                'long_message' => $longMessage,
                'action' => $locale === 'fa'
                    ? 'ترشحات رو زیر نظر داشته باش، این اطلاعات مهمی هستن.'
                    : 'Keep track of your discharge, this is important information.',
                'dos' => $locale === 'fa'
                    ? ['ثبت تغییرات', 'بهداشت مناسب']
                    : ['Track changes', 'Proper hygiene'],
                'donts' => $locale === 'fa'
                    ? ['استفاده از صابون‌های قوی']
                    : ['Using harsh soaps'],
            ];
        }

        if ($overrideType === OverrideType::HEADACHE) {
            $longMessage = $locale === 'fa'
                ? 'نوسان هورمونی ممکنه باعث سردرد خفیف بشه، سعی کن آب زیاد بخوری.'
                : 'Hormonal fluctuation may cause mild headache, try to drink lots of water.';

            if ($isPremium) {
                $longMessage = $locale === 'fa'
                    ? 'گاهی افزایش استروژن باعث تحریک میگرن میشه. سعی کن با منیزیم و خواب کافی مدیریتش کنی چون مصرف مسکن‌های قوی نزدیک به تخمک‌گذاری ایده‌آل نیست.'
                    : 'Sometimes rising estrogen can trigger migraines. Try to manage it with magnesium and enough sleep, as taking strong painkillers close to ovulation isn\'t ideal.';
            }

            return [
                'short_message' => $locale === 'fa'
                    ? 'سردرد هورمونی، آب زیاد بخور.'
                    : 'Hormonal headache, drink lots of water.',
                'long_message' => $longMessage,
                'action' => $locale === 'fa'
                    ? 'آب زیاد بخور و در محیط آرام استراحت کن.'
                    : 'Drink lots of water and rest in a calm environment.',
                'dos' => $locale === 'fa'
                    ? ['آب زیاد', 'خواب کافی', 'منیزیم']
                    : ['Lots of water', 'Enough sleep', 'Magnesium'],
                'donts' => $locale === 'fa'
                    ? ['مسکن‌های قوی', 'کم‌خوابی']
                    : ['Strong painkillers', 'Sleep deprivation'],
            ];
        }

        // Normal follicular message
        return [
            'short_message' => $locale === 'fa'
                ? 'استروژن بالا میره، بهترین زمان برای کارهای سخته!'
                : 'Estrogen is rising, best time for challenging tasks!',
            'long_message' => $locale === 'fa'
                ? 'استروژن داره بالا میره و مغزت شفاف‌ترین حالت رو داره. الان بهترین زمان برای یادگیری چیزهای سخت، جلسات مهم کاری یا برنامه‌ریزی‌های پیچیده‌ست. از این موج انرژی استفاده کن!'
                : 'Estrogen is rising and your brain is at its clearest. Now is the best time for learning difficult things, important work meetings, or complex planning. Use this energy wave!',
            'action' => $locale === 'fa'
                ? 'لیست کارهای عقب‌افتاده رو بیار؛ امروز می‌تونی همه‌شو تیک بزنی.'
                : 'Bring out your backlog list; today you can check them all off.',
            'dos' => $locale === 'fa'
                ? ['برنامه‌های روزانه‌ت رو منظم کن', 'آب‌رسانی به بدن', 'آجیل خام و سبزیجات برگ سبز']
                : ['Organize your daily schedule', 'Stay hydrated', 'Raw nuts and leafy greens'],
            'donts' => $locale === 'fa'
                ? ['ورزش‌های خیلی سنگین (انرژی رو ذخیره کن)', 'شکر و شیرینی‌جات زیاد']
                : ['Very heavy exercises (save energy)', 'Too much sugar and sweets'],
            'fun_fact' => $locale === 'fa'
                ? 'دانستنی: هر کاری که الان برای سلامتیت انجام بدی، اثر مستقیمش رو چند روز دیگه توی کیفیت تخمک‌گذاری می‌بینی.'
                : 'Fun fact: Everything you do for your health now will directly affect your ovulation quality in a few days.',
        ];
    }

    /**
     * Phase 3: Ovulation (The Summer - Days 12-16)
     */
    private function getOvulationMessage(int $cycleDay, OverrideType $overrideType, bool $isPremium): array
    {
        $locale = $this->locale;

        if ($overrideType === OverrideType::OVARIAN_PAIN && $isPremium) {
            return [
                'short_message' => $locale === 'fa'
                    ? 'این درد نشونه آزاد شدن تخمکه.'
                    : 'This pain is a sign of egg release.',
                'long_message' => $locale === 'fa'
                    ? 'اون تیر کشیدن خفیف در سمت راست یا چپ شکمت، نشونه آزاد شدن تخمکه. این درد یعنی بدنت ساعت دقیقش رو اعلام کرده. نگران نباش، معمولاً چند ساعت بیشتر طول نمی‌کشه.'
                    : 'That slight pulling pain on the right or left side of your stomach is a sign of egg release. This pain means your body has announced its exact timing. Don\'t worry, it usually doesn\'t last more than a few hours.',
                'action' => $locale === 'fa'
                    ? 'اگر دردش اذیتت می‌کنه، یک دوش آب گرم و کمی استراحت کافیه.'
                    : 'If the pain bothers you, a warm shower and some rest is enough.',
                'dos' => $locale === 'fa'
                    ? ['دوش آب گرم', 'استراحت کوتاه', 'ثبت زمان درد']
                    : ['Warm shower', 'Short rest', 'Record pain timing'],
                'donts' => $locale === 'fa'
                    ? ['ورزش سنگین در صورت درد']
                    : ['Heavy exercise if in pain'],
            ];
        }

        if ($overrideType === OverrideType::STRESS) {
            $longMessage = $locale === 'fa'
                ? 'اضطراب می‌تونه لذت این روزها رو از بین ببره، سعی کن با تنفس عمیق آروم بشی.'
                : 'Anxiety can ruin the joy of these days, try to calm down with deep breathing.';

            if ($isPremium) {
                $longMessage = $locale === 'fa'
                    ? 'استرس شدید باعث ترشح هورمون‌هایی میشه که می‌تونن تخمک‌گذاری رو به تأخیر بندازن یا مختل کنن. بهترین کاری که برای بچه‌دار شدن می‌تونی بکنی اینه که بیخیالِ نتیجه بشی و فقط از رابطه لذت ببری.'
                    : 'Severe stress causes the release of hormones that can delay or disrupt ovulation. The best thing you can do for conception is to let go of the outcome and just enjoy intimacy.';
            }

            return [
                'short_message' => $locale === 'fa'
                    ? 'استرس رو کنار بذار، آرامش مهمه.'
                    : 'Put stress aside, calmness is important.',
                'long_message' => $longMessage,
                'action' => $locale === 'fa'
                    ? 'تنفس عمیق، مدیتیشن کوتاه، یا قدم زدن در طبیعت.'
                    : 'Deep breathing, short meditation, or a walk in nature.',
                'dos' => $locale === 'fa'
                    ? ['تنفس عمیق', 'مدیتیشن', 'قدم زدن']
                    : ['Deep breathing', 'Meditation', 'Walking'],
                'donts' => $locale === 'fa'
                    ? ['تمرکز روی نتیجه', 'فشار آوردن به خود']
                    : ['Focusing on outcome', 'Putting pressure on yourself'],
            ];
        }

        // Normal ovulation message
        return [
            'short_message' => $locale === 'fa'
                ? 'روز درخشش توئه! جذاب‌ترین نسخه خودتی.'
                : 'It\'s your shining day! You\'re the most attractive version of yourself.',
            'long_message' => $locale === 'fa'
                ? 'الان جذاب‌ترین، خوش‌صحبت‌ترین و پرانرژی‌ترین نسخه خودتی. ولی حواست باشه! احتمال بارداری الان در سقفِ خودش قرار داره. اگر بچه نمی‌خوای، این روزها باید دوبرابر مراقب باشی.'
                : 'You\'re now the most attractive, articulate, and energetic version of yourself. But be careful! Pregnancy probability is at its peak. If you don\'t want a baby, you need to be twice as careful these days.',
            'action' => $locale === 'fa'
                ? 'اگر قرار مهم یا ارائه کاری داری، امروز روز درخشش توئه. اعتماد به‌نفست عالیه.'
                : 'If you have an important meeting or presentation, today is your shining day. Your confidence is great.',
            'dos' => $locale === 'fa'
                ? ['استفاده از انرژی برای کارهای مهم', 'جلسات مهم کاری', 'قرارهای اجتماعی']
                : ['Use energy for important tasks', 'Important work meetings', 'Social dates'],
            'donts' => $locale === 'fa'
                ? ['اتکا به ریتم برای پیشگیری', 'تصمیم‌گیری‌های احساسی بزرگ']
                : ['Relying on rhythm for contraception', 'Big emotional decisions'],
            'fun_fact' => $locale === 'fa'
                ? 'هشدار: احتمال بارداری در اوج است. اگر بچه نمی‌خواهی مراقب باش.'
                : 'Warning: Pregnancy probability is at its peak. If you don\'t want a baby, be careful.',
        ];
    }

    /**
     * Phase 4: Luteal (The Autumn - Days 17-28)
     */
    private function getLutealMessage(
        int $cycleDay,
        CycleSubphase $subphase,
        OverrideType $overrideType,
        bool $isPremium
    ): array {
        $locale = $this->locale;

        if ($overrideType === OverrideType::MOOD_ANGRY && $isPremium) {
            return [
                'short_message' => $locale === 'fa'
                    ? 'آستانه تحملت پایین اومده، عینک بدبینی هورمونی روی چشمته.'
                    : 'Your tolerance threshold is down, hormonal pessimism glasses are on.',
                'long_message' => $locale === 'fa'
                    ? 'احساس می‌کنی همه رفتن روی اعصابت؟ حق داری. آستانه تحملت پایین اومده. قبل از اینکه بحثی رو شروع کنی یا تصمیمی بگیری، یادت باشه الان «عینک بدبینی هورمونی» روی چشمته.'
                    : 'Feel like everyone is getting on your nerves? You\'re right. Your tolerance threshold is down. Before starting an argument or making a decision, remember you\'re wearing "hormonal pessimism glasses" right now.',
                'action' => $locale === 'fa'
                    ? 'از کافئین دوری کن! قهوه الان فقط اضطراب و عصبانیتت رو بیشتر می‌کنه. دمنوش گل‌گاوزبان یا بابونه بخور.'
                    : 'Stay away from caffeine! Coffee now only increases your anxiety and anger. Drink borage or chamomile tea.',
                'dos' => $locale === 'fa'
                    ? ['دمنوش آرام‌بخش', 'تنفس عمیق', 'فاصله گرفتن از بحث']
                    : ['Calming herbal tea', 'Deep breathing', 'Distance from arguments'],
                'donts' => $locale === 'fa'
                    ? ['کافئین', 'شروع بحث', 'تصمیم‌گیری مهم']
                    : ['Caffeine', 'Starting arguments', 'Important decisions'],
            ];
        }

        if ($overrideType === OverrideType::SPOTTING) {
            $longMessage = $locale === 'fa'
                ? 'لکه‌بینی مختصر قبل از پریود می‌تونه طبیعی باشه اما اگر ادامه داشت بررسی کن.'
                : 'Light spotting before period can be normal, but check if it continues.';

            if ($isPremium) {
                $longMessage = $locale === 'fa'
                    ? 'دیدن چند قطره خون صورتی یا قهوه‌ای در این روزها "ممکنه" نشانه لانه‌گزینی باشه اما قطعی نیست. هیجان‌زده یا ناامید نشو، فقط ثبتش کن و صبر کن ببین پریود میشی یا نه.'
                    : 'Seeing a few drops of pink or brown blood these days "might" be a sign of implantation, but it\'s not certain. Don\'t get excited or disappointed, just record it and wait to see if you get your period.';
            }

            return [
                'short_message' => $locale === 'fa'
                    ? 'لکه‌بینی دیدی؟ ثبتش کن و صبر کن.'
                    : 'Saw spotting? Record it and wait.',
                'long_message' => $longMessage,
                'action' => $locale === 'fa'
                    ? 'ثبت کن و تا موعد پریود صبر کن.'
                    : 'Record it and wait until your period is due.',
                'dos' => $locale === 'fa'
                    ? ['ثبت علائم', 'صبر کردن']
                    : ['Record symptoms', 'Be patient'],
                'donts' => $locale === 'fa'
                    ? ['هیجان‌زده شدن', 'تست زودهنگام']
                    : ['Getting excited', 'Early testing'],
            ];
        }

        // PMS symptoms override
        if ($overrideType === OverrideType::PAIN || $subphase === CycleSubphase::LATE_LUTEAL) {
            $longMessage = $locale === 'fa'
                ? 'حساسیت سینه و تغییرات خلقی بخاطر پروژسترون بالاست و طبیعیه.'
                : 'Breast sensitivity and mood changes are due to high progesterone and are normal.';

            if ($isPremium) {
                $longMessage = $locale === 'fa'
                    ? 'واقعیت اینه که علائم بارداری و علائم قبل از پریود (PMS) کاملاً شبیه هم هستن چون هر دو ناشی از پروژسترونن. پس لطفاً این علائم رو به عنوان نشانه قطعی بارداری تفسیر نکن تا الکی امیدوار یا ناامید نشی.'
                    : 'The reality is that pregnancy symptoms and PMS symptoms are completely similar because both are caused by progesterone. So please don\'t interpret these symptoms as a definite sign of pregnancy to avoid false hope or disappointment.';
            }
        }

        // Normal luteal message
        return [
            'short_message' => $locale === 'fa'
                ? 'وارد فاز استراحت میشی، کارها رو تمام کن.'
                : 'Entering rest phase, finish up tasks.',
            'long_message' => $locale === 'fa'
                ? 'کم‌کم داری وارد فاز استراحت میشی. اگر احساس می‌کنی زودرنج شدی یا تمرکزت کم شده، کار خودته نیست، کار پروژسترونه. الان وقت تمام کردن کاره، نه شروع پروژه‌های بزرگ.'
                : 'You\'re gradually entering the rest phase. If you feel irritable or unfocused, it\'s not you, it\'s progesterone. Now is the time to finish tasks, not start big projects.',
            'action' => $locale === 'fa'
                ? 'کارهای روتین و کم‌چالش رو انجام بده. لیست خرید یا مرتب کردن فایل‌ها عالیه.'
                : 'Do routine and low-challenge tasks. Shopping lists or organizing files are great.',
            'dos' => $locale === 'fa'
                ? ['کارهای روتین', 'غذای سبک', 'مدیتیشن', 'ژورنال نویسی', 'نظم دادن']
                : ['Routine tasks', 'Light food', 'Meditation', 'Journaling', 'Organizing'],
            'donts' => $locale === 'fa'
                ? ['کافئین زیاد', 'تصمیم‌گیری مهم', 'شروع رژیم جدید', 'تغییرات اساسی']
                : ['Too much caffeine', 'Important decisions', 'Starting new diet', 'Major changes'],
        ];
    }
}
