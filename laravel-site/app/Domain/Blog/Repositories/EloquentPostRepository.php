<?php

declare(strict_types=1);

namespace App\Domain\Blog\Repositories;

use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Data\PostCardData;
use App\Domain\Blog\Data\PostData;
use App\Domain\Blog\Data\PostPage;
use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\PostSlug;
use App\Domain\Blog\Queries\LatestPosts;
use App\Domain\Blog\Queries\PostsByCategory;
use App\Domain\Blog\Queries\RelatedPosts;

final class EloquentPostRepository implements PostRepository
{
    public function findPublishedBySlug(string $slug): ?PostData
    {
        $post = Post::query()
            ->published()
            ->where('slug', $slug)
            ->with(['category', 'tags', 'author', 'reviewer'])
            ->first();

        return $post === null ? null : PostData::fromModel($post);
    }

    public function currentSlugFor(string $previousSlug): ?string
    {
        $postId = PostSlug::query()->where('slug', $previousSlug)->value('post_id');
        if ($postId === null) {
            return null;
        }

        $slug = Post::query()->published()->whereKey((int) $postId)->value('slug');

        return is_string($slug) ? $slug : null;
    }

    public function latest(int $page = 1, int $perPage = 12, ?LifeStage $stage = null): PostPage
    {
        return (new LatestPosts($page, $perPage, stage: $stage))->get();
    }

    public function featured(): ?PostCardData
    {
        $post = Post::query()
            ->published()
            ->where('is_featured', true)
            ->with('category')
            ->orderByDesc('published_at')
            ->orderByDesc('id')
            ->first();

        return $post === null ? null : PostCardData::fromModel($post);
    }

    public function byCategory(int $categoryId, int $page = 1, int $perPage = 12): PostPage
    {
        return (new PostsByCategory($categoryId, $page, $perPage))->get();
    }

    public function byTag(int $tagId, int $page = 1, int $perPage = 12): PostPage
    {
        return (new LatestPosts($page, $perPage, tagId: $tagId))->get();
    }

    public function byAuthor(int $authorId, int $page = 1, int $perPage = 12): PostPage
    {
        return (new LatestPosts($page, $perPage, authorId: $authorId))->get();
    }

    public function related(int $postId, int $limit = 3): array
    {
        return (new RelatedPosts($postId, $limit))->get();
    }
}
