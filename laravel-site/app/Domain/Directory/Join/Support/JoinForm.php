<?php

declare(strict_types=1);

namespace App\Domain\Directory\Join\Support;

use App\Domain\Directory\Enums\Weekday;

/**
 * Limits and defaults of the join form, shared by the page (hints, `data-*` for the client-side checks), the
 * validation and the tests.
 *
 * Photo limits follow the server: one file may not exceed PHP's `upload_max_filesize` (capped at PHOTO_MAX_BYTES) and
 * the whole post must fit `post_max_size`, otherwise PHP drops the files before Laravel sees them.
 */
final class JoinForm
{
    public const STEPS = 4;

    public const PHOTOS_MAX = 10;

    public const PHOTOS_RECOMMENDED = 4;

    public const PHOTO_MAX_BYTES = 5 * 1024 * 1024;

    /** Kept free in the post for the text fields when computing the total photo budget. */
    private const POST_MARGIN_BYTES = 256 * 1024;

    public const PHOTO_MIMES = ['image/jpeg', 'image/png', 'image/webp'];

    public const NAME_MAX = 120;

    public const ABOUT_MAX = 600;

    public const SERVICES_MAX = 1500;

    public static function photoMaxBytes(): int
    {
        $ini = self::iniBytes((string) ini_get('upload_max_filesize'));

        return $ini > 0 ? min(self::PHOTO_MAX_BYTES, $ini) : self::PHOTO_MAX_BYTES;
    }

    /** Budget for all photos of one submission (0 = unlimited by PHP). */
    public static function photosTotalBytes(): int
    {
        $post = self::iniBytes((string) ini_get('post_max_size'));

        return $post > 0 ? max(0, $post - self::POST_MARGIN_BYTES) : 0;
    }

    /**
     * Pre-filled hours (the design's week: Saturday–Wednesday 09–20, Thursday 09–14, Friday closed).
     *
     * @return array<string, array{open: bool, opens: string, closes: string}>
     */
    public static function defaultHours(): array
    {
        $hours = [];
        foreach (Weekday::cases() as $day) {
            $hours[$day->value] = match ($day) {
                Weekday::Thursday => ['open' => true, 'opens' => '09:00', 'closes' => '14:00'],
                Weekday::Friday => ['open' => false, 'opens' => '09:00', 'closes' => '14:00'],
                default => ['open' => true, 'opens' => '09:00', 'closes' => '20:00'],
            };
        }

        return $hours;
    }

    /** `8M`, `512K`, `2G`, `1048576` → bytes (0 for empty / unlimited). */
    public static function iniBytes(string $value): int
    {
        $value = trim($value);
        if ($value === '' || $value === '-1') {
            return 0;
        }

        $number = (int) $value;

        return match (strtolower(substr($value, -1))) {
            'g' => $number * 1024 ** 3,
            'm' => $number * 1024 ** 2,
            'k' => $number * 1024,
            default => $number,
        };
    }
}
