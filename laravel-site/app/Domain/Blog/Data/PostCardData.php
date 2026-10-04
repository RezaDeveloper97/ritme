<?php

declare(strict_types=1);

namespace App\Domain\Blog\Data;

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Blog\Models\Post;
use Carbon\CarbonImmutable;

/**
 * A post in lists (blog grid, related posts, stage readings, home). Covers are media ids for <x-picture>; without a
 * cover the card falls back to the stage gradient of `lifeStage` (docs/AUDIT.md §4.2).
 */
final readonly class PostCardData
{
    public function __construct(
        public int $id,
        public string $title,
        public string $slug,
        public ?string $excerpt,
        public ?CategoryData $category,
        public ?LifeStage $lifeStage,
        public ?int $coverMediaId,
        public ?int $mobileCoverMediaId,
        public int $readingTime,
        public CarbonImmutable $publishedAt,
        public CarbonImmutable $updatedContentAt,
        public bool $isFeatured = false,
    ) {}

    /**
     * Expects `category` loaded.
     */
    public static function fromModel(Post $post): self
    {
        $published = ($post->published_at ?? $post->created_at ?? now())->toImmutable();

        return new self(
            id: $post->id,
            title: $post->title,
            slug: $post->slug,
            excerpt: $post->excerpt,
            category: $post->category === null ? null : CategoryData::fromModel($post->category),
            lifeStage: $post->life_stage ?? $post->category?->life_stage,
            coverMediaId: $post->cover_media_id,
            mobileCoverMediaId: $post->cover_mobile_media_id,
            readingTime: $post->reading_time,
            publishedAt: $published,
            updatedContentAt: $post->updated_content_at?->toImmutable() ?? $published,
            isFeatured: $post->is_featured,
        );
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'id' => $this->id, 'title' => $this->title, 'slug' => $this->slug, 'excerpt' => $this->excerpt,
            'category' => $this->category?->toArray(), 'lifeStage' => $this->lifeStage?->value,
            'coverMediaId' => $this->coverMediaId, 'mobileCoverMediaId' => $this->mobileCoverMediaId,
            'readingTime' => $this->readingTime, 'publishedAt' => $this->publishedAt->toIso8601String(),
            'updatedContentAt' => $this->updatedContentAt->toIso8601String(), 'isFeatured' => $this->isFeatured,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        /** @var array<string, mixed>|null $category */
        $category = $data['category'] ?? null;

        return new self(
            id: (int) $data['id'],
            title: (string) $data['title'],
            slug: (string) $data['slug'],
            excerpt: isset($data['excerpt']) ? (string) $data['excerpt'] : null,
            category: $category === null ? null : CategoryData::fromArray($category),
            lifeStage: isset($data['lifeStage']) ? LifeStage::tryFrom((string) $data['lifeStage']) : null,
            coverMediaId: isset($data['coverMediaId']) ? (int) $data['coverMediaId'] : null,
            mobileCoverMediaId: isset($data['mobileCoverMediaId']) ? (int) $data['mobileCoverMediaId'] : null,
            readingTime: (int) $data['readingTime'],
            publishedAt: CarbonImmutable::parse((string) $data['publishedAt']),
            updatedContentAt: CarbonImmutable::parse((string) $data['updatedContentAt']),
            isFeatured: (bool) ($data['isFeatured'] ?? false),
        );
    }
}
