<?php

declare(strict_types=1);

namespace App\View\Components;

use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Media\Data\MediaData;
use App\Domain\Media\Enums\MediaFormat;
use Illuminate\Contracts\View\View;
use Illuminate\View\Component;
use InvalidArgumentException;

/**
 * <x-picture :media="$m" :mobile="$m2" sizes="(max-width: 768px) 100vw, 50vw" alt="…" priority /> — the only way a
 * raster image reaches a public page.
 *
 *  - `media` / `mobile`: a MediaData (preferred, resolved by the page's repository/DTO) or a media id (resolved here
 *    through the cached MediaRepository — never a query in Blade). Missing media renders nothing.
 *  - `<source>` per generated format (avif when encoded, webp) over the mobile 480/768 + desktop 1280/1920 variants;
 *    the fallback `<img>` carries the jpg/png srcset. `mobile` = art direction: its sources come first with
 *    `media="(max-width: 767px)"` and their own width/height.
 *  - Always width + height (intrinsic size of the media; override with :width/:height) and alt: the explicit `alt`,
 *    else the media's alt; an empty alt only with `decorative`. A missing alt throws outside production.
 *  - Lazy by default (`loading="lazy"`, `decoding="async"`). `priority` (LCP) → eager + `fetchpriority="high"` +
 *    `<link rel="preload" as="image" imagesrcset imagesizes type>` pushed once into the `head` stack. Each attribute
 *    can be overridden with `loading` / `fetchpriority` / `decoding`.
 *  - Placeholder: opaque images get the `bg-lavender` token class while loading (no inline style, CSP stays strict).
 *    The stored LQIP/dominant colour are not emitted — both would need an inline style or a per-image CSS rule.
 *  - Other attributes (class, data-*, …) go on the `<img>`; `picture-class` goes on the `<picture>`.
 */
final class Picture extends Component
{
    public const MOBILE_QUERY = '(max-width: 767px)';

    public const DESKTOP_QUERY = '(min-width: 768px)';

    public const PLACEHOLDER_CLASS = 'bg-lavender';

    public readonly ?MediaData $image;

    public readonly ?MediaData $mobileImage;

    /** @var list<array{media: string|null, type: string, srcset: string, width: int|null, height: int|null}> */
    public readonly array $sources;

    public readonly string $src;

    public readonly ?string $srcset;

    public readonly ?int $imgWidth;

    public readonly ?int $imgHeight;

    public readonly string $altText;

    public readonly string $loadingValue;

    public readonly ?string $fetchpriorityValue;

    public readonly string $decodingValue;

    public readonly ?string $placeholderClass;

    /** @var list<array{href: string|null, imagesrcset: string|null, imagesizes: string|null, type: string, media: string|null}> */
    public readonly array $preloads;

    public readonly string $preloadKey;

    public function __construct(
        MediaData|int|null $media = null,
        MediaData|int|null $mobile = null,
        public readonly string $sizes = '100vw',
        ?string $alt = null,
        public readonly bool $decorative = false,
        public readonly bool $priority = false,
        ?string $loading = null,
        ?string $fetchpriority = null,
        ?string $decoding = null,
        ?int $width = null,
        ?int $height = null,
        public readonly ?string $pictureClass = null,
        ?MediaRepository $repository = null,
    ) {
        $this->image = self::find($media, $repository);
        $mobileImage = self::find($mobile, $repository);
        $this->mobileImage = $mobileImage !== null && $this->image !== null && $mobileImage->id !== $this->image->id
            && ! $mobileImage->isSvg() ? $mobileImage : null;

        $this->loadingValue = $loading ?? ($priority ? 'eager' : 'lazy');
        $this->fetchpriorityValue = $fetchpriority ?? ($priority ? 'high' : null);
        $this->decodingValue = $decoding ?? 'async';
        $this->preloadKey = 'picture-preload-'.($this->image->id ?? 0).'-'.($this->mobileImage->id ?? 0);

        $image = $this->image;
        if ($image === null) {
            $this->sources = $this->preloads = [];
            $this->src = $this->altText = '';
            $this->srcset = $this->placeholderClass = null;
            $this->imgWidth = $this->imgHeight = null;

            return;
        }

        $this->imgWidth = $width ?? $image->width;
        $this->imgHeight = $height ?? $image->height;
        if (($this->imgWidth === null || $this->imgHeight === null) && ! app()->isProduction()) {
            throw new InvalidArgumentException("<x-picture>: media #{$image->id} has no intrinsic size; pass :width and :height.");
        }

        $this->altText = $this->resolveAlt($image, $alt);

        $fallback = $image->fallbackFormat();
        $hasVariants = ! $image->isSvg() && $image->formats() !== [];

        $this->src = $hasVariants ? $image->url('desktop_1280', $fallback) : $image->url;
        $this->srcset = $hasVariants ? self::nonEmpty($image->srcset($fallback)) : null;
        $this->placeholderClass = $fallback === 'jpg' ? self::PLACEHOLDER_CLASS : null;

        $sources = [];
        if ($this->mobileImage !== null) {
            foreach (self::formatsWithFallback($this->mobileImage) as $format) {
                $srcset = self::nonEmpty($this->mobileImage->srcset($format));
                if ($srcset !== null) {
                    $sources[] = [
                        'media' => self::MOBILE_QUERY,
                        'type' => self::mime($format),
                        'srcset' => $srcset,
                        'width' => $this->mobileImage->width,
                        'height' => $this->mobileImage->height,
                    ];
                }
            }
        }
        if ($hasVariants) {
            foreach ($image->formats() as $format) {
                if ($format === $fallback) {
                    continue;
                }
                $srcset = self::nonEmpty($image->srcset($format));
                if ($srcset !== null) {
                    $sources[] = ['media' => null, 'type' => self::mime($format), 'srcset' => $srcset, 'width' => null, 'height' => null];
                }
            }
        }
        $this->sources = $sources;
        $this->preloads = $priority ? $this->buildPreloads($image) : [];
    }

