<?php

declare(strict_types=1);

namespace App\Domain\Blog\Contracts;

use App\Domain\Blog\Data\CategoryData;

/**
 * Magazine categories (with published post counts). Cached in the `blog` namespace.
 */
interface CategoryRepository
{
    /**
     * Every category ordered by sort_order, then name (chips on /blog).
     *
     * @return list<CategoryData>
     */
    public function all(): array;

    public function findBySlug(string $slug): ?CategoryData;
}
