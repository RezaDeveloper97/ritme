<?php

namespace App\Services\MessageSystem\Engines;

use App\Models\User;
use App\Services\MessageSystem\Contracts\MessageEngineInterface;
use App\Services\MessageSystem\Contracts\ProvidesMessageContent;
use App\Services\MessageSystem\Core\MessageContext;
use App\Services\MessageSystem\Core\MessageResult;
use App\Services\MessageSystem\Enums\MessageMode;
use App\Services\MessageSystem\Support\MessageContentRepository;

class PregnancyMessageEngine implements MessageEngineInterface, ProvidesMessageContent
{
    public function __construct(
        private readonly User $user,
        private readonly string $locale = 'fa',
    ) {}

    public function getMode(): string
    {
        return MessageMode::PREGNANCY->value;
    }

    public function canHandle(MessageContext $context): bool
    {
        return $context->isPregnancyMode();
    }

    public function generateMessages(MessageContext $context): MessageResult
    {
        throw new \RuntimeException('Use MessageManager::generateMessages() instead');
    }

    public function getEnums(string $locale = 'en'): array
    {
        return [
            'trimesters' => [
                ['value' => 1, 'label' => $locale === 'fa' ? 'سه‌ماهه اول' : 'First Trimester', 'weeks' => '1-13'],
                ['value' => 2, 'label' => $locale === 'fa' ? 'سه‌ماهه دوم' : 'Second Trimester', 'weeks' => '14-27'],
                ['value' => 3, 'label' => $locale === 'fa' ? 'سه‌ماهه سوم' : 'Third Trimester', 'weeks' => '28-40'],
            ],
        ];
    }

    private function content(): MessageContentRepository
    {
        return app(MessageContentRepository::class);
    }

    /**
     * Get base message for the current pregnancy week (Layer 1)
     */
    public function getBaseMessage(MessageContext $context): array
    {
        $week = $context->pregnancyWeek;
        $trimester = $context->trimester;

        if (!$week) {
            return $this->getDefaultMessage();
        }

        // Get trimester-based message as base
        $trimesterMessage = $this->getTrimesterMessage($trimester ?? 1);

        // Get week-specific additions
        $weekSpecific = $this->getWeekSpecificMessage($week);

        return [
            'week' => $week,
            'day' => $context->pregnancyDay,
            'trimester' => $trimester,
            'trimester_label' => $this->getTrimesterLabel($trimester ?? 1),
            'gestational_age' => $context->getGestationalAgeString(),
            'short_message' => $weekSpecific['short'] ?? $trimesterMessage['short'] ?? '',
            'long_message' => $weekSpecific['long'] ?? $trimesterMessage['long'] ?? '',
            'action_suggestion' => $weekSpecific['action'] ?? $trimesterMessage['action'] ?? '',
            'dos' => $trimesterMessage['dos'] ?? [],
            'donts' => $trimesterMessage['donts'] ?? [],
            'baby_development' => $weekSpecific['baby'] ?? null,
            'body_changes' => $weekSpecific['body'] ?? null,
        ];
    }

    /**
     * Get override message based on pregnancy symptoms (Layer 2)
     */
    public function getOverrideMessage(MessageContext $context): ?array
    {
        if (empty($context->symptoms)) {
            return null;
        }

        // Check for pregnancy-specific overrides
        $override = $this->determinePregnancyOverride($context->symptoms, $context->trimester ?? 1);

        if (!$override) {
            return null;
        }

        return $override;
    }

    /**
     * Determine pregnancy-specific override
     */
    private function determinePregnancyOverride(array $symptoms, int $trimester): ?array
    {
        // Check for severe symptoms first
        if (in_array('nausea', $symptoms) || in_array('vomiting', $symptoms)) {
            return $this->getOverrideForType('nausea');
        }

        if (in_array('fatigue', $symptoms) || in_array('low_energy', $symptoms)) {
            return $this->getOverrideForType('fatigue');
        }

        if (in_array('backache', $symptoms)) {
            return $this->getOverrideForType('backache');
        }

        if (in_array('mood_anxious', $symptoms) || in_array('mood_sad', $symptoms)) {
            return $this->getOverrideForType('anxiety');
        }

        return null;
    }

