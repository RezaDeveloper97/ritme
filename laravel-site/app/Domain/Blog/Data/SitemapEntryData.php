<?php

declare(strict_types=1);

namespace App\Domain\Blog\Data;

use Carbon\CarbonImmutable;

/**
 * One `<url>` of a sitemap: absolute loc, real lastmod and the primary image (image:image).
 * Shape meant to map 1:1 onto the Seo sitemap entry introduced by L1-06.
 */
final readonly class SitemapEntryData
{
    public function __construct(
        public string $loc,
        public ?CarbonImmutable $lastmod = null,
        public ?string $imageUrl = null,
        public ?string $imageTitle = null,
    ) {}

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'loc' => $this->loc,
            'lastmod' => $this->lastmod?->toIso8601String(),
            'imageUrl' => $this->imageUrl,
            'imageTitle' => $this->imageTitle,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        return new self(
            loc: (string) $data['loc'],
            lastmod: isset($data['lastmod']) ? CarbonImmutable::parse((string) $data['lastmod']) : null,
            imageUrl: isset($data['imageUrl']) ? (string) $data['imageUrl'] : null,
            imageTitle: isset($data['imageTitle']) ? (string) $data['imageTitle'] : null,
        );
    }
}
