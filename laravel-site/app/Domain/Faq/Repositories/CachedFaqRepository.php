<?php

declare(strict_types=1);

namespace App\Domain\Faq\Repositories;

use App\Domain\Faq\Contracts\FaqRepository;
use App\Domain\Faq\Data\FaqGroupData;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CachedRepository;

/**
 * `faq` namespace. All groups with their published items are one cache entry (a few dozen rows), so every lookup —
 * including unknown slugs — is served from it without a query.
 */
final class CachedFaqRepository extends CachedRepository implements FaqRepository
{
    public function __construct(private readonly EloquentFaqRepository $inner, CacheAside $cache)
    {
        parent::__construct($cache);
    }

    protected function namespace(): string
    {
        return 'faq';
    }

    public function group(string $slug): ?FaqGroupData
    {
        foreach ($this->all() as $group) {
            if ($group->slug === $slug) {
                return $group;
            }
        }

        return null;
    }

    public function listed(): array
    {
        return array_values(array_filter($this->all(), static fn (FaqGroupData $group): bool => $group->listed && ! $group->isEmpty()));
    }

    /**
     * @return list<FaqGroupData>
     */
    private function all(): array
    {
        /** @var list<array<string, mixed>> $data */
        $data = $this->remember('groups', fn (): array => array_map(
            static fn (FaqGroupData $group): array => $group->toArray(),
            $this->inner->all(),
        ));

        return array_map(FaqGroupData::fromArray(...), $data);
    }
}
