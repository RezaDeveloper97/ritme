<?php

declare(strict_types=1);

namespace App\Domain\Media\Support;

use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Seo\Contracts\OgImageResolver;
use App\Domain\Seo\Data\SeoImage;
use Illuminate\Contracts\Routing\UrlGenerator;

/**
 * OG / Twitter image for a media id: the 1200×630 `og` variant (jpg/png — what every crawler reads), else the raster
 * original. SVGs and missing media yield null (crawlers ignore SVG og:image).
 */
final class MediaOgImageResolver implements OgImageResolver
{
    public function __construct(private readonly MediaRepository $media, private readonly UrlGenerator $url) {}

    public function resolve(int $mediaId, string $alt): ?SeoImage
    {
        $media = $this->media->find($mediaId);
        if ($media === null || $media->isSvg()) {
            return null;
        }

        $alt = $media->alt !== null && $media->alt !== '' ? $media->alt : $alt;

        foreach (['jpg' => 'image/jpeg', 'png' => 'image/png'] as $format => $mime) {
            $file = $media->file('og', $format);
            if ($file !== null) {
                return new SeoImage($this->url->to($file['url']), $file['w'], $file['h'], $alt, $mime);
            }
        }

        if ($media->width === null || $media->height === null) {
            return null;
        }

        return new SeoImage($this->url->to($media->url), $media->width, $media->height, $alt, $media->mime);
    }
}
