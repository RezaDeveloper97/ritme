<?php

namespace App\Services\MessageSystem\Engines;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Enums\OverrideType;
use App\Models\User;
use App\Services\MessageSystem\Contracts\MessageEngineInterface;
use App\Services\MessageSystem\Core\MessageContext;
use App\Services\MessageSystem\Core\MessageResult;
use App\Services\MessageSystem\Enums\MessageMode;

class CycleMessageEngine implements MessageEngineInterface
{
    public function __construct(
        private readonly User $user,
        private readonly string $locale = 'fa',
    ) {}

    public function getMode(): string
    {
        return MessageMode::CYCLE->value;
    }

    public function canHandle(MessageContext $context): bool
    {
        return $context->isCycleMode();
    }

    public function generateMessages(MessageContext $context): MessageResult
    {
        // This is handled by MessageManager for consistency
        throw new \RuntimeException('Use MessageManager::generateMessages() instead');
    }

    public function getEnums(string $locale = 'en'): array
    {
        return [
            'phases' => collect(CyclePhase::cases())->map(fn($p) => [
                'value' => $p->value,
                'label' => $p->label($locale),
                'description' => $p->description($locale),
            ])->values()->toArray(),
            'subphases' => collect(CycleSubphase::cases())->map(fn($s) => [
                'value' => $s->value,
                'label' => $s->label($locale),
            ])->values()->toArray(),
            'override_types' => collect(OverrideType::cases())->map(fn($o) => [
                'value' => $o->value,
                'label' => $o->label($locale),
            ])->values()->toArray(),
        ];
    }

    /**
     * Get base message for the current phase (Layer 1)
     */
    public function getBaseMessage(MessageContext $context): array
    {
        if (!$context->cyclePhase) {
            return $this->getDefaultMessage();
        }

        $phase = CyclePhase::from($context->cyclePhase);

        // Check if TTC mode for different messaging
        if ($context->isTTC()) {
            return $this->getTTCBaseMessage($phase, $context);
        }

        return $this->getNonTTCBaseMessage($phase, $context);
    }

    /**
     * Get override message based on symptoms (Layer 2)
     */
    public function getOverrideMessage(MessageContext $context): ?array
    {
        if (empty($context->symptoms)) {
            return null;
        }

        // Determine override type from symptoms
        $overrideType = $this->determineOverrideType($context->symptoms);

        if (!$overrideType) {
            return null;
        }

        $phase = $context->cyclePhase ? CyclePhase::from($context->cyclePhase) : null;

        return $this->getOverrideMessageForType($overrideType, $phase, $context);
    }

    /**
     * Determine which override type applies based on symptoms
     */
    private function determineOverrideType(array $symptoms): ?OverrideType
    {
        // Priority order for override types
        if (in_array('heavy_flow', $symptoms)) {
            return OverrideType::HEAVY_FLOW;
        }
        if (in_array('cramps', $symptoms) || in_array('headache', $symptoms) || in_array('backache', $symptoms)) {
            return OverrideType::PAIN;
        }
        if (in_array('mood_sad', $symptoms) || in_array('mood_anxious', $symptoms)) {
            return OverrideType::MOOD_SAD;
        }
        if (in_array('mood_angry', $symptoms)) {
            return OverrideType::MOOD_ANGRY;
        }
        if (in_array('low_energy', $symptoms) || in_array('fatigue', $symptoms)) {
            return OverrideType::LOW_ENERGY;
        }
        if (in_array('bloating', $symptoms)) {
            return OverrideType::BLOATING;
        }
        if (in_array('acne', $symptoms)) {
            return OverrideType::ACNE;
        }

        return null;
    }

