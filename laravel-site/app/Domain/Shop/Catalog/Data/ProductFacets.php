<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Data;

use App\Support\Money\Money;

/**
 * Filter options of a listing, computed over the published products of the category (incl. subcategories) BEFORE the
 * other filters, so options never disappear while filtering: product counts per direct subcategory, sizes and colours
 * of active variants (variant order), brands with counts, and the price range.
 */
final readonly class ProductFacets
{
    /**
     * @param  array<int, int>  $categoryCounts  direct subcategory id => products in its subtree
     * @param  list<string>  $sizes
     * @param  list<array{name: string, hex: string|null}>  $colors
     * @param  list<array{id: int, name: string, slug: string, count: int}>  $brands
     */
    public function __construct(
        public array $categoryCounts = [],
        public array $sizes = [],
        public array $colors = [],
        public array $brands = [],
        public ?Money $minPrice = null,
        public ?Money $maxPrice = null,
    ) {}

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            // JSON object keys must survive the cache round trip as ids: stored as a list of pairs.
            'categoryCounts' => array_map(static fn (int $id, int $count): array => [$id, $count], array_keys($this->categoryCounts), $this->categoryCounts),
            'sizes' => $this->sizes, 'colors' => $this->colors, 'brands' => $this->brands,
            'minPrice' => $this->minPrice?->rial, 'maxPrice' => $this->maxPrice?->rial,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        $counts = [];
        foreach ((array) ($data['categoryCounts'] ?? []) as $pair) {
            if (is_array($pair) && count($pair) === 2) {
                $counts[(int) $pair[0]] = (int) $pair[1];
            }
        }
        /** @var list<string> $sizes */
        $sizes = (array) ($data['sizes'] ?? []);
        /** @var list<array{name: string, hex: string|null}> $colors */
        $colors = (array) ($data['colors'] ?? []);
        /** @var list<array{id: int, name: string, slug: string, count: int}> $brands */
        $brands = (array) ($data['brands'] ?? []);

        return new self(
            $counts, $sizes, $colors, $brands,
            isset($data['minPrice']) ? Money::fromRial((int) $data['minPrice']) : null,
            isset($data['maxPrice']) ? Money::fromRial((int) $data['maxPrice']) : null,
        );
    }
}
