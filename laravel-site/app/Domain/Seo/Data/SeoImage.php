<?php

declare(strict_types=1);

namespace App\Domain\Seo\Data;

/**
 * An absolute image URL with the dimensions and alt text the OG / Twitter tags need.
 */
final readonly class SeoImage
{
    public function __construct(
        public string $url,
        public int $width,
        public int $height,
        public string $alt,
        public string $type = 'image/jpeg',
    ) {}
}