    /**
     * Get Non-TTC base message for phase
     */
    private function getNonTTCBaseMessage(CyclePhase $phase, MessageContext $context): array
    {
        $messages = $this->getNonTTCMessages();
        $phaseMessages = $messages[$phase->value] ?? $messages['default'];

        return [
            'phase' => $phase->value,
            'phase_label' => $phase->label($this->locale),
            'cycle_day' => $context->cycleDay,
            'short_message' => $phaseMessages['short'][$this->locale] ?? $phaseMessages['short']['fa'],
            'long_message' => $phaseMessages['long'][$this->locale] ?? $phaseMessages['long']['fa'],
            'action_suggestion' => $phaseMessages['action'][$this->locale] ?? $phaseMessages['action']['fa'],
            'dos' => $phaseMessages['dos'][$this->locale] ?? $phaseMessages['dos']['fa'],
            'donts' => $phaseMessages['donts'][$this->locale] ?? $phaseMessages['donts']['fa'],
        ];
    }

    /**
     * Get TTC base message for phase
     */
    private function getTTCBaseMessage(CyclePhase $phase, MessageContext $context): array
    {
        $messages = $this->getTTCMessages();
        $phaseMessages = $messages[$phase->value] ?? $messages['default'];

        $isFertile = $context->isFertileWindow ?? false;

        // Add fertility-specific info for TTC users
        $fertilityInfo = null;
        if ($isFertile) {
            $fertilityInfo = $this->locale === 'fa'
                ? 'شما در پنجره باروری هستید - بهترین زمان برای باردار شدن!'
                : 'You are in your fertile window - best time to conceive!';
        }

        return [
            'phase' => $phase->value,
            'phase_label' => $phase->label($this->locale),
            'cycle_day' => $context->cycleDay,
            'is_fertile_window' => $isFertile,
            'fertility_info' => $fertilityInfo,
            'short_message' => $phaseMessages['short'][$this->locale] ?? $phaseMessages['short']['fa'],
            'long_message' => $phaseMessages['long'][$this->locale] ?? $phaseMessages['long']['fa'],
            'action_suggestion' => $phaseMessages['action'][$this->locale] ?? $phaseMessages['action']['fa'],
            'dos' => $phaseMessages['dos'][$this->locale] ?? $phaseMessages['dos']['fa'],
            'donts' => $phaseMessages['donts'][$this->locale] ?? $phaseMessages['donts']['fa'],
            'ttc_tips' => $phaseMessages['ttc_tips'][$this->locale] ?? $phaseMessages['ttc_tips']['fa'] ?? [],
        ];
    }

    /**
     * Get override message for specific type
     */
    private function getOverrideMessageForType(OverrideType $type, ?CyclePhase $phase, MessageContext $context): array
    {
        $overrides = $this->getOverrideMessages();
        $override = $overrides[$type->value] ?? null;

        if (!$override) {
            return [];
        }

        return [
            'override_type' => $type->value,
            'override_label' => $type->label($this->locale),
            'override_message' => $override['message'][$this->locale] ?? $override['message']['fa'],
            'override_action' => $override['action'][$this->locale] ?? $override['action']['fa'],
            'override_dos' => $override['dos'][$this->locale] ?? $override['dos']['fa'],
            'override_donts' => $override['donts'][$this->locale] ?? $override['donts']['fa'],
        ];
    }

    /**
     * Get default message when no phase is available
     */
    private function getDefaultMessage(): array
    {
        return [
            'phase' => null,
            'phase_label' => null,
            'short_message' => $this->locale === 'fa'
                ? 'به سلامت خود توجه کنید'
                : 'Take care of your health',
            'long_message' => $this->locale === 'fa'
                ? 'برای دریافت پیام‌های شخصی‌سازی شده، لطفاً تاریخ آخرین پریود خود را وارد کنید.'
                : 'Please enter your last period date to receive personalized messages.',
            'action_suggestion' => $this->locale === 'fa'
                ? 'پروفایل خود را تکمیل کنید'
                : 'Complete your profile',
            'dos' => [],
            'donts' => [],
        ];
    }

