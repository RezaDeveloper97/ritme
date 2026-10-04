<?php

declare(strict_types=1);

namespace App\Domain\Blog\Contracts;

use App\Domain\Blog\Data\AuthorData;

/**
 * Authors and medical reviewers. Cached in the `blog` namespace.
 */
interface AuthorRepository
{
    public function find(int $id): ?AuthorData;

    public function findBySlug(string $slug): ?AuthorData;
}
