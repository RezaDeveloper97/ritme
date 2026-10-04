<?php

declare(strict_types=1);

namespace App\Domain\Blog\Data;

use App\Domain\Blog\Models\Tag;

/**
 * A tag for views; `postCount` = published posts (tag pages go noindex below a threshold, L4-02).
 */
final readonly class TagData
{
    public function __construct(
        public int $id,
        public string $name,
        public string $slug,
        public int $postCount = 0,
    ) {}

    public static function fromModel(Tag $tag, int $postCount = 0): self
    {
        return new self($tag->id, $tag->name, $tag->slug, $postCount);
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return ['id' => $this->id, 'name' => $this->name, 'slug' => $this->slug, 'postCount' => $this->postCount];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        return new self((int) $data['id'], (string) $data['name'], (string) $data['slug'], (int) ($data['postCount'] ?? 0));
    }
}