    /**
     * Non-TTC message database
     */
    private function getNonTTCMessages(): array
    {
        return [
            'menstruation' => [
                'short' => [
                    'fa' => 'امروز روز استراحت و مراقبت از خودته',
                    'en' => 'Today is a day for rest and self-care',
                ],
                'long' => [
                    'fa' => 'در دوران قاعدگی، بدنت نیاز به استراحت و توجه بیشتری داره. به خودت اجازه بده کمی آروم‌تر باشی و از فعالیت‌های سنگین اجتناب کنی.',
                    'en' => 'During menstruation, your body needs more rest and attention. Allow yourself to slow down and avoid heavy activities.',
                ],
                'action' => [
                    'fa' => 'یک نوشیدنی گرم بنوش و استراحت کن',
                    'en' => 'Have a warm drink and rest',
                ],
                'dos' => [
                    'fa' => ['استراحت کافی', 'نوشیدن آب فراوان', 'مصرف آهن', 'پیاده‌روی سبک'],
                    'en' => ['Get enough rest', 'Drink plenty of water', 'Take iron', 'Light walking'],
                ],
                'donts' => [
                    'fa' => ['ورزش سنگین', 'کافئین زیاد', 'غذاهای شور', 'بی‌خوابی'],
                    'en' => ['Heavy exercise', 'Too much caffeine', 'Salty foods', 'Sleep deprivation'],
                ],
            ],
            'follicular' => [
                'short' => [
                    'fa' => 'انرژیت داره برمی‌گرده! وقت شروع‌های تازه‌ست',
                    'en' => 'Your energy is coming back! Time for fresh starts',
                ],
                'long' => [
                    'fa' => 'در فاز فولیکولار، سطح استروژن بالا میره و انرژی و خلاقیتت افزایش پیدا می‌کنه. بهترین زمان برای شروع پروژه‌های جدید و تمرینات ورزشی شدیدتره.',
                    'en' => 'In the follicular phase, estrogen levels rise and your energy and creativity increase. Best time to start new projects and more intense workouts.',
                ],
                'action' => [
                    'fa' => 'یک هدف جدید برای این هفته تعیین کن',
                    'en' => 'Set a new goal for this week',
                ],
                'dos' => [
                    'fa' => ['ورزش با شدت متوسط تا بالا', 'شروع پروژه‌های جدید', 'ملاقات‌های اجتماعی', 'یادگیری مهارت جدید'],
                    'en' => ['Medium to high intensity exercise', 'Start new projects', 'Social meetings', 'Learn new skills'],
                ],
                'donts' => [
                    'fa' => ['بی‌برنامگی', 'کم‌خوابی', 'غذاهای فرآوری شده'],
                    'en' => ['Being unorganized', 'Sleep deprivation', 'Processed foods'],
                ],
            ],
            'ovulation' => [
                'short' => [
                    'fa' => 'در اوج انرژی و جذابیت هستی!',
                    'en' => 'You\'re at peak energy and attractiveness!',
                ],
                'long' => [
                    'fa' => 'زمان تخمک‌گذاری، انرژی، اعتماد به نفس و جذابیتت در بالاترین حد هستن. بهترین زمان برای ارائه‌ها، مذاکرات و فعالیت‌های اجتماعی.',
                    'en' => 'During ovulation, your energy, confidence and attractiveness are at their peak. Best time for presentations, negotiations and social activities.',
                ],
                'action' => [
                    'fa' => 'از این انرژی برای کارهای مهم استفاده کن',
                    'en' => 'Use this energy for important tasks',
                ],
                'dos' => [
                    'fa' => ['ورزش‌های قدرتی', 'ارائه و مذاکره', 'فعالیت‌های اجتماعی', 'تصمیم‌گیری‌های مهم'],
                    'en' => ['Strength training', 'Presentations and negotiations', 'Social activities', 'Important decisions'],
                ],
                'donts' => [
                    'fa' => ['از دست دادن این انرژی', 'انزوا', 'به تعویق انداختن کارها'],
                    'en' => ['Wasting this energy', 'Isolation', 'Procrastination'],
                ],
            ],
            'luteal' => [
                'short' => [
                    'fa' => 'وقت تمرکز روی خودت و کارهای نیمه‌تمام',
                    'en' => 'Time to focus on yourself and unfinished tasks',
                ],
                'long' => [
                    'fa' => 'در فاز لوتئال، انرژی کم‌کم کاهش پیدا می‌کنه. بهترین زمان برای تکمیل پروژه‌ها، مراقبت از خود و آماده شدن برای سیکل بعدی.',
                    'en' => 'In the luteal phase, energy gradually decreases. Best time to complete projects, self-care and prepare for the next cycle.',
                ],
                'action' => [
                    'fa' => 'یک کار نیمه‌تمام رو تموم کن',
                    'en' => 'Complete an unfinished task',
                ],
                'dos' => [
                    'fa' => ['یوگا و کشش', 'تکمیل پروژه‌ها', 'خواب کافی', 'غذاهای مغذی'],
                    'en' => ['Yoga and stretching', 'Complete projects', 'Adequate sleep', 'Nutritious foods'],
                ],
                'donts' => [
                    'fa' => ['شروع پروژه بزرگ', 'کافئین زیاد', 'شکر زیاد', 'فعالیت‌های استرس‌زا'],
                    'en' => ['Starting big projects', 'Too much caffeine', 'Too much sugar', 'Stressful activities'],
                ],
            ],
            'default' => [
                'short' => [
                    'fa' => 'به سلامت خود توجه کنید',
                    'en' => 'Take care of your health',
                ],
                'long' => [
                    'fa' => 'هر روز فرصتی برای مراقبت از خودته.',
                    'en' => 'Every day is an opportunity to take care of yourself.',
                ],
                'action' => [
                    'fa' => 'آب کافی بنوش',
                    'en' => 'Drink enough water',
                ],
                'dos' => [
                    'fa' => ['آب فراوان', 'خواب کافی', 'ورزش منظم'],
                    'en' => ['Plenty of water', 'Adequate sleep', 'Regular exercise'],
                ],
                'donts' => [
                    'fa' => ['استرس زیاد', 'بی‌خوابی'],
                    'en' => ['Too much stress', 'Sleep deprivation'],
                ],
            ],
        ];
    }

