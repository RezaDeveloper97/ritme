<?php

declare(strict_types=1);

namespace App\Domain\Blog\Contracts;

use App\Domain\Blog\Data\TagData;

/**
 * Cached in the `blog` namespace.
 */
interface TagRepository
{
    public function findBySlug(string $slug): ?TagData;
}