    private function getTrimesterLabel(int $trimester): string
    {
        return match ($trimester) {
            1 => $this->locale === 'fa' ? 'سه‌ماهه اول' : 'First Trimester',
            2 => $this->locale === 'fa' ? 'سه‌ماهه دوم' : 'Second Trimester',
            3 => $this->locale === 'fa' ? 'سه‌ماهه سوم' : 'Third Trimester',
            default => $this->locale === 'fa' ? 'نامشخص' : 'Unknown',
        };
    }

    /**
     * Get default message when no pregnancy week is available (DB-editable, falls back to defaults)
     */
    private function getDefaultMessage(): array
    {
        $fallback = self::slice(self::baseMessages()['default'], $this->locale);
        $p = $this->content()->resolve('pregnancy_base', 'default', $this->locale, $fallback);

        return [
            'week' => null,
            'trimester' => null,
            'short_message' => $p['short'] ?? '',
            'long_message' => $p['long'] ?? '',
            'action_suggestion' => $p['action'] ?? '',
            'dos' => [],
            'donts' => [],
        ];
    }

    /**
     * Get trimester-based message (DB-editable, falls back to defaults)
     */
    private function getTrimesterMessage(int $trimester): array
    {
        $messages = self::trimesterMessages();
        $key = array_key_exists($trimester, $messages) ? $trimester : 1;
        $fallback = self::slice($messages[$key], $this->locale);

        return $this->content()->resolve('pregnancy_trimester', (string) $key, $this->locale, $fallback);
    }

    /**
     * Get week-specific milestone message (DB-editable, falls back to defaults)
     */
    private function getWeekSpecificMessage(int $week): array
    {
        $milestones = self::weekMilestones();

        // Find closest milestone or return empty
        $closest = null;
        $minDiff = 100;
        foreach (array_keys($milestones) as $milestone) {
            $diff = abs($week - $milestone);
            if ($diff < $minDiff && $diff <= 2) {
                $minDiff = $diff;
                $closest = $milestone;
            }
        }

        if ($closest !== null) {
            $fallback = self::slice($milestones[$closest], $this->locale);

            return $this->content()->resolve('pregnancy_week', (string) $closest, $this->locale, $fallback);
        }

        return [
            'short' => $this->locale === 'fa' ? "هفته {$week} بارداری" : "Week {$week} of pregnancy",
            'long' => null,
            'action' => null,
            'baby' => null,
            'body' => null,
        ];
    }

