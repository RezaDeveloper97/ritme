<?php

declare(strict_types=1);

namespace App\Domain\Blog\Repositories;

use App\Domain\Blog\Contracts\TagRepository;
use App\Domain\Blog\Data\TagData;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CachedRepository;

final class CachedTagRepository extends CachedRepository implements TagRepository
{
    public function __construct(private readonly TagRepository $inner, CacheAside $cache)
    {
        parent::__construct($cache);
    }

    protected function namespace(): string
    {
        return 'blog';
    }

    public function findBySlug(string $slug): ?TagData
    {
        if ($slug === '') {
            return null;
        }

        /** @var array<string, mixed>|null $data */
        $data = $this->remember(['tag', $slug], fn (): ?array => $this->inner->findBySlug($slug)?->toArray());

        return $data === null ? null : TagData::fromArray($data);
    }
}
