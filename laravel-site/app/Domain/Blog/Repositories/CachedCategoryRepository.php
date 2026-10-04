<?php

declare(strict_types=1);

namespace App\Domain\Blog\Repositories;

use App\Domain\Blog\Contracts\CategoryRepository;
use App\Domain\Blog\Data\CategoryData;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CachedRepository;

/**
 * `blog` namespace; slug lookups go through the cached list (categories are few), so unknown slugs cost no query.
 */
final class CachedCategoryRepository extends CachedRepository implements CategoryRepository
{
    public function __construct(private readonly CategoryRepository $inner, CacheAside $cache)
    {
        parent::__construct($cache);
    }

    protected function namespace(): string
    {
        return 'blog';
    }

    public function all(): array
    {
        /** @var list<array<string, mixed>> $data */
        $data = $this->remember('categories', fn (): array => array_map(
            static fn (CategoryData $category): array => $category->toArray(),
            $this->inner->all(),
        ));

        return array_map(CategoryData::fromArray(...), $data);
    }

    public function findBySlug(string $slug): ?CategoryData
    {
        foreach ($this->all() as $category) {
            if ($category->slug === $slug) {
                return $category;
            }
        }

        return null;
    }
}
