<?php

declare(strict_types=1);

namespace App\Domain\Seo\Support;

use App\Domain\Seo\Contracts\OgImageResolver;
use App\Domain\Seo\Data\SeoImage;

/**
 * Placeholder until the media library (L2) binds a real resolver: no media, no OG image.
 */
final class NullOgImageResolver implements OgImageResolver
{
    public function resolve(int $mediaId, string $alt): ?SeoImage
    {
        return null;
    }
}
