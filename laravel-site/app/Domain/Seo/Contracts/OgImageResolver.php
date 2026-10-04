<?php

declare(strict_types=1);

namespace App\Domain\Seo\Contracts;

use App\Domain\Seo\Data\SeoImage;

/**
 * Turns a media id into the 1200×630 OG image. Bound to NullOgImageResolver until the media library (L2) exists.
 */
interface OgImageResolver
{
    public function resolve(int $mediaId, string $alt): ?SeoImage;
}
