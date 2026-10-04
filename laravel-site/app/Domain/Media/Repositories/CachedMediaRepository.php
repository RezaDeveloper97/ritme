<?php

declare(strict_types=1);

namespace App\Domain\Media\Repositories;

use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Media\Data\MediaData;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CachedRepository;

/**
 * Caches each media item as a plain array (not the DTO, so a deploy that changes MediaData never unserializes a
 * stale object) in the `media` namespace; misses are cached too. MediaObserver bumps the namespace.
 */
final class CachedMediaRepository extends CachedRepository implements MediaRepository
{
    public function __construct(private readonly MediaRepository $inner, CacheAside $cache)
    {
        parent::__construct($cache);
    }

    protected function namespace(): string
    {
        return 'media';
    }

    public function find(int $id): ?MediaData
    {
        /** @var array<string, mixed>|null $data */
        $data = $this->remember(['item', $id], fn (): ?array => $this->inner->find($id)?->toArray(), cacheNull: true);

        return $data === null ? null : MediaData::fromArray($data);
    }

    public function findMany(array $ids): array
    {
        $result = [];
        foreach (array_values(array_unique($ids)) as $id) {
            $media = $this->find($id);
            if ($media !== null) {
                $result[$id] = $media;
            }
        }

        return $result;
    }
}
