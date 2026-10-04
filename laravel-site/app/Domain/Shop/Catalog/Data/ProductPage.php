<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Data;

/**
 * One page of product cards plus the totals and filter options a listing needs.
 */
final readonly class ProductPage
{
    /**
     * @param  list<ProductCardData>  $items
     */
    public function __construct(
        public array $items,
        public int $total,
        public int $page,
        public int $perPage,
        public ProductFacets $facets = new ProductFacets,
    ) {}

    public function lastPage(): int
    {
        return max(1, (int) ceil($this->total / max(1, $this->perPage)));
    }

    public function isOutOfRange(): bool
    {
        return $this->page < 1 || $this->page > $this->lastPage();
    }

    public function isEmpty(): bool
    {
        return $this->items === [];
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'items' => array_map(static fn (ProductCardData $card): array => $card->toArray(), $this->items),
            'total' => $this->total, 'page' => $this->page, 'perPage' => $this->perPage,
            'facets' => $this->facets->toArray(),
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        /** @var list<array<string, mixed>> $items */
        $items = (array) ($data['items'] ?? []);
        /** @var array<string, mixed> $facets */
        $facets = (array) ($data['facets'] ?? []);

        return new self(
            array_map(ProductCardData::fromArray(...), $items),
            (int) $data['total'], (int) $data['page'], (int) $data['perPage'],
            ProductFacets::fromArray($facets),
        );
    }
}
