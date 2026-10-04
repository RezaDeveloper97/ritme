<?php

declare(strict_types=1);

namespace App\Domain\Blog\Repositories;

use App\Domain\Blog\Contracts\AuthorRepository;
use App\Domain\Blog\Data\AuthorData;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CachedRepository;

final class CachedAuthorRepository extends CachedRepository implements AuthorRepository
{
    public function __construct(private readonly AuthorRepository $inner, CacheAside $cache)
    {
        parent::__construct($cache);
    }

    protected function namespace(): string
    {
        return 'blog';
    }

    public function find(int $id): ?AuthorData
    {
        /** @var array<string, mixed>|null $data */
        $data = $this->remember(['author', $id], fn (): ?array => $this->inner->find($id)?->toArray());

        return $data === null ? null : AuthorData::fromArray($data);
    }

    public function findBySlug(string $slug): ?AuthorData
    {
        if ($slug === '') {
            return null;
        }

        /** @var array<string, mixed>|null $data */
        $data = $this->remember(['author-slug', $slug], fn (): ?array => $this->inner->findBySlug($slug)?->toArray());

        return $data === null ? null : AuthorData::fromArray($data);
    }
}
