<?php

declare(strict_types=1);

namespace App\Domain\Blog\Contracts;

use App\Domain\Blog\Data\PostCardData;
use App\Domain\Blog\Data\PostData;
use App\Domain\Blog\Data\PostPage;
use App\Domain\Blog\Enums\LifeStage;

/**
 * Read side of the magazine (published posts only). Cached in the `blog` namespace; lists are newest first.
 */
interface PostRepository
{
    public function findPublishedBySlug(string $slug): ?PostData;

    /**
     * Current slug of the published post that used to live at $previousSlug (slug history → 301), or null.
     */
    public function currentSlugFor(string $previousSlug): ?string;

    public function latest(int $page = 1, int $perPage = 12, ?LifeStage $stage = null): PostPage;

    /**
     * The newest featured post (blog hero), or null when none is featured.
     */
    public function featured(): ?PostCardData;

    /**
     * Posts of a category and its direct sub-categories.
     */
    public function byCategory(int $categoryId, int $page = 1, int $perPage = 12): PostPage;

    public function byTag(int $tagId, int $page = 1, int $perPage = 12): PostPage;

    public function byAuthor(int $authorId, int $page = 1, int $perPage = 12): PostPage;

    /**
     * Posts sharing tags / category with the given post, weighted by recency; topped up with the latest posts.
     *
     * @return list<PostCardData>
     */
    public function related(int $postId, int $limit = 3): array;
}
