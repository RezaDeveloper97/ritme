<?php

declare(strict_types=1);

namespace App\Domain\Blog\Queries;

use App\Domain\Blog\Data\PostPage;
use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;

/**
 * Published posts of a category and its direct children (categories are tree-light: one level of nesting).
 */
final class PostsByCategory
{
    use PaginatesPosts;

    public function __construct(
        private readonly int $categoryId,
        private readonly int $page = 1,
        private readonly int $perPage = 12,
    ) {}

    public function get(): PostPage
    {
        $ids = Category::query()->where('parent_id', $this->categoryId)->pluck('id')->map(intval(...))->all();
        $ids[] = $this->categoryId;

        return $this->paginate(
            Post::query()->published()->whereIn('category_id', array_values(array_unique($ids))),
            $this->page,
            $this->perPage,
        );
    }
}
