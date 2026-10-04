<?php

declare(strict_types=1);

namespace App\Domain\Directory\Repositories;

use App\Domain\Directory\Contracts\PlaceRepository;
use App\Domain\Directory\Data\PlaceData;
use App\Domain\Directory\Data\PlacePage;
use App\Domain\Directory\Data\PlaceSearchCriteria;
use App\Domain\Directory\Data\RatingSummaryData;
use App\Domain\Directory\Data\ReviewPage;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CachedRepository;

/**
 * Caches plain arrays in the `directory` namespace; place, service, review and taxonomy observers bump it. Misses of
 * slug lookups are not cached. "Open now" searches live only for one 5-minute bucket.
 */
final class CachedPlaceRepository extends CachedRepository implements PlaceRepository
{
    private const OPEN_NOW_TTL = 300;

    public function __construct(private readonly PlaceRepository $inner, CacheAside $cache)
    {
        parent::__construct($cache);
    }

    protected function namespace(): string
    {
        return 'directory';
    }

    public function findPublishedBySlug(string $slug): ?PlaceData
    {
        if ($slug === '') {
            return null;
        }

        /** @var array<string, mixed>|null $data */
        $data = $this->remember(['place', $slug], fn (): ?array => $this->inner->findPublishedBySlug($slug)?->toArray());

        return $data === null ? null : PlaceData::fromArray($data);
    }

    public function currentSlugFor(string $previousSlug): ?string
    {
        if ($previousSlug === '') {
            return null;
        }

        return $this->remember(['slug-redirect', $previousSlug], fn (): ?string => $this->inner->currentSlugFor($previousSlug));
    }

    public function search(PlaceSearchCriteria $criteria): PlacePage
    {
        /** @var array<string, mixed> $data */
        $data = $this->remember(
            ['search', $criteria->cacheKey()],
            fn (): array => $this->inner->search($criteria)->toArray(),
            $criteria->openAt === null ? null : self::OPEN_NOW_TTL,
        );

        return PlacePage::fromArray($data);
    }

    public function reviews(int $placeId, int $page = 1, int $perPage = 10): ReviewPage
    {
        /** @var array<string, mixed> $data */
        $data = $this->remember(['reviews', $placeId, $page, $perPage], fn (): array => $this->inner->reviews($placeId, $page, $perPage)->toArray());

        return ReviewPage::fromArray($data);
    }

    public function ratingSummary(int $placeId): RatingSummaryData
    {
        /** @var array<string, mixed> $data */
        $data = $this->remember(['rating', $placeId], fn (): array => $this->inner->ratingSummary($placeId)->toArray());

        return RatingSummaryData::fromArray($data);
    }
}
