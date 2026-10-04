<?php

declare(strict_types=1);

namespace App\Domain\Media\Actions;

use App\Domain\Media\Enums\MediaFormat;
use App\Domain\Media\Models\Media;
use Illuminate\Contracts\Config\Repository as Config;

/**
 * Edits the describable part of a media item (alt/title/caption) and its focal point. Moving the focal point
 * re-queues the focal crops the item already has (thumb, og, square…) so they follow the new subject; responsive
 * widths are not cropped and stay as they are.
 */
final class UpdateMediaDetails
{
    private const FOCAL_EPSILON = 0.0005;

    public function __construct(
        private readonly RegenerateMediaVariants $regenerate,
        private readonly Config $config,
    ) {}

    public function handle(Media $media, ?string $alt, ?string $title, ?string $caption, ?float $focalX = null, ?float $focalY = null): Media
    {
        $focalX = self::clamp($focalX ?? $media->focal_x);
        $focalY = self::clamp($focalY ?? $media->focal_y);

        $focalMoved = abs($focalX - $media->focal_x) > self::FOCAL_EPSILON || abs($focalY - $media->focal_y) > self::FOCAL_EPSILON;

        $media->fill([
            'alt' => self::clean($alt, 255),
            'title' => self::clean($title, 255),
            'caption' => self::clean($caption, null),
            'focal_x' => round($focalX, 4),
            'focal_y' => round($focalY, 4),
        ])->save();

        if ($focalMoved) {
            $presets = $this->cropPresetsOf($media);
            if ($presets !== []) {
                $this->regenerate->handle($media, $presets);
            }
        }

        return $media;
    }

    /**
     * Focal-cropped presets this item currently has variants for.
     *
     * @return list<string>
     */
    private function cropPresetsOf(Media $media): array
    {
        if (MediaFormat::fromMime($media->mime) === MediaFormat::Svg) {
            return [];
        }

        $existing = [];
        foreach (array_keys($media->variants ?? []) as $name) {
            $existing[explode('_', (string) $name, 2)[0]] = true;
        }

        $presets = [];
        foreach ((array) $this->config->get('media.presets', []) as $name => $preset) {
            if (is_array($preset) && (bool) ($preset['crop'] ?? false) && isset($existing[$name])) {
                $presets[] = (string) $name;
            }
        }

        return $presets;
    }

    private static function clamp(float $value): float
    {
        return max(0.0, min(1.0, $value));
    }

    private static function clean(?string $value, ?int $max): ?string
    {
        $value = trim((string) $value);
        if ($value === '') {
            return null;
        }

        return $max === null ? $value : mb_substr($value, 0, $max);
    }
}
