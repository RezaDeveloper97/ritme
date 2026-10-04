<?php

declare(strict_types=1);

namespace App\Domain\Blog\Queries;

use App\Domain\Blog\Data\PostCardData;
use App\Domain\Blog\Data\PostPage;
use App\Domain\Blog\Models\Post;
use Illuminate\Database\Eloquent\Builder;

/**
 * Turns a published-posts query into a PostPage (newest first, stable order by id).
 */
trait PaginatesPosts
{
    /**
     * @param  Builder<Post>  $query
     */
    private function paginate(Builder $query, int $page, int $perPage): PostPage
    {
        $page = max(1, $page);
        $perPage = max(1, min(100, $perPage));
        $total = (clone $query)->count();

        $items = $total === 0 || ($page - 1) * $perPage >= $total ? [] : $query
            ->with('category')
            ->orderByDesc('published_at')
            ->orderByDesc('id')
            ->forPage($page, $perPage)
            ->get()
            ->map(PostCardData::fromModel(...))
            ->values()
            ->all();

        return new PostPage($items, $total, $page, $perPage);
    }
}
