<?php

declare(strict_types=1);

namespace App\Domain\Media\Support;

use App\Domain\Media\Data\VariantSpec;
use App\Domain\Media\Enums\MediaFormat;

/**
 * Turns config/media.php presets into the concrete variants for one image. Never upscales: responsive widths wider
 * than the source are skipped, crops shrink to the largest box of the right ratio that fits.
 */
final class VariantPlanner
{
    /**
     * @param  array<string, array<string, mixed>>  $presets  config('media.presets')
     * @param  list<string>  $defaultFormats  config('media.formats')
     */
    public function __construct(
        private readonly array $presets,
        private readonly array $defaultFormats,
        private readonly FormatSupport $support,
        private readonly int $posterWidth = 1280,
    ) {}

    /**
     * @param  list<string>|null  $only  preset names; null = every preset marked `auto`
     * @return list<VariantSpec>
     */
    public function plan(int $width, int $height, bool $hasAlpha, ?array $only = null, float $focalX = 0.5, float $focalY = 0.5): array
    {
        $specs = [];

        foreach ($this->selected($only) as $name => $preset) {
            $formats = $this->formats($preset, $hasAlpha);
            if ($formats === []) {
                continue;
            }

            if ((bool) ($preset['crop'] ?? false)) {
                $box = FocalCrop::box($width, $height, (int) $preset['width'], (int) $preset['height'], $focalX, $focalY);
                $specs[] = new VariantSpec($name, $name, $box['outWidth'], $box['outHeight'], true, $formats);

                continue;
            }

            /** @var list<int> $widths */
            $widths = array_map('intval', (array) ($preset['widths'] ?? [$preset['width'] ?? 0]));
            foreach ($widths as $target) {
                if ($target < 1 || $target > $width) {
                    continue; // never upscale
                }
                $specs[] = new VariantSpec("{$name}_{$target}", $name, $target, max(1, (int) round($height * $target / $width)), false, $formats);
            }
        }

        return $specs;
    }

    /**
     * Animated GIFs stay untouched; they get a still poster (first frame) and the thumb.
     *
     * @return list<VariantSpec>
     */
    public function planAnimated(int $width, int $height, bool $hasAlpha, float $focalX = 0.5, float $focalY = 0.5): array
    {
        $target = min($this->posterWidth, $width);
        $poster = new VariantSpec('poster', 'poster', $target, max(1, (int) round($height * $target / $width)), false, $this->support->resolve($this->defaultFormats, $hasAlpha));

        return [$poster, ...$this->plan($width, $height, $hasAlpha, ['thumb'], $focalX, $focalY)];
    }

    /**
     * @return list<string>
     */
    public function presetNames(): array
    {
        return array_keys($this->presets);
    }

    /**
     * @param  list<string>|null  $only
     * @return array<string, array<string, mixed>>
     */
    private function selected(?array $only): array
    {
        if ($only === null) {
            return array_filter($this->presets, static fn (array $preset): bool => (bool) ($preset['auto'] ?? true));
        }

        return array_intersect_key($this->presets, array_flip($only));
    }

    /**
     * @param  array<string, mixed>  $preset
     * @return list<MediaFormat>
     */
    private function formats(array $preset, bool $hasAlpha): array
    {
        /** @var list<string> $configured */
        $configured = array_values(array_map('strval', (array) ($preset['formats'] ?? $this->defaultFormats)));

        return $this->support->resolve($configured, $hasAlpha);
    }
}
