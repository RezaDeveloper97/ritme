<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Repositories;

use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Data\ProductCardData;
use App\Domain\Shop\Catalog\Data\ProductCriteria;
use App\Domain\Shop\Catalog\Data\ProductData;
use App\Domain\Shop\Catalog\Data\ProductPage;
use App\Domain\Shop\Catalog\Data\ReviewPage;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CachedRepository;

/**
 * Caches plain arrays in the `shop` namespace; product, variant, review and taxonomy observers (and the Sync* /
 * AdjustStock actions) bump it. Misses of slug lookups are not cached.
 */
final class CachedProductRepository extends CachedRepository implements ProductRepository
{
    public function __construct(private readonly ProductRepository $inner, CacheAside $cache)
    {
        parent::__construct($cache);
    }

    protected function namespace(): string
    {
        return 'shop';
    }

    public function findPublishedBySlug(string $slug): ?ProductData
    {
        if ($slug === '') {
            return null;
        }

        /** @var array<string, mixed>|null $data */
        $data = $this->remember(['product', $slug], fn (): ?array => $this->inner->findPublishedBySlug($slug)?->toArray());

        return $data === null ? null : ProductData::fromArray($data);
    }

    public function currentSlugFor(string $previousSlug): ?string
    {
        if ($previousSlug === '') {
            return null;
        }

        return $this->remember(['slug-redirect', $previousSlug], fn (): ?string => $this->inner->currentSlugFor($previousSlug));
    }

    public function list(ProductCriteria $criteria): ProductPage
    {
        /** @var array<string, mixed> $data */
        $data = $this->remember(['list', $criteria->cacheKey()], fn (): array => $this->inner->list($criteria)->toArray());

        return ProductPage::fromArray($data);
    }

    public function bestSellers(int $limit = 10, ?int $categoryId = null): array
    {
        /** @var list<array<string, mixed>> $data */
        $data = $this->remember(
            ['best-sellers', $limit, $categoryId ?? 'all'],
            fn (): array => array_map(static fn (ProductCardData $c): array => $c->toArray(), $this->inner->bestSellers($limit, $categoryId)),
        );

        return array_map(ProductCardData::fromArray(...), $data);
    }

    public function frequentlyBoughtWith(int $productId, int $limit = 5): array
    {
        /** @var list<array<string, mixed>> $data */
        $data = $this->remember(
            ['bought-with', $productId, $limit],
            fn (): array => array_map(static fn (ProductCardData $c): array => $c->toArray(), $this->inner->frequentlyBoughtWith($productId, $limit)),
        );

        return array_map(ProductCardData::fromArray(...), $data);
    }

    public function reviews(int $productId, int $page = 1, int $perPage = 10): ReviewPage
    {
        /** @var array<string, mixed> $data */
        $data = $this->remember(['reviews', $productId, $page, $perPage], fn (): array => $this->inner->reviews($productId, $page, $perPage)->toArray());

        return ReviewPage::fromArray($data);
    }
}