    /**
     * TTC message database
     */
    private function getTTCMessages(): array
    {
        return [
            'menstruation' => [
                'short' => [
                    'fa' => 'سیکل جدید، فرصت جدید',
                    'en' => 'New cycle, new opportunity',
                ],
                'long' => [
                    'fa' => 'شروع یک سیکل جدید به معنی یک شانس تازه برای بارداری است. از این زمان برای استراحت و آماده‌سازی بدنت استفاده کن.',
                    'en' => 'A new cycle means a fresh chance for pregnancy. Use this time to rest and prepare your body.',
                ],
                'action' => [
                    'fa' => 'ویتامین‌های قبل از بارداری رو ادامه بده',
                    'en' => 'Continue prenatal vitamins',
                ],
                'dos' => [
                    'fa' => ['مصرف فولیک اسید', 'استراحت کافی', 'آب فراوان'],
                    'en' => ['Take folic acid', 'Get enough rest', 'Drink plenty of water'],
                ],
                'donts' => [
                    'fa' => ['الکل', 'سیگار', 'استرس زیاد'],
                    'en' => ['Alcohol', 'Smoking', 'Too much stress'],
                ],
                'ttc_tips' => [
                    'fa' => ['این زمان خوبی برای بررسی سلامت کلی است', 'با پزشک مشورت کنید'],
                    'en' => ['Good time to check overall health', 'Consult with your doctor'],
                ],
            ],
            'follicular' => [
                'short' => [
                    'fa' => 'بدنت داره برای تخمک‌گذاری آماده می‌شه',
                    'en' => 'Your body is preparing for ovulation',
                ],
                'long' => [
                    'fa' => 'تخمدان‌ها در حال رشد فولیکول هستند. این زمان مناسبی برای افزایش فعالیت بدنی و تقویت سلامت باروری است.',
                    'en' => 'Ovaries are growing follicles. Good time to increase physical activity and strengthen fertility health.',
                ],
                'action' => [
                    'fa' => 'ورزش منظم و تغذیه سالم',
                    'en' => 'Regular exercise and healthy nutrition',
                ],
                'dos' => [
                    'fa' => ['غذاهای غنی از آنتی‌اکسیدان', 'ورزش متوسط', 'خواب ۸ ساعته'],
                    'en' => ['Antioxidant-rich foods', 'Moderate exercise', '8 hours sleep'],
                ],
                'donts' => [
                    'fa' => ['استرس', 'غذاهای فرآوری شده', 'کم‌خوابی'],
                    'en' => ['Stress', 'Processed foods', 'Sleep deprivation'],
                ],
                'ttc_tips' => [
                    'fa' => ['تست تخمک‌گذاری رو از چند روز دیگه شروع کن', 'روابط زناشویی رو از الان برنامه‌ریزی کن'],
                    'en' => ['Start ovulation tests in a few days', 'Plan intimacy from now'],
                ],
            ],
            'ovulation' => [
                'short' => [
                    'fa' => '🎯 پنجره باروری! بهترین زمان برای باردار شدن',
                    'en' => '🎯 Fertile window! Best time to conceive',
                ],
                'long' => [
                    'fa' => 'تخمک‌گذاری در حال وقوع است یا به زودی اتفاق می‌افته. این ۴۸-۷۲ ساعت بهترین شانس برای باردار شدن است.',
                    'en' => 'Ovulation is happening or will happen soon. This 48-72 hours is your best chance to conceive.',
                ],
                'action' => [
                    'fa' => 'رابطه زناشویی در این ۲-۳ روز',
                    'en' => 'Intimacy in these 2-3 days',
                ],
                'dos' => [
                    'fa' => ['رابطه هر ۱-۲ روز', 'آرامش و لذت', 'پوزیشن‌های مناسب'],
                    'en' => ['Intimacy every 1-2 days', 'Relaxation and enjoyment', 'Suitable positions'],
                ],
                'donts' => [
                    'fa' => ['استرس و فشار', 'روان‌کننده‌های معمولی', 'حمام داغ طولانی'],
                    'en' => ['Stress and pressure', 'Regular lubricants', 'Long hot baths'],
                ],
                'ttc_tips' => [
                    'fa' => ['بعد از رابطه ۱۵ دقیقه دراز بکش', 'از روان‌کننده‌های fertility-friendly استفاده کن', 'تست تخمک‌گذاری مثبت = بهترین زمان'],
                    'en' => ['Lie down 15 minutes after intimacy', 'Use fertility-friendly lubricants', 'Positive OPK = best time'],
                ],
            ],
            'luteal' => [
                'short' => [
                    'fa' => 'زمان انتظار و امید',
                    'en' => 'Time for waiting and hope',
                ],
                'long' => [
                    'fa' => 'اگر لقاح اتفاق افتاده باشه، الان جنین داره به رحم سفر می‌کنه. آرامش و صبر کلید این دوره است.',
                    'en' => 'If fertilization happened, the embryo is traveling to the uterus now. Calm and patience are key.',
                ],
                'action' => [
                    'fa' => 'آرامش و امیدواری',
                    'en' => 'Calm and hopeful',
                ],
                'dos' => [
                    'fa' => ['ادامه فولیک اسید', 'استرس کم', 'خواب کافی', 'غذای سالم'],
                    'en' => ['Continue folic acid', 'Low stress', 'Adequate sleep', 'Healthy food'],
                ],
                'donts' => [
                    'fa' => ['تست بارداری زودهنگام', 'استرس زیاد', 'ورزش سنگین', 'الکل و سیگار'],
                    'en' => ['Early pregnancy test', 'Too much stress', 'Heavy exercise', 'Alcohol and smoking'],
                ],
                'ttc_tips' => [
                    'fa' => ['تست بارداری از روز ۱۲ پس از تخمک‌گذاری', 'علائم اولیه بارداری ممکنه شبیه PMS باشن', 'صبور باش و امیدوار'],
                    'en' => ['Pregnancy test from day 12 post-ovulation', 'Early pregnancy symptoms may be similar to PMS', 'Be patient and hopeful'],
                ],
            ],
            'default' => [
                'short' => [
                    'fa' => 'در مسیر بارداری',
                    'en' => 'On the path to pregnancy',
                ],
                'long' => [
                    'fa' => 'هر روز فرصتی برای مراقبت از سلامت باروری است.',
                    'en' => 'Every day is an opportunity to care for your fertility health.',
                ],
                'action' => [
                    'fa' => 'سبک زندگی سالم',
                    'en' => 'Healthy lifestyle',
                ],
                'dos' => [
                    'fa' => ['فولیک اسید', 'ورزش منظم', 'تغذیه سالم'],
                    'en' => ['Folic acid', 'Regular exercise', 'Healthy nutrition'],
                ],
                'donts' => [
                    'fa' => ['الکل', 'سیگار', 'استرس'],
                    'en' => ['Alcohol', 'Smoking', 'Stress'],
                ],
                'ttc_tips' => [
                    'fa' => [],
                    'en' => [],
                ],
            ],
        ];
    }

