<?php

declare(strict_types=1);

namespace App\Domain\Blog\Data;

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Blog\Models\Post;
use App\Support\Html\HeadingAnchors;
use Carbon\CarbonImmutable;

/**
 * A full article for the article page. `body` and `sources` are sanitised HTML (safe to print unescaped); body
 * headings carry ids, `outline()` gives the TOC. Images in the body are `<img data-media-id>` for the renderer.
 *
 * @phpstan-import-type Heading from HeadingAnchors
 */
final readonly class PostData
{
    /**
     * @param  list<TagData>  $tags
     */
    public function __construct(
        public int $id,
        public string $title,
        public string $slug,
        public ?string $excerpt,
        public string $body,
        public ?string $sources,
        public ?CategoryData $category,
        public ?LifeStage $lifeStage,
        public array $tags,
        public ?AuthorData $author,
        public ?AuthorData $reviewer,
        public ?CarbonImmutable $reviewedAt,
        public ?int $coverMediaId,
        public ?int $mobileCoverMediaId,
        public int $readingTime,
        public int $wordCount,
        public CarbonImmutable $publishedAt,
        public CarbonImmutable $updatedContentAt,
        public bool $isFeatured = false,
    ) {}

    /**
     * Expects `category`, `tags`, `author`, `reviewer` loaded.
     */
    public static function fromModel(Post $post): self
    {
        $card = PostCardData::fromModel($post);

        return new self(
            id: $post->id,
            title: $post->title,
            slug: $post->slug,
            excerpt: $post->excerpt,
            body: $post->body,
            sources: $post->sources,
            category: $card->category,
            lifeStage: $card->lifeStage,
            tags: array_values($post->tags->sortBy('name')->map(TagData::fromModel(...))->all()),
            author: $post->author === null ? null : AuthorData::fromModel($post->author),
            reviewer: $post->reviewer === null ? null : AuthorData::fromModel($post->reviewer),
            reviewedAt: $post->reviewed_at?->toImmutable(),
            coverMediaId: $post->cover_media_id,
            mobileCoverMediaId: $post->cover_mobile_media_id,
            readingTime: $post->reading_time,
            wordCount: $post->word_count,
            publishedAt: $card->publishedAt,
            updatedContentAt: $card->updatedContentAt,
            isFeatured: $post->is_featured,
        );
    }

    public function toCard(): PostCardData
    {
        return new PostCardData(
            $this->id, $this->title, $this->slug, $this->excerpt, $this->category, $this->lifeStage,
            $this->coverMediaId, $this->mobileCoverMediaId, $this->readingTime, $this->publishedAt,
            $this->updatedContentAt, $this->isFeatured,
        );
    }

    /**
     * @return list<Heading>
     */
    public function outline(): array
    {
        return HeadingAnchors::outline($this->body);
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            ...$this->toCard()->toArray(),
            'body' => $this->body,
            'sources' => $this->sources,
            'tags' => array_map(static fn (TagData $tag): array => $tag->toArray(), $this->tags),
            'author' => $this->author?->toArray(),
            'reviewer' => $this->reviewer?->toArray(),
            'reviewedAt' => $this->reviewedAt?->toIso8601String(),
            'wordCount' => $this->wordCount,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        $card = PostCardData::fromArray($data);
        /** @var list<array<string, mixed>> $tags */
        $tags = (array) ($data['tags'] ?? []);
        /** @var array<string, mixed>|null $author */
        $author = $data['author'] ?? null;
        /** @var array<string, mixed>|null $reviewer */
        $reviewer = $data['reviewer'] ?? null;

        return new self(
            id: $card->id,
            title: $card->title,
            slug: $card->slug,
            excerpt: $card->excerpt,
            body: (string) ($data['body'] ?? ''),
            sources: isset($data['sources']) ? (string) $data['sources'] : null,
            category: $card->category,
            lifeStage: $card->lifeStage,
            tags: array_map(TagData::fromArray(...), $tags),
            author: $author === null ? null : AuthorData::fromArray($author),
            reviewer: $reviewer === null ? null : AuthorData::fromArray($reviewer),
            reviewedAt: isset($data['reviewedAt']) ? CarbonImmutable::parse((string) $data['reviewedAt']) : null,
            coverMediaId: $card->coverMediaId,
            mobileCoverMediaId: $card->mobileCoverMediaId,
            readingTime: $card->readingTime,
            wordCount: (int) ($data['wordCount'] ?? 0),
            publishedAt: $card->publishedAt,
            updatedContentAt: $card->updatedContentAt,
            isFeatured: $card->isFeatured,
        );
    }
}
