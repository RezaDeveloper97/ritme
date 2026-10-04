<?php

declare(strict_types=1);

namespace App\Domain\Media\Data;

/**
 * Read model of one media item for views (<x-picture>, OG tags, JSON-LD). URLs are already resolved against the
 * disk (root-relative on the public disk). Any missing variant falls back to the original — never a broken image.
 *
 * @phpstan-type VariantFile array{w: int, h: int, url: string, size: int}
 */
final readonly class MediaData
{
    /**
     * Responsive presets used by srcset() by default.
     */
    public const RESPONSIVE_PRESETS = ['mobile', 'desktop'];

    /**
     * @param  array<string, array<string, VariantFile>>  $variants  name => format => file
     */
    public function __construct(
        public int $id,
        public string $url,
        public string $mime,
        public string $extension,
        public ?int $width,
        public ?int $height,
        public ?string $alt,
        public ?string $title,
        public ?string $caption,
        public float $focalX,
        public float $focalY,
        public ?string $dominantColor,
        public ?string $lqip,
        public array $variants,
        public bool $optimized,
    ) {}

    public function isSvg(): bool
    {
        return $this->mime === 'image/svg+xml';
    }

    /**
     * @return VariantFile|null
     */
    public function file(string $variant, string $format): ?array
    {
        return $this->variants[$variant][$format] ?? null;
    }

    /**
     * URL of a variant in a format; without a format the variant's fallback (jpg/png) is used; a missing variant or
     * format falls back to the variant's fallback, then to the original.
     */
    public function url(?string $variant = null, ?string $format = null): string
    {
        if ($variant === null || ! isset($this->variants[$variant])) {
            return $this->url;
        }

        $files = $this->variants[$variant];
        $format ??= $this->fallbackFormat();

        $file = $files[$format] ?? $files[$this->fallbackFormat()] ?? null;

        return $file === null ? $this->url : $file['url'];
    }

    /**
     * The universally supported format of the variants: jpg for photos, png for transparent images (or the original
     * extension when nothing was generated).
     */
    public function fallbackFormat(): string
    {
        foreach ($this->variants as $files) {
            foreach (['jpg', 'png'] as $format) {
                if (isset($files[$format])) {
                    return $format;
                }
            }
        }

        return $this->extension;
    }

    /**
     * Formats available for the responsive presets, in the order they were generated (e.g. avif, webp, jpg).
     *
     * @param  list<string>  $presets
     * @return list<string>
     */
    public function formats(array $presets = self::RESPONSIVE_PRESETS): array
    {
        $formats = [];
        foreach ($this->variantsOf($presets) as $files) {
            foreach (array_keys($files) as $format) {
                $formats[$format] = true;
            }
        }

        return array_keys($formats);
    }

    /**
     * `srcset` for one format over the responsive presets, ascending width. The original is appended when it is in
     * the same format and wider than every variant (or is the only candidate).
     *
     * @param  list<string>  $presets
     */
    public function srcset(string $format, array $presets = self::RESPONSIVE_PRESETS): string
    {
        $candidates = [];
        foreach ($this->variantsOf($presets) as $files) {
            if (isset($files[$format])) {
                $candidates[$files[$format]['w']] = $files[$format]['url'];
            }
        }

        if ($format === $this->extension && $this->width !== null && ($candidates === [] || $this->width > max(array_keys($candidates)))) {
            $candidates[$this->width] = $this->url;
        }

        ksort($candidates);

        return implode(', ', array_map(
            static fn (int $width, string $url): string => "{$url} {$width}w",
            array_keys($candidates),
            $candidates,
        ));
    }

    /**
     * @param  list<string>  $presets
     * @return array<string, array<string, VariantFile>>
     */
    private function variantsOf(array $presets): array
    {
        return array_filter(
            $this->variants,
            static fn (string $name): bool => in_array(explode('_', $name, 2)[0], $presets, true),
            ARRAY_FILTER_USE_KEY,
        );
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return get_object_vars($this);
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        /** @var array<string, array<string, VariantFile>> $variants */
        $variants = (array) ($data['variants'] ?? []);

        return new self(
            id: (int) $data['id'],
            url: (string) $data['url'],
            mime: (string) $data['mime'],
            extension: (string) $data['extension'],
            width: isset($data['width']) ? (int) $data['width'] : null,
            height: isset($data['height']) ? (int) $data['height'] : null,
            alt: isset($data['alt']) ? (string) $data['alt'] : null,
            title: isset($data['title']) ? (string) $data['title'] : null,
            caption: isset($data['caption']) ? (string) $data['caption'] : null,
            focalX: (float) ($data['focalX'] ?? 0.5),
            focalY: (float) ($data['focalY'] ?? 0.5),
            dominantColor: isset($data['dominantColor']) ? (string) $data['dominantColor'] : null,
            lqip: isset($data['lqip']) ? (string) $data['lqip'] : null,
            variants: $variants,
            optimized: (bool) ($data['optimized'] ?? false),
        );
    }
}
