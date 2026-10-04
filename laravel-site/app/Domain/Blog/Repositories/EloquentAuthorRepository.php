<?php

declare(strict_types=1);

namespace App\Domain\Blog\Repositories;

use App\Domain\Blog\Contracts\AuthorRepository;
use App\Domain\Blog\Data\AuthorData;
use App\Domain\Blog\Models\Author;

final class EloquentAuthorRepository implements AuthorRepository
{
    public function find(int $id): ?AuthorData
    {
        $author = Author::query()->find($id);

        return $author === null ? null : AuthorData::fromModel($author);
    }

    public function findBySlug(string $slug): ?AuthorData
    {
        $author = Author::query()->where('slug', $slug)->first();

        return $author === null ? null : AuthorData::fromModel($author);
    }
}
