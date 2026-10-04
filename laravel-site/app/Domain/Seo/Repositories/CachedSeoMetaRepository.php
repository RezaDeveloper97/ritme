<?php

declare(strict_types=1);

namespace App\Domain\Seo\Repositories;

use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Data\SeoMetaData;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CachedRepository;

/**
 * Cache-aside over seo_meta in the `seo` namespace (SeoMetaObserver bumps it). Misses are cached too (most pages
 * have no row). Arrays rather than DTOs are cached so a deploy that changes the DTO never unserializes stale objects.
 */
final class CachedSeoMetaRepository extends CachedRepository implements SeoMetaRepository
{
    public function __construct(private readonly SeoMetaRepository $inner, CacheAside $cache)
    {
        parent::__construct($cache);
    }

    protected function namespace(): string
    {
        return 'seo';
    }

    public function forRoute(string $routeName): ?SeoMetaData
    {
        return $this->hydrate($this->remember(
            ['meta', 'route', $routeName],
            fn (): ?array => $this->inner->forRoute($routeName)?->toArray(),
            cacheNull: true,
        ));
    }

    public function forModel(string $morphType, int|string $id): ?SeoMetaData
    {
        return $this->hydrate($this->remember(
            ['meta', 'model', $morphType, $id],
            fn (): ?array => $this->inner->forModel($morphType, $id)?->toArray(),
            cacheNull: true,
        ));
    }

    /**
     * @param  array<string, mixed>|null  $values
     */
    private function hydrate(?array $values): ?SeoMetaData
    {
        return $values === null ? null : SeoMetaData::fromArray($values);
    }
}
