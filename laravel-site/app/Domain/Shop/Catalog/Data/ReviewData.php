<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Data;

use App\Domain\Shop\Catalog\Models\ProductReview;

/**
 * An approved review for display. `isDemo` = seeded design sample: the page labels it «نمونه» and it never counts
 * towards the rating.
 */
final readonly class ReviewData
{
    public function __construct(
        public int $id,
        public string $authorName,
        public int $rating,
        public string $body,
        public ?string $variantLabel,
        public bool $isVerifiedPurchase,
        public bool $isDemo,
        public ?string $approvedAt,
    ) {}

    public static function fromModel(ProductReview $review): self
    {
        return new self(
            $review->id, $review->author_name, $review->rating, $review->body, $review->variant_label,
            $review->is_verified_purchase, $review->is_demo, $review->approved_at?->toIso8601String(),
        );
    }

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'id' => $this->id, 'authorName' => $this->authorName, 'rating' => $this->rating, 'body' => $this->body,
            'variantLabel' => $this->variantLabel, 'isVerifiedPurchase' => $this->isVerifiedPurchase,
            'isDemo' => $this->isDemo, 'approvedAt' => $this->approvedAt,
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        return new self(
            (int) $data['id'], (string) $data['authorName'], (int) $data['rating'], (string) $data['body'],
            isset($data['variantLabel']) ? (string) $data['variantLabel'] : null,
            (bool) $data['isVerifiedPurchase'], (bool) $data['isDemo'],
            isset($data['approvedAt']) ? (string) $data['approvedAt'] : null,
        );
    }
}
