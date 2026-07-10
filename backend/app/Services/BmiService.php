<?php

namespace App\Services;

use App\Enums\BmiCategory;
use App\Models\UserProfile;
use App\Services\MessageSystem\Contracts\ProvidesMessageContent;
use App\Services\MessageSystem\Support\MessageContentRepository;

/**
 * Computes the body-mass index from a profile's height (cm) and weight (kg) and
 * pairs it with a supportive, band-specific message. The message text is
 * admin-editable via the message_contents table (group `bmi_message`, item_key
 * = category, locale); {@see contentDefaults()} is both the seed and the runtime
 * fallback so the feature works before any row is edited.
 */
class BmiService implements ProvidesMessageContent
{
    private const GROUP = 'bmi_message';

    public function __construct(
        private readonly MessageContentRepository $content = new MessageContentRepository,
    ) {}

    /**
     * Full BMI payload for a profile, or null when height/weight are missing so
     * the caller can omit the section entirely.
     *
     * @return array{value: float, category: string, category_label: string, message: string}|null
     */
    public function forProfile(UserProfile $profile, string $locale = 'fa'): ?array
    {
        // Classify from the raw ratio so a value that only rounds onto a band
        // boundary (e.g. 24.96 -> displayed 25.0) keeps its true category.
        $raw = self::rawBmi($profile->weight, $profile->height);

        if ($raw === null) {
            return null;
        }

        $category = BmiCategory::fromBmi($raw);

        return [
            'value' => round($raw, 1),
            'category' => $category->value,
            'category_label' => $category->label($locale),
            'message' => $this->messageFor($category, $locale),
        ];
    }

    /**
     * BMI rounded to one decimal, or null when either metric is absent or
     * non-positive (guards divide-by-zero and bogus stored values).
     */
    public static function calculate(?float $weightKg, ?int $heightCm): ?float
    {
        $raw = self::rawBmi($weightKg, $heightCm);

        return $raw === null ? null : round($raw, 1);
    }

    /** Unrounded BMI ratio, or null when a metric is missing/non-positive. */
    private static function rawBmi(?float $weightKg, ?int $heightCm): ?float
    {
        if (! $weightKg || ! $heightCm || $weightKg <= 0 || $heightCm <= 0) {
            return null;
        }

        $heightM = $heightCm / 100;

        return $weightKg / ($heightM * $heightM);
    }

    /** Admin-editable message for a band, falling back to the seeded default. */
    private function messageFor(BmiCategory $category, string $locale): string
    {
        $fallback = self::messageDefaults()[$category->value][$locale]
            ?? self::messageDefaults()[$category->value]['fa'];

        $payload = $this->content->resolve(self::GROUP, $category->value, $locale, $fallback);

        return $payload['message'] ?? $fallback['message'];
    }

    /**
     * @return array<string, array<string, array<string, mixed>>>
     */
    public static function contentDefaults(): array
    {
        return [self::GROUP => self::messageDefaults()];
    }

    /**
     * The canonical band messages. Seeded into message_contents and used as the
     * runtime fallback. Keyed by category value then locale.
     *
     * @return array<string, array<string, array{message: string}>>
     */
    private static function messageDefaults(): array
    {
        return [
            BmiCategory::UNDERWEIGHT->value => [
                'fa' => ['message' => 'بر اساس قد و وزن وارد شده، در محدوده کم‌وزن قرار می‌گیری. اگر این وضعیت برای مدت طولانی ادامه داشته باشد، می‌تواند روی انرژی، خلق و سیکل قاعدگی‌ات اثر بگذارد. اگر نگران هستی، بهتر است با پزشک یا کارشناس تغذیه صحبت کنی.'],
                'en' => ['message' => 'Based on the height and weight you entered, you fall in the underweight range. If this continues for a long time, it can affect your energy, mood, and menstrual cycle. If you are concerned, it is best to talk to a doctor or a nutrition specialist.'],
            ],
            BmiCategory::NORMAL->value => [
                'fa' => ['message' => 'بر اساس قد و وزن وارد شده، در محدوده‌ی وزنی طبیعی قرار می‌گیری. این محدوده معمولاً برای سلامت عمومی و سیکل قاعدگی مناسب است. Ritme تلاش می‌کند به حفظ این وضعیت کمک کند.'],
                'en' => ['message' => 'Based on the height and weight you entered, you are in the normal weight range. This range is usually good for overall health and your menstrual cycle. Ritme aims to help you maintain it.'],
            ],
            BmiCategory::OVERWEIGHT->value => [
                'fa' => ['message' => 'بر اساس قد و وزن وارد شده، در محدوده‌ی اضافه‌وزن قرار می‌گیری. این وضعیت می‌تواند در بعضی افراد روی انرژی، خواب و سیکل قاعدگی (به‌خصوص در PCOS) اثر بگذارد. تغییرات کوچک و پایدار در فعالیت و تغذیه، بیشتر از رژیم‌های سخت به سلامتت کمک می‌کنند.'],
                'en' => ['message' => 'Based on the height and weight you entered, you fall in the overweight range. In some people this can affect energy, sleep, and the menstrual cycle (especially with PCOS). Small, sustainable changes in activity and nutrition help your health more than strict diets.'],
            ],
            BmiCategory::OBESE->value => [
                'fa' => ['message' => 'بر اساس قد و وزن وارد شده، در محدوده‌ی چاقی قرار می‌گیری. این می‌تواند در طولانی‌مدت روی سلامت قلب، فشار خون، قند و سیکل قاعدگی اثر بگذارد. اگر امکانش را داری، صحبت با پزشک یا کارشناس تغذیه می‌تواند خیلی کمک‌کننده باشد. در Ritme سعی می‌کنیم با توصیه‌های کوچک قابل‌اجرا، به روند سلامتت کمک کنیم.'],
                'en' => ['message' => 'Based on the height and weight you entered, you fall in the obesity range. Over the long term this can affect heart health, blood pressure, blood sugar, and the menstrual cycle. If possible, talking to a doctor or a nutrition specialist can be very helpful. At Ritme we try to support your health journey with small, doable recommendations.'],
            ],
        ];
    }
}