    /**
     * Override message database
     */
    private function getOverrideMessages(): array
    {
        return [
            'pain' => [
                'message' => [
                    'fa' => 'متوجه درد هستیم. اول از همه به خودت برس.',
                    'en' => 'We notice you\'re in pain. Take care of yourself first.',
                ],
                'action' => [
                    'fa' => 'کمپرس گرم و استراحت',
                    'en' => 'Warm compress and rest',
                ],
                'dos' => [
                    'fa' => ['کمپرس گرم', 'چای زنجبیل', 'ماساژ ملایم', 'مسکن در صورت نیاز'],
                    'en' => ['Warm compress', 'Ginger tea', 'Gentle massage', 'Pain relief if needed'],
                ],
                'donts' => [
                    'fa' => ['فعالیت سنگین', 'غذاهای سرد', 'استرس'],
                    'en' => ['Heavy activity', 'Cold foods', 'Stress'],
                ],
            ],
            'mood_sad' => [
                'message' => [
                    'fa' => 'احساسات منفی طبیعی هستن. با خودت مهربان باش.',
                    'en' => 'Negative feelings are normal. Be kind to yourself.',
                ],
                'action' => [
                    'fa' => 'با یک دوست صحبت کن یا پیاده‌روی کن',
                    'en' => 'Talk to a friend or take a walk',
                ],
                'dos' => [
                    'fa' => ['صحبت با دوست', 'پیاده‌روی در طبیعت', 'موسیقی آرام', 'نور آفتاب'],
                    'en' => ['Talk to a friend', 'Walk in nature', 'Calm music', 'Sunlight'],
                ],
                'donts' => [
                    'fa' => ['انزوا', 'شبکه‌های اجتماعی زیاد', 'قهوه زیاد'],
                    'en' => ['Isolation', 'Too much social media', 'Too much coffee'],
                ],
            ],
            'mood_angry' => [
                'message' => [
                    'fa' => 'عصبانیت هم بخشی از چرخه هورمونی است.',
                    'en' => 'Anger is also part of the hormonal cycle.',
                ],
                'action' => [
                    'fa' => 'تنفس عمیق و فاصله گرفتن',
                    'en' => 'Deep breathing and taking distance',
                ],
                'dos' => [
                    'fa' => ['تنفس عمیق', 'ورزش', 'نوشتن احساسات', 'فاصله گرفتن'],
                    'en' => ['Deep breathing', 'Exercise', 'Write feelings', 'Take distance'],
                ],
                'donts' => [
                    'fa' => ['تصمیم‌گیری مهم', 'بحث کردن', 'کافئین'],
                    'en' => ['Important decisions', 'Arguments', 'Caffeine'],
                ],
            ],
            'heavy_flow' => [
                'message' => [
                    'fa' => 'خونریزی شدید می‌تونه خسته‌کننده باشه. آهن بیشتر مصرف کن.',
                    'en' => 'Heavy flow can be tiring. Consume more iron.',
                ],
                'action' => [
                    'fa' => 'غذاهای غنی از آهن و استراحت',
                    'en' => 'Iron-rich foods and rest',
                ],
                'dos' => [
                    'fa' => ['گوشت قرمز', 'اسفناج', 'ویتامین C', 'آب فراوان'],
                    'en' => ['Red meat', 'Spinach', 'Vitamin C', 'Plenty of water'],
                ],
                'donts' => [
                    'fa' => ['چای و قهوه با غذا', 'ورزش سنگین'],
                    'en' => ['Tea and coffee with meals', 'Heavy exercise'],
                ],
            ],
            'low_energy' => [
                'message' => [
                    'fa' => 'انرژی کم طبیعیه. به بدنت گوش بده.',
                    'en' => 'Low energy is normal. Listen to your body.',
                ],
                'action' => [
                    'fa' => 'استراحت کوتاه و میان‌وعده سالم',
                    'en' => 'Short rest and healthy snack',
                ],
                'dos' => [
                    'fa' => ['استراحت کوتاه', 'میوه و آجیل', 'آب', 'پیاده‌روی کوتاه'],
                    'en' => ['Short rest', 'Fruit and nuts', 'Water', 'Short walk'],
                ],
                'donts' => [
                    'fa' => ['قهوه زیاد', 'شیرینی', 'کار سنگین'],
                    'en' => ['Too much coffee', 'Sweets', 'Heavy work'],
                ],
            ],
            'bloating' => [
                'message' => [
                    'fa' => 'نفخ یکی از علائم شایع هورمونی است.',
                    'en' => 'Bloating is a common hormonal symptom.',
                ],
                'action' => [
                    'fa' => 'چای نعناع و پیاده‌روی',
                    'en' => 'Peppermint tea and walking',
                ],
                'dos' => [
                    'fa' => ['چای نعناع', 'پیاده‌روی', 'غذای کم نمک', 'آب فراوان'],
                    'en' => ['Peppermint tea', 'Walking', 'Low-salt food', 'Plenty of water'],
                ],
                'donts' => [
                    'fa' => ['نوشابه', 'غذای شور', 'لبنیات زیاد'],
                    'en' => ['Soda', 'Salty food', 'Too much dairy'],
                ],
            ],
            'acne' => [
                'message' => [
                    'fa' => 'جوش‌های هورمونی موقتی هستن.',
                    'en' => 'Hormonal acne is temporary.',
                ],
                'action' => [
                    'fa' => 'پاکسازی ملایم پوست',
                    'en' => 'Gentle skin cleansing',
                ],
                'dos' => [
                    'fa' => ['شستشوی ملایم', 'آب فراوان', 'میوه و سبزی'],
                    'en' => ['Gentle washing', 'Plenty of water', 'Fruits and vegetables'],
                ],
                'donts' => [
                    'fa' => ['دست زدن به صورت', 'شیرینی زیاد', 'آرایش سنگین'],
                    'en' => ['Touching face', 'Too much sugar', 'Heavy makeup'],
                ],
            ],
        ];
    }
}
