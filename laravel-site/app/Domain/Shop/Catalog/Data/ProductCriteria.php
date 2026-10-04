<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Data;

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Search\Support\SearchTerms;
use App\Domain\Shop\Catalog\Enums\ProductSort;
use App\Support\Money\Money;

/**
 * Filters of the product listing (ProductsInCategory). Controllers resolve slugs to ids through CatalogRepository.
 * `categoryId` includes its subcategories. Sizes / colours match ACTIVE variants (any of the given values; both lists
 * must match the same variant). Price bounds compare the product price. `text` is normalised like site search.
 */
final readonly class ProductCriteria
{
    public const MAX_PER_PAGE = 60;

    /** @var list<int> */
    public array $brandIds;

    /** @var list<string> */
    public array $sizes;

    /** @var list<string> */
    public array $colors;

    /** @var list<string> */
    public array $tokens;

    public int $page;

    public int $perPage;

    /**
     * @param  list<int>  $brandIds
     * @param  list<string>  $sizes
     * @param  list<string>  $colors
     */
    public function __construct(
        public ?int $categoryId = null,
        array $brandIds = [],
        array $sizes = [],
        array $colors = [],
        public ?Money $minPrice = null,
        public ?Money $maxPrice = null,
        public bool $inStockOnly = false,
        public ?LifeStage $lifeStage = null,
        public ?string $text = null,
        public ProductSort $sort = ProductSort::BestSelling,
        int $page = 1,
        int $perPage = 24,
    ) {
        $ids = array_values(array_unique(array_filter(array_map(intval(...), $brandIds), static fn (int $id): bool => $id > 0)));
        sort($ids);
        $this->brandIds = $ids;
        $this->sizes = self::values($sizes);
        $this->colors = self::values($colors);
        $this->tokens = SearchTerms::fromInput($text)->tokens;
        $this->page = max(1, $page);
        $this->perPage = max(1, min(self::MAX_PER_PAGE, $perPage));
    }

    /** Whether anything beyond category, sort and page narrows the list (listing pages noindex such URLs). */
    public function isFiltered(): bool
    {
        return $this->brandIds !== [] || $this->sizes !== [] || $this->colors !== [] || $this->minPrice !== null
            || $this->maxPrice !== null || $this->inStockOnly || $this->lifeStage !== null || $this->tokens !== [];
    }

    /**
     * Stable key of everything that changes the result.
     */
    public function cacheKey(): string
    {
        return md5(json_encode([
            $this->categoryId, $this->brandIds, $this->sizes, $this->colors, $this->minPrice?->rial, $this->maxPrice?->rial,
            $this->inStockOnly, $this->lifeStage?->value, $this->tokens, $this->sort->value, $this->page, $this->perPage,
        ], JSON_THROW_ON_ERROR));
    }

    /**
     * @param  list<string>  $values
     * @return list<string>
     */
    private static function values(array $values): array
    {
        $clean = array_values(array_unique(array_filter(array_map(static fn (mixed $v): string => trim((string) $v), $values), static fn (string $v): bool => $v !== '')));
        sort($clean);

        return $clean;
    }
}