    /**
     * URL of a media item (or id) for meta tags, JSON-LD and feeds: absolute, variant-aware (`og`, `thumb`,
     * `desktop_1280` …), falling back to the original; null for missing media. This is the `media_url()` helper.
     */
    public static function mediaUrl(MediaData|int|null $media, ?string $variant = null, ?string $format = null): ?string
    {
        $data = self::find($media, null);

        return $data === null ? null : url($data->url($variant, $format));
    }

    public function shouldRender(): bool
    {
        return $this->image !== null;
    }

    public function render(): View
    {
        return view('components.picture');
    }

    private static function find(MediaData|int|null $media, ?MediaRepository $repository): ?MediaData
    {
        if ($media === null || $media instanceof MediaData) {
            return $media;
        }

        return ($repository ?? app(MediaRepository::class))->find($media);
    }

    private function resolveAlt(MediaData $image, ?string $alt): string
    {
        if ($this->decorative) {
            return '';
        }

        $resolved = trim($alt ?? $image->alt ?? '');
        if ($resolved === '' && ! app()->isProduction()) {
            throw new InvalidArgumentException("<x-picture>: media #{$image->id} needs an alt (or mark it `decorative`).");
        }

        return $resolved;
    }

    /**
     * @return list<array{href: string|null, imagesrcset: string|null, imagesizes: string|null, type: string, media: string|null}>
     */
    private function buildPreloads(MediaData $image): array
    {
        $desktopMedia = $this->mobileImage !== null ? self::DESKTOP_QUERY : null;
        $preloads = [];

        if ($this->mobileImage !== null) {
            $preloads[] = $this->preloadFor($this->mobileImage, self::MOBILE_QUERY);
        }

        $preloads[] = $image->isSvg() || $image->formats() === []
            ? ['href' => $image->url, 'imagesrcset' => null, 'imagesizes' => null, 'type' => $image->mime, 'media' => $desktopMedia]
            : $this->preloadFor($image, $desktopMedia);

        return $preloads;
    }

    /**
     * Preloads the best (first generated) format — the one a modern browser picks from the matching `<source>`.
     *
     * @return array{href: string|null, imagesrcset: string|null, imagesizes: string|null, type: string, media: string|null}
     */
    private function preloadFor(MediaData $media, ?string $query): array
    {
        $format = self::formatsWithFallback($media)[0];

        return [
            'href' => null,
            'imagesrcset' => self::nonEmpty($media->srcset($format)) ?? $media->url,
            'imagesizes' => $this->sizes,
            'type' => self::mime($format),
            'media' => $query,
        ];
    }

    /**
     * @return non-empty-list<string>
     */
    private static function formatsWithFallback(MediaData $media): array
    {
        $formats = $media->formats();
        if ($formats === []) {
            return [$media->extension];
        }

        $fallback = $media->fallbackFormat();
        if (! in_array($fallback, $formats, true)) {
            $formats[] = $fallback;
        }

        return $formats;
    }

    private static function mime(string $format): string
    {
        return MediaFormat::tryFrom($format)?->mime() ?? 'image/'.$format;
    }

    private static function nonEmpty(string $value): ?string
    {
        return $value === '' ? null : $value;
    }
}
