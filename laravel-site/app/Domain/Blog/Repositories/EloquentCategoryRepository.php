<?php

declare(strict_types=1);

namespace App\Domain\Blog\Repositories;

use App\Domain\Blog\Contracts\CategoryRepository;
use App\Domain\Blog\Data\CategoryData;
use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;
use Illuminate\Database\Eloquent\Builder;

final class EloquentCategoryRepository implements CategoryRepository
{
    public function all(): array
    {
        return $this->query()
            ->orderBy('sort_order')
            ->orderBy('name')
            ->get()
            ->map(static fn (Category $category): CategoryData => CategoryData::fromModel($category, (int) $category->getAttribute('published_posts_count')))
            ->values()
            ->all();
    }

    public function findBySlug(string $slug): ?CategoryData
    {
        $category = $this->query()->where('slug', $slug)->first();

        return $category === null ? null : CategoryData::fromModel($category, (int) $category->getAttribute('published_posts_count'));
    }

    /**
     * @return Builder<Category>
     */
    private function query(): Builder
    {
        return Category::query()->withCount(['posts as published_posts_count' => static function (Builder $q): void {
            /** @var Builder<Post> $q */
            $q->published();
        }]);
    }
}
