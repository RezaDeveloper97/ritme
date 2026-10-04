<?php

declare(strict_types=1);

namespace App\Domain\Blog\Support;

use App\Domain\Blog\Data\TagData;
use Illuminate\Contracts\Config\Repository as Config;

/**
 * Tag pages with fewer than N published posts are thin: they render `noindex,follow` and stay out of the
 * `blog-tags` sitemap. N = `blog.tag_index_min_posts` (default 3; no config file yet — add `config/blog.php` to change it).
 */
final class TagIndexing
{
    public const DEFAULT_MIN_POSTS = 3;

    public function __construct(private readonly Config $config) {}

    public function minPosts(): int
    {
        return max(1, (int) $this->config->get('blog.tag_index_min_posts', self::DEFAULT_MIN_POSTS));
    }

    public function indexable(TagData $tag): bool
    {
        return $tag->postCount >= $this->minPosts();
    }
}
