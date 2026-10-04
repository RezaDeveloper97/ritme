<?php

declare(strict_types=1);

namespace App\Domain\Blog\Repositories;

use App\Domain\Blog\Contracts\PostRepository;
use App\Domain\Blog\Data\PostCardData;
use App\Domain\Blog\Data\PostData;
use App\Domain\Blog\Data\PostPage;
use App\Domain\Blog\Enums\LifeStage;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CachedRepository;

/**
 * Caches plain arrays (not DTOs) in the `blog` namespace; Post/Category/Tag/Author observers bump it. Misses of
 * slug lookups are not cached (random slugs from crawlers would only fill the cache).
 */
final class CachedPostRepository extends CachedRepository implements PostRepository
{
    public function __construct(private readonly PostRepository $inner, CacheAside $cache)
    {
        parent::__construct($cache);
    }

    protected function namespace(): string
    {
        return 'blog';
    }

    public function findPublishedBySlug(string $slug): ?PostData
    {
        if ($slug === '') {
            return null;
        }

        /** @var array<string, mixed>|null $data */
        $data = $this->remember(['post', $slug], fn (): ?array => $this->inner->findPublishedBySlug($slug)?->toArray());

        return $data === null ? null : PostData::fromArray($data);
    }

    public function currentSlugFor(string $previousSlug): ?string
    {
        if ($previousSlug === '') {
            return null;
        }

        return $this->remember(['slug-redirect', $previousSlug], fn (): ?string => $this->inner->currentSlugFor($previousSlug));
    }

    public function latest(int $page = 1, int $perPage = 12, ?LifeStage $stage = null): PostPage
    {
        return $this->page(['latest', $stage->value ?? 'all', $page, $perPage], fn (): PostPage => $this->inner->latest($page, $perPage, $stage));
    }

    public function featured(): ?PostCardData
    {
        /** @var array<string, mixed>|null $data */
        $data = $this->remember('featured', fn (): ?array => $this->inner->featured()?->toArray(), cacheNull: true);

        return $data === null ? null : PostCardData::fromArray($data);
    }

    public function byCategory(int $categoryId, int $page = 1, int $perPage = 12): PostPage
    {
        return $this->page(['category', $categoryId, $page, $perPage], fn (): PostPage => $this->inner->byCategory($categoryId, $page, $perPage));
    }

    public function byTag(int $tagId, int $page = 1, int $perPage = 12): PostPage
    {
        return $this->page(['tag', $tagId, $page, $perPage], fn (): PostPage => $this->inner->byTag($tagId, $page, $perPage));
    }

    public function byAuthor(int $authorId, int $page = 1, int $perPage = 12): PostPage
    {
        return $this->page(['author', $authorId, $page, $perPage], fn (): PostPage => $this->inner->byAuthor($authorId, $page, $perPage));
    }

    public function related(int $postId, int $limit = 3): array
    {
        /** @var list<array<string, mixed>> $data */
        $data = $this->remember(['related', $postId, $limit], fn (): array => array_map(
            static fn (PostCardData $card): array => $card->toArray(),
            $this->inner->related($postId, $limit),
        ));

        return array_map(PostCardData::fromArray(...), $data);
    }

    /**
     * @param  list<string|int>  $key
     * @param  \Closure(): PostPage  $loader
     */
    private function page(array $key, \Closure $loader): PostPage
    {
        /** @var array<string, mixed> $data */
        $data = $this->remember($key, static fn (): array => $loader()->toArray());

        return PostPage::fromArray($data);
    }
}
