<?php

declare(strict_types=1);

namespace App\Domain\Blog\Support;

use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Data\AuthorData;
use App\Domain\Blog\Data\PostPage;

/**
 * Which magazine lists may be indexed (L4-02, L7-05b) — one rule for BlogListingController (robots) and
 * PagesSitemapProvider (`/blog` in `pages.xml`):
 *  - a list without published posts is thin → `noindex,follow` and out of the sitemap;
 *  - an author page without posts is indexable only for a medical reviewer with a real bio (E-E-A-T credentials page);
 *    a placeholder bio («[…]») or one shorter than a meta description is not a real bio.
 */
final class PostListIndexing
{
    /** A bio shorter than this cannot stand in for the meta description (SeoAuditor::DESCRIPTION_MIN). */
    public const MIN_BIO_LENGTH = 70;

    public function __construct(private readonly PostRepository $posts) {}

    public static function listIndexable(PostPage $list): bool
    {
        return $list->total > 0;
    }

    public static function authorIndexable(AuthorData $author, PostPage $list): bool
    {
        return self::listIndexable($list) || ($author->isMedicalReviewer && self::hasRealBio($author->bio));
    }

    /**
     * A bio an editor actually wrote: not empty, no `[placeholder]` from the seeder, at least MIN_BIO_LENGTH characters.
     */
    public static function hasRealBio(?string $bio): bool
    {
        $bio = trim((string) $bio);

        return $bio !== ''
            && preg_match('/\[[^\]]*\]/u', $bio) !== 1
            && mb_strlen($bio) >= self::MIN_BIO_LENGTH;
    }

    /**
     * `/blog` (first page, no filter) — same cached repository read as the controller.
     */
    public function indexIndexable(): bool
    {
        return self::listIndexable($this->posts->latest(1, 1));
    }
}
