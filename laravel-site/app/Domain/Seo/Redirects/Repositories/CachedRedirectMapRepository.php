<?php

declare(strict_types=1);

namespace App\Domain\Seo\Redirects\Repositories;

use App\Domain\Seo\Redirects\Contracts\RedirectMapRepository;
use App\Domain\Seo\Redirects\Data\RedirectMap;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CachedRepository;

/**
 * The whole redirect map as one cache-aside entry in the `seo` namespace (RedirectObserver bumps it): a warm lookup
 * is one cache read and no query. Read only for 404s and non-canonical URLs (ApplyRedirects).
 */
final class CachedRedirectMapRepository extends CachedRepository implements RedirectMapRepository
{
    public function __construct(private readonly RedirectMapRepository $inner, CacheAside $cache)
    {
        parent::__construct($cache);
    }

    protected function namespace(): string
    {
        return 'seo';
    }

    public function map(): RedirectMap
    {
        /** @var array{exact?: array<string, array{0: int, 1: string|null, 2: int}>, regex?: list<array{0: int, 1: string, 2: string|null, 3: int}>} $data */
        $data = $this->remember(['redirects', 'map'], fn (): array => $this->inner->map()->toArray());

        return RedirectMap::fromArray($data);
    }
}