    /**
     * Get pregnancy override message for specific type (DB-editable, falls back to defaults)
     */
    private function getOverrideForType(string $type): ?array
    {
        $overrides = self::overrideMessages();
        if (!isset($overrides[$type])) {
            return null;
        }

        $fallback = self::slice($overrides[$type], $this->locale);
        $p = $this->content()->resolve('pregnancy_override', $type, $this->locale, $fallback);

        return [
            'override_type' => $type,
            'override_label' => $p['label'] ?? '',
            'override_message' => $p['message'] ?? '',
            'override_action' => $p['action'] ?? '',
            'override_dos' => $p['dos'] ?? [],
            'override_donts' => $p['donts'] ?? [],
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
     * @inheritDoc — seed/fallback source for all pregnancy message text.
     */
    public static function contentDefaults(): array
    {
        $out = [];
        foreach (self::trimesterMessages() as $key => $entry) {
            foreach (['fa', 'en'] as $loc) {
                $out['pregnancy_trimester'][(string) $key][$loc] = self::slice($entry, $loc);
            }
        }
        foreach (self::weekMilestones() as $key => $entry) {
            foreach (['fa', 'en'] as $loc) {
                $out['pregnancy_week'][(string) $key][$loc] = self::slice($entry, $loc);
            }
        }
        foreach (self::overrideMessages() as $key => $entry) {
            foreach (['fa', 'en'] as $loc) {
                $out['pregnancy_override'][(string) $key][$loc] = self::slice($entry, $loc);
            }
        }
        foreach (self::baseMessages() as $key => $entry) {
            foreach (['fa', 'en'] as $loc) {
                $out['pregnancy_base'][(string) $key][$loc] = self::slice($entry, $loc);
            }
        }
        return $out;
    }

    /**
     * Trimester message database (seed/fallback)
     */
    private static function trimesterMessages(): array
    {
        return [
            1 => [
                'short' => [
                    'fa' => 'سه‌ماهه اول: زمان تشکیل پایه‌ها',
                    'en' => 'First Trimester: Foundation building time',
                ],
                'long' => [
                    'fa' => 'در این دوره، اندام‌های حیاتی نوزاد در حال شکل‌گیری هستند. استراحت کافی و تغذیه سالم بسیار مهم است.',
                    'en' => 'During this period, vital organs are forming. Adequate rest and healthy nutrition are crucial.',
                ],
                'action' => [
                    'fa' => 'فولیک اسید و ویتامین‌های قبل از زایمان را مصرف کنید',
                    'en' => 'Take folic acid and prenatal vitamins',
                ],
                'dos' => [
                    'fa' => ['فولیک اسید', 'استراحت کافی', 'غذاهای کوچک و مکرر', 'آب فراوان'],
                    'en' => ['Folic acid', 'Adequate rest', 'Small frequent meals', 'Plenty of water'],
                ],
                'donts' => [
                    'fa' => ['الکل', 'سیگار', 'غذاهای خام', 'داروهای بدون مشورت پزشک'],
                    'en' => ['Alcohol', 'Smoking', 'Raw foods', 'Medications without doctor consultation'],
                ],
            ],
            2 => [
                'short' => [
                    'fa' => 'سه‌ماهه دوم: دوران طلایی بارداری',
                    'en' => 'Second Trimester: Golden period of pregnancy',
                ],
                'long' => [
                    'fa' => 'انرژی برگشته و تهوع‌ها کمتر شده. زمان مناسبی برای ورزش ملایم و لذت بردن از بارداری است.',
                    'en' => 'Energy is back and nausea is less. Good time for gentle exercise and enjoying pregnancy.',
                ],
                'action' => [
                    'fa' => 'شنا یا یوگای بارداری را امتحان کنید',
                    'en' => 'Try swimming or prenatal yoga',
                ],
                'dos' => [
                    'fa' => ['ورزش ملایم', 'پروتئین کافی', 'آهن و کلسیم', 'سونوگرافی آنومالی'],
                    'en' => ['Gentle exercise', 'Adequate protein', 'Iron and calcium', 'Anomaly ultrasound'],
                ],
                'donts' => [
                    'fa' => ['دراز کشیدن به پشت طولانی', 'ورزش سنگین', 'سونا و جکوزی'],
                    'en' => ['Lying on back for long', 'Heavy exercise', 'Sauna and jacuzzi'],
                ],
            ],
            3 => [
                'short' => [
                    'fa' => 'سه‌ماهه سوم: آماده‌سازی برای ملاقات',
                    'en' => 'Third Trimester: Preparing for the meeting',
                ],
                'long' => [
                    'fa' => 'نوزاد در حال رشد نهایی است و شما برای زایمان آماده می‌شوید. استراحت و آمادگی ذهنی مهم است.',
                    'en' => 'Baby is in final growth and you are preparing for delivery. Rest and mental preparation are important.',
                ],
                'action' => [
                    'fa' => 'کلاس آمادگی زایمان و ساک بیمارستان',
                    'en' => 'Birth preparation class and hospital bag',
                ],
                'dos' => [
                    'fa' => ['شمارش حرکات جنین', 'استراحت کافی', 'تمرینات تنفسی', 'آماده‌سازی وسایل نوزاد'],
                    'en' => ['Count fetal movements', 'Adequate rest', 'Breathing exercises', 'Prepare baby supplies'],
                ],
                'donts' => [
                    'fa' => ['ایستادن طولانی', 'بلند کردن اجسام سنگین', 'مسافرت طولانی'],
                    'en' => ['Standing for long', 'Lifting heavy objects', 'Long travels'],
                ],
            ],
        ];
    }

    /**
     * Week milestone message database (seed/fallback)
     */
    private static function weekMilestones(): array
    {
        return [
            4 => [
                'short' => ['fa' => 'جنین به دیواره رحم چسبیده است', 'en' => 'Embryo has implanted'],
                'baby' => ['fa' => 'جنین به اندازه دانه خشخاش است', 'en' => 'Embryo is poppy seed sized'],
                'body' => ['fa' => 'ممکن است لکه‌بینی خفیف داشته باشید', 'en' => 'You may have light spotting'],
            ],
            8 => [
                'short' => ['fa' => 'قلب نوزاد شروع به تپیدن کرده', 'en' => 'Baby\'s heart has started beating'],
                'baby' => ['fa' => 'قلب با ۱۵۰-۱۷۰ ضربه در دقیقه می‌تپد', 'en' => 'Heart beats 150-170 per minute'],
                'body' => ['fa' => 'تهوع صبحگاهی در اوج است', 'en' => 'Morning sickness is at its peak'],
            ],
            12 => [
                'short' => ['fa' => 'پایان سه‌ماهه اول', 'en' => 'End of first trimester'],
                'baby' => ['fa' => 'اندام‌های حیاتی شکل گرفته‌اند', 'en' => 'Vital organs have formed'],
                'body' => ['fa' => 'تهوع کمتر می‌شود', 'en' => 'Nausea is decreasing'],
            ],
            20 => [
                'short' => ['fa' => 'نیمه راه! سونوگرافی آنومالی', 'en' => 'Halfway! Anomaly scan'],
                'baby' => ['fa' => 'می‌توانید جنسیت را بدانید', 'en' => 'You can know the gender'],
                'body' => ['fa' => 'حرکات جنین را احساس می‌کنید', 'en' => 'You feel fetal movements'],
            ],
            28 => [
                'short' => ['fa' => 'شروع سه‌ماهه سوم', 'en' => 'Start of third trimester'],
                'baby' => ['fa' => 'چشم‌ها باز می‌شوند', 'en' => 'Eyes are opening'],
                'body' => ['fa' => 'ممکن است انقباضات تمرینی داشته باشید', 'en' => 'You may have practice contractions'],
            ],
            36 => [
                'short' => ['fa' => 'نوزاد در حال آماده شدن است', 'en' => 'Baby is getting ready'],
                'baby' => ['fa' => 'ریه‌ها تقریباً کامل شده‌اند', 'en' => 'Lungs are almost complete'],
                'body' => ['fa' => 'نوزاد به سمت پایین می‌چرخد', 'en' => 'Baby turns head down'],
            ],
            40 => [
                'short' => ['fa' => 'موعد مقرر! آماده ملاقات باشید', 'en' => 'Due date! Be ready to meet'],
                'baby' => ['fa' => 'نوزاد آماده تولد است', 'en' => 'Baby is ready to be born'],
                'body' => ['fa' => 'علائم زایمان را بشناسید', 'en' => 'Know the signs of labor'],
            ],
        ];
    }

    /**
     * Pregnancy override message database (seed/fallback)
     */
    private static function overrideMessages(): array
    {
        return [
            'nausea' => [
                'label' => [
                    'fa' => 'تهوع',
                    'en' => 'Nausea',
                ],
                'message' => [
                    'fa' => 'تهوع بارداری ناراحت‌کننده است اما معمولاً نشانه بارداری سالم است.',
                    'en' => 'Pregnancy nausea is uncomfortable but usually a sign of healthy pregnancy.',
                ],
                'action' => [
                    'fa' => 'غذاهای کوچک و مکرر، زنجبیل',
                    'en' => 'Small frequent meals, ginger',
                ],
                'dos' => [
                    'fa' => ['غذای کوچک قبل از بلند شدن', 'زنجبیل', 'لیمو', 'غذاهای خشک'],
                    'en' => ['Small meal before getting up', 'Ginger', 'Lemon', 'Dry foods'],
                ],
                'donts' => [
                    'fa' => ['بوی قوی', 'غذای چرب', 'معده خالی'],
                    'en' => ['Strong smells', 'Fatty food', 'Empty stomach'],
                ],
            ],
            'fatigue' => [
                'label' => [
                    'fa' => 'خستگی',
                    'en' => 'Fatigue',
                ],
                'message' => [
                    'fa' => 'خستگی در بارداری طبیعی است. بدن شما سخت کار می‌کند.',
                    'en' => 'Fatigue in pregnancy is normal. Your body is working hard.',
                ],
                'action' => [
                    'fa' => 'استراحت بیشتر و خواب کافی',
                    'en' => 'More rest and adequate sleep',
                ],
                'dos' => [
                    'fa' => ['چرت کوتاه', 'خواب زود', 'آهن کافی'],
                    'en' => ['Short naps', 'Early sleep', 'Adequate iron'],
                ],
                'donts' => [
                    'fa' => ['بی‌خوابی', 'کار زیاد', 'استرس'],
                    'en' => ['Sleep deprivation', 'Overwork', 'Stress'],
                ],
            ],
            'backache' => [
                'label' => [
                    'fa' => 'کمردرد',
                    'en' => 'Back Pain',
                ],
                'message' => [
                    'fa' => 'کمردرد با رشد شکم شایع می‌شود. وضعیت بدنی درست کمک می‌کند.',
                    'en' => 'Back pain becomes common as belly grows. Good posture helps.',
                ],
                'action' => [
                    'fa' => 'کمربند بارداری و تمرینات کششی',
                    'en' => 'Pregnancy belt and stretching exercises',
                ],
                'dos' => [
                    'fa' => ['کمربند بارداری', 'شنا', 'ماساژ', 'کفش راحت'],
                    'en' => ['Pregnancy belt', 'Swimming', 'Massage', 'Comfortable shoes'],
                ],
                'donts' => [
                    'fa' => ['کفش پاشنه بلند', 'ایستادن طولانی', 'بلند کردن سنگین'],
                    'en' => ['High heels', 'Standing long', 'Heavy lifting'],
                ],
            ],
            'anxiety' => [
                'label' => [
                    'fa' => 'اضطراب',
                    'en' => 'Anxiety',
                ],
                'message' => [
                    'fa' => 'اضطراب در بارداری شایع است. صحبت کردن کمک می‌کند.',
                    'en' => 'Anxiety in pregnancy is common. Talking helps.',
                ],
                'action' => [
                    'fa' => 'تنفس عمیق و صحبت با کسی',
                    'en' => 'Deep breathing and talking to someone',
                ],
                'dos' => [
                    'fa' => ['تنفس عمیق', 'یوگا', 'صحبت با همسر', 'کلاس آمادگی'],
                    'en' => ['Deep breathing', 'Yoga', 'Talk to partner', 'Preparation class'],
                ],
                'donts' => [
                    'fa' => ['خبرهای منفی', 'انزوا', 'کافئین زیاد'],
                    'en' => ['Negative news', 'Isolation', 'Too much caffeine'],
                ],
            ],
        ];
    }

    /**
     * Base/default message database (seed/fallback)
     */
    private static function baseMessages(): array
    {
        return [
            'default' => [
                'short' => [
                    'fa' => 'به دوران بارداری خوش آمدید',
                    'en' => 'Welcome to your pregnancy journey',
                ],
                'long' => [
                    'fa' => 'برای دریافت پیام‌های شخصی‌سازی شده، لطفاً اطلاعات بارداری را تکمیل کنید.',
                    'en' => 'Please complete your pregnancy information to receive personalized messages.',
                ],
                'action' => [
                    'fa' => 'پروفایل بارداری را تکمیل کنید',
                    'en' => 'Complete your pregnancy profile',
                ],
            ],
        ];
    }
}
