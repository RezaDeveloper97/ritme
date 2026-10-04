<?php

declare(strict_types=1);

namespace App\Domain\Directory\Data;

use Carbon\CarbonImmutable;

/**
 * An indexable landing page: a city (`categorySlug` null) or a city × category with at least one published place.
 * `lastmod` = newest update among its places. Used for sitemaps and crawlable city/category links (L5-02).
 */
final readonly class LandingComboData
{
    public function __construct(
        public string $citySlug,
        public string $cityName,
        public ?string $categorySlug,
        public ?string $categoryName,
        public int $placeCount,
        public ?CarbonImmutable $lastmod = null,
    ) {}

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'citySlug' => $this->citySlug, 'cityName' => $this->cityName, 'categorySlug' => $this->categorySlug,
            'categoryName' => $this->categoryName, 'placeCount' => $this->placeCount, 'lastmod' => $this->lastmod?->toIso8601String(),
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        return new self(
            citySlug: (string) $data['citySlug'],
            cityName: (string) $data['cityName'],
            categorySlug: isset($data['categorySlug']) ? (string) $data['categorySlug'] : null,
            categoryName: isset($data['categoryName']) ? (string) $data['categoryName'] : null,
            placeCount: (int) $data['placeCount'],
            lastmod: isset($data['lastmod']) ? CarbonImmutable::parse((string) $data['lastmod']) : null,
        );
    }
}
