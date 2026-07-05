<?php

namespace App\Services\MessageSystem\Engines;

use App\Models\User;
use App\Services\MessageSystem\Contracts\MessageEngineInterface;
use App\Services\MessageSystem\Core\MessageContext;
use App\Services\MessageSystem\Core\MessageResult;
use App\Services\MessageSystem\Enums\MessageMode;

class PregnancyMessageEngine implements MessageEngineInterface
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
            'short_message' => $weekSpecific['short'][$this->locale] ?? $trimesterMessage['short'][$this->locale],
            'long_message' => $weekSpecific['long'][$this->locale] ?? $trimesterMessage['long'][$this->locale],
            'action_suggestion' => $weekSpecific['action'][$this->locale] ?? $trimesterMessage['action'][$this->locale],
            'dos' => $trimesterMessage['dos'][$this->locale] ?? [],
            'donts' => $trimesterMessage['donts'][$this->locale] ?? [],
            'baby_development' => $weekSpecific['baby'][$this->locale] ?? null,
            'body_changes' => $weekSpecific['body'][$this->locale] ?? null,
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
        $overrides = $this->getPregnancyOverrides();

        // Check for severe symptoms first
        if (in_array('nausea', $symptoms) || in_array('vomiting', $symptoms)) {
            return $overrides['nausea'] ?? null;
        }

        if (in_array('fatigue', $symptoms) || in_array('low_energy', $symptoms)) {
            return $overrides['fatigue'] ?? null;
        }

        if (in_array('backache', $symptoms)) {
            return $overrides['backache'] ?? null;
        }

        if (in_array('mood_anxious', $symptoms) || in_array('mood_sad', $symptoms)) {
            return $overrides['anxiety'] ?? null;
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

    private function getDefaultMessage(): array
    {
        return [
            'week' => null,
            'trimester' => null,
            'short_message' => $this->locale === 'fa'
                ? 'به دوران بارداری خوش آمدید'
                : 'Welcome to your pregnancy journey',
            'long_message' => $this->locale === 'fa'
                ? 'برای دریافت پیام‌های شخصی‌سازی شده، لطفاً اطلاعات بارداری را تکمیل کنید.'
                : 'Please complete your pregnancy information to receive personalized messages.',
            'action_suggestion' => $this->locale === 'fa'
                ? 'پروفایل بارداری را تکمیل کنید'
                : 'Complete your pregnancy profile',
            'dos' => [],
            'donts' => [],
        ];
    }

    private function getTrimesterMessage(int $trimester): array
    {
        $messages = [
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

        return $messages[$trimester] ?? $messages[1];
    }

    private function getWeekSpecificMessage(int $week): array
    {
        // Key milestone weeks
        $milestones = [
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

        return $milestones[$closest] ?? [
            'short' => ['fa' => "هفته {$week} بارداری", 'en' => "Week {$week} of pregnancy"],
            'long' => ['fa' => null, 'en' => null],
            'action' => ['fa' => null, 'en' => null],
            'baby' => ['fa' => null, 'en' => null],
            'body' => ['fa' => null, 'en' => null],
        ];
    }

    private function getPregnancyOverrides(): array
    {
        return [
            'nausea' => [
                'override_type' => 'nausea',
                'override_label' => $this->locale === 'fa' ? 'تهوع' : 'Nausea',
                'override_message' => $this->locale === 'fa'
                    ? 'تهوع بارداری ناراحت‌کننده است اما معمولاً نشانه بارداری سالم است.'
                    : 'Pregnancy nausea is uncomfortable but usually a sign of healthy pregnancy.',
                'override_action' => $this->locale === 'fa'
                    ? 'غذاهای کوچک و مکرر، زنجبیل'
                    : 'Small frequent meals, ginger',
                'override_dos' => $this->locale === 'fa'
                    ? ['غذای کوچک قبل از بلند شدن', 'زنجبیل', 'لیمو', 'غذاهای خشک']
                    : ['Small meal before getting up', 'Ginger', 'Lemon', 'Dry foods'],
                'override_donts' => $this->locale === 'fa'
                    ? ['بوی قوی', 'غذای چرب', 'معده خالی']
                    : ['Strong smells', 'Fatty food', 'Empty stomach'],
            ],
            'fatigue' => [
                'override_type' => 'fatigue',
                'override_label' => $this->locale === 'fa' ? 'خستگی' : 'Fatigue',
                'override_message' => $this->locale === 'fa'
                    ? 'خستگی در بارداری طبیعی است. بدن شما سخت کار می‌کند.'
                    : 'Fatigue in pregnancy is normal. Your body is working hard.',
                'override_action' => $this->locale === 'fa'
                    ? 'استراحت بیشتر و خواب کافی'
                    : 'More rest and adequate sleep',
                'override_dos' => $this->locale === 'fa'
                    ? ['چرت کوتاه', 'خواب زود', 'آهن کافی']
                    : ['Short naps', 'Early sleep', 'Adequate iron'],
                'override_donts' => $this->locale === 'fa'
                    ? ['بی‌خوابی', 'کار زیاد', 'استرس']
                    : ['Sleep deprivation', 'Overwork', 'Stress'],
            ],
            'backache' => [
                'override_type' => 'backache',
                'override_label' => $this->locale === 'fa' ? 'کمردرد' : 'Back Pain',
                'override_message' => $this->locale === 'fa'
                    ? 'کمردرد با رشد شکم شایع می‌شود. وضعیت بدنی درست کمک می‌کند.'
                    : 'Back pain becomes common as belly grows. Good posture helps.',
                'override_action' => $this->locale === 'fa'
                    ? 'کمربند بارداری و تمرینات کششی'
                    : 'Pregnancy belt and stretching exercises',
                'override_dos' => $this->locale === 'fa'
                    ? ['کمربند بارداری', 'شنا', 'ماساژ', 'کفش راحت']
                    : ['Pregnancy belt', 'Swimming', 'Massage', 'Comfortable shoes'],
                'override_donts' => $this->locale === 'fa'
                    ? ['کفش پاشنه بلند', 'ایستادن طولانی', 'بلند کردن سنگین']
                    : ['High heels', 'Standing long', 'Heavy lifting'],
            ],
            'anxiety' => [
                'override_type' => 'anxiety',
                'override_label' => $this->locale === 'fa' ? 'اضطراب' : 'Anxiety',
                'override_message' => $this->locale === 'fa'
                    ? 'اضطراب در بارداری شایع است. صحبت کردن کمک می‌کند.'
                    : 'Anxiety in pregnancy is common. Talking helps.',
                'override_action' => $this->locale === 'fa'
                    ? 'تنفس عمیق و صحبت با کسی'
                    : 'Deep breathing and talking to someone',
                'override_dos' => $this->locale === 'fa'
                    ? ['تنفس عمیق', 'یوگا', 'صحبت با همسر', 'کلاس آمادگی']
                    : ['Deep breathing', 'Yoga', 'Talk to partner', 'Preparation class'],
                'override_donts' => $this->locale === 'fa'
                    ? ['خبرهای منفی', 'انزوا', 'کافئین زیاد']
                    : ['Negative news', 'Isolation', 'Too much caffeine'],
            ],
        ];
    }
}
