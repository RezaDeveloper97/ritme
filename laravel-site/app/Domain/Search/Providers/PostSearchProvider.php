<?php

declare(strict_types=1);

namespace App\Domain\Search\Providers;

use App\Domain\Blog\Data\PostCardData;
use App\Domain\Blog\Queries\SearchPosts;
use App\Domain\Blog\Support\BlogUrls;
use App\Domain\Search\Contracts\SearchProvider;
use App\Domain\Search\Data\SearchHit;
use App\Domain\Search\Support\SearchTerms;
use Illuminate\Support\Str;

/**
 * Published magazine posts (title, excerpt, body) via the Blog read-only query SearchPosts.
 */
final class PostSearchProvider implements SearchProvider
{
    public function __construct(private readonly BlogUrls $urls) {}

    public function key(): string
    {
        return 'posts';
    }

    public function label(): string
    {
        return (string) __('search.types.posts');
    }

    public function search(SearchTerms $terms, int $limit): array
    {
        $posts = (new SearchPosts($terms->tokens, SearchTerms::CHARACTER_MAP, $limit))->get();

        return array_map(fn (PostCardData $post): SearchHit => new SearchHit(
            type: $this->key(),
            typeLabel: $this->label(),
            title: $post->title,
            url: $this->urls->post($post->slug),
            snippet: $post->excerpt === null || trim($post->excerpt) === '' ? null : Str::limit(trim($post->excerpt), 180),
            score: $terms->touches($post->title) ? 2 : 1,
        ), $posts);
    }
}
