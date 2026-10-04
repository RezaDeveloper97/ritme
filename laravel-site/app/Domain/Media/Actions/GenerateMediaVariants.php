<?php

declare(strict_types=1);

namespace App\Domain\Media\Actions;

use App\Domain\Media\Data\VariantSpec;
use App\Domain\Media\Enums\MediaFormat;
use App\Domain\Media\Models\Media;
use App\Domain\Media\Support\FocalCrop;
use App\Domain\Media\Support\ImageAnalyzer;
use App\Domain\Media\Support\ImageEncoder;
use App\Domain\Media\Support\MemoryLimit;
use App\Domain\Media\Support\VariantPlanner;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Filesystem\Factory as Filesystems;
use Intervention\Image\ImageManager;
use Intervention\Image\Interfaces\ImageInterface;

/**
 * Writes the variants of one media item from its stored original: responsive widths (mobile/desktop), focal crops
 * (thumb/og/square) in avif/webp/fallback, or a poster + thumb for animated GIFs. With `$presets` only those presets
 * are (re)generated and merged into the existing set; otherwise every automatic preset replaces the old set.
 *
 * `/media` is served `immutable` for a year (L1-07), so a file name must never be reused for different bytes: every
 * variant file name carries a short hash of its encoded content (`{stem}-{variant}.{version}.{ext}`). A regenerate
 * that changes the pixels (e.g. a moved focal point) yields new URLs; an identical result keeps the same URL. The
 * new set is written first, then files no longer referenced are deleted. Saving the row bumps `media`, `seo` and
 * `pages` (MediaObserver), so cached DTOs, OG tags and full pages pick up the new URLs. Rows generated before
 * versioning keep their stored unversioned paths until they are regenerated.
 */
final class GenerateMediaVariants
{
    public const VERSION_LENGTH = 10;

    public function __construct(
        private readonly ImageManager $images,
        private readonly VariantPlanner $planner,
        private readonly ImageEncoder $encoder,
        private readonly ImageAnalyzer $analyzer,
        private readonly Filesystems $filesystems,
        private readonly Config $config,
    ) {}

    /**
     * @param  list<string>|null  $presets
     */
    public function handle(Media $media, ?array $presets = null): Media
    {
        if (MediaFormat::fromMime($media->mime) === MediaFormat::Svg) {
            $media->forceFill(['variants' => [], 'optimized_at' => now()])->save();

            return $media;
        }

        MemoryLimit::raise((string) $this->config->get('media.memory_limit', '512M'));

        $disk = $this->filesystems->disk($media->disk);
        $image = $this->images->read((string) $disk->get($media->path()));

        $animated = $media->mime === MediaFormat::Gif->mime() && $image->isAnimated();
        if ($animated) {
            $image = $image->removeAnimation(0);
        }

        $alpha = $media->mime !== MediaFormat::Jpg->mime() && $this->analyzer->hasAlpha($image);

        $specs = $animated
            ? $this->planner->planAnimated($image->width(), $image->height(), $alpha, $media->focal_x, $media->focal_y)
            : $this->planner->plan($image->width(), $image->height(), $alpha, $presets, $media->focal_x, $media->focal_y);

        if ($animated && $presets !== null) {
            $specs = array_values(array_filter($specs, static fn (VariantSpec $s): bool => in_array($s->preset, [...$presets, 'poster'], true)));
        }

        $old = $media->variants ?? [];
        $variants = $presets === null ? [] : array_filter(
            $old,
            static fn (string $name): bool => ! in_array(explode('_', $name, 2)[0], $presets, true),
            ARRAY_FILTER_USE_KEY,
        );

        $stem = pathinfo($media->filename, PATHINFO_FILENAME);
        foreach ($specs as $spec) {
            $resized = $this->transform($image, $spec, $media);

            foreach ($spec->formats as $format) {
                $encoded = $this->encoder->encode($resized, $format);
                $bytes = $encoded->toString();
                $path = ltrim("{$media->directory}/{$stem}-{$spec->name}.".self::version($bytes).".{$format->value}", '/');
                $disk->put($path, $bytes);

                $variants[$spec->name][$format->value] = ['w' => $resized->width(), 'h' => $resized->height(), 'path' => $path, 'size' => $encoded->size()];
            }
        }

        $this->deleteStale($media, $old, $variants);

        $media->forceFill(['variants' => $variants, 'optimized_at' => now()])->save();

        return $media;
    }

    /**
     * Short content hash used as the variant's version segment (cache-busting under immutable caching).
     */
    public static function version(string $bytes): string
    {
        return substr(hash('xxh128', $bytes), 0, self::VERSION_LENGTH);
    }

    private function transform(ImageInterface $image, VariantSpec $spec, Media $media): ImageInterface
    {
        $copy = clone $image;

        if (! $spec->crop) {
            return $copy->scaleDown($spec->width, $spec->height);
        }

        $preset = (array) $this->config->get("media.presets.{$spec->preset}", []);
        $box = FocalCrop::box($copy->width(), $copy->height(), (int) ($preset['width'] ?? $spec->width), (int) ($preset['height'] ?? $spec->height), $media->focal_x, $media->focal_y);

        $copy->crop($box['width'], $box['height'], $box['x'], $box['y']);

        return $box['outWidth'] === $box['width'] ? $copy : $copy->resize($box['outWidth'], $box['outHeight']);
    }

    /**
     * @param  array<string, array<string, array{w: int, h: int, path: string, size: int}>>  $old
     * @param  array<string, array<string, array{w: int, h: int, path: string, size: int}>>  $new
     */
    private function deleteStale(Media $media, array $old, array $new): void
    {
        $keep = [$media->path() => true];
        foreach ($new as $formats) {
            foreach ($formats as $file) {
                $keep[$file['path']] = true;
            }
        }

        $stale = [];
        foreach ($old as $formats) {
            foreach ($formats as $file) {
                if (! isset($keep[$file['path']])) {
                    $stale[] = $file['path'];
                }
            }
        }

        if ($stale !== []) {
            $this->filesystems->disk($media->disk)->delete($stale);
        }
    }
}
