<?php

declare(strict_types=1);

namespace App\Filament\Resources\Media;

use App\Domain\Media\Models\Media;
use Illuminate\Support\Facades\Storage;

/**
 * Admin-only presentation of a media row: preview URLs, human sizes, variant list and savings against the stored
 * original. Pure formatting — no writes, no queries.
 */
final class MediaPresenter
{
    /**
     * Path (on the media disk) of the smallest preview available: thumb (webp > fallback), else the original.
     */
    public static function thumbPath(Media $media): string
    {
        $thumb = $media->variants['thumb'] ?? $media->variants['poster'] ?? [];
        foreach (['webp', 'jpg', 'png'] as $format) {
            if (isset($thumb[$format]['path'])) {
                return $thumb[$format]['path'];
            }
        }

        return $media->path();
    }

    public static function thumbUrl(Media $media): string
    {
        return Storage::disk($media->disk)->url(self::thumbPath($media));
    }

    /**
     * Uncropped preview for the focal-point picker: a responsive variant up to 1280 px, else the original.
     */
    public static function previewUrl(Media $media): string
    {
        $path = $media->path();
        foreach (['desktop_1280', 'mobile_768', 'poster'] as $name) {
            foreach (['webp', 'jpg', 'png'] as $format) {
                if (isset($media->variants[$name][$format]['path'])) {
                    $path = $media->variants[$name][$format]['path'];
                    break 2;
                }
            }
        }

        return Storage::disk($media->disk)->url($path);
    }

    public static function humanSize(int $bytes): string
    {
        if ($bytes >= 1024 * 1024) {
            return number_format($bytes / (1024 * 1024), 1).' MB';
        }

        return number_format(max(1, (int) round($bytes / 1024))).' KB';
    }

    public static function dimensions(Media $media): string
    {
        return $media->width !== null && $media->height !== null ? "{$media->width}×{$media->height}" : '—';
    }

    /**
     * Savings of the largest responsive variant (smallest format) compared with the stored original, in percent.
     * Null while nothing has been generated.
     */
    public static function savingsPercent(Media $media): ?int
    {
        $best = null;
        $bestWidth = 0;
        foreach ($media->variants ?? [] as $name => $formats) {
            if (! in_array(explode('_', (string) $name, 2)[0], ['mobile', 'desktop', 'poster'], true) || $formats === []) {
                continue;
            }
            $width = max(array_map(static fn (array $f): int => (int) $f['w'], $formats));
            if ($width >= $bestWidth) {
                $bestWidth = $width;
                $best = min(array_map(static fn (array $f): int => (int) $f['size'], $formats));
            }
        }

        return $best === null || $media->size < 1 ? null : self::percent($best, $media->size);
    }

    /**
     * One line per variant file: "desktop_1280 · webp · 1280×720 · 84 KB · −71٪".
     *
     * @return list<string>
     */
    public static function variantLines(Media $media): array
    {
        $lines = [];
        foreach ($media->variants ?? [] as $name => $formats) {
            foreach ($formats as $format => $file) {
                $lines[] = sprintf(
                    '%s · %s · %d×%d · %s · %s',
                    $name,
                    $format,
                    (int) $file['w'],
                    (int) $file['h'],
                    self::humanSize((int) $file['size']),
                    self::savingsLabel(self::percent((int) $file['size'], $media->size)),
                );
            }
        }

        return $lines;
    }

    public static function savingsLabel(?int $percent): string
    {
        if ($percent === null) {
            return '—';
        }

        return $percent >= 0 ? "{$percent}٪ کم‌حجم‌تر" : abs($percent).'٪ پرحجم‌تر';
    }

    public static function status(Media $media): string
    {
        if ($media->optimized_at === null) {
            return 'در صف بهینه‌سازی';
        }

        return $media->variants === [] || $media->variants === null ? 'بدون نسخه (برداری)' : 'بهینه شده';
    }

    private static function percent(int $size, int $original): int
    {
        return $original < 1 ? 0 : (int) round((1 - $size / $original) * 100);
    }
}
