<?php

declare(strict_types=1);

namespace App\Domain\Directory\Data;

use App\Domain\Directory\Models\PlaceReview;
use Carbon\CarbonImmutable;

/**
 * An approved review for the place page. `isDemo` = seeded design sample: the view must label it as such; it never
 * counts towards the rating.
 */
final readonly class ReviewData
{
    /**
     * @param  array<string, int>  $aspects  ReviewAspect value => 1–5
     */
    public function __construct(
        public int $id,
        public string $authorName,
        public int $rating,
        public array $aspects,
        public string $body,
        public CarbonImmutable $createdAt,
        public bool $isDemo = false,
    ) {}

    public static function fromModel(PlaceReview $review): self
    {
        return new self(
            id: $review->id,
            authorName: $review->author_name,
            rating: $review->rating,
            aspects: array_map(intval(...), $review->aspects ?? []),
            body: $review->body,
            createdAt: ($review->approved_at ?? $review->created_at ?? now())->toImmutable(),
            isDemo: $review->is_demo,
        );
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'id' => $this->id, 'authorName' => $this->authorName, 'rating' => $this->rating, 'aspects' => $this->aspects,
            'body' => $this->body, 'createdAt' => $this->createdAt->toIso8601String(), 'isDemo' => $this->isDemo,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        return new self(
            id: (int) $data['id'],
            authorName: (string) $data['authorName'],
            rating: (int) $data['rating'],
            aspects: array_map(intval(...), (array) ($data['aspects'] ?? [])),
            body: (string) $data['body'],
            createdAt: CarbonImmutable::parse((string) $data['createdAt']),
            isDemo: (bool) ($data['isDemo'] ?? false),
        );
    }
}
