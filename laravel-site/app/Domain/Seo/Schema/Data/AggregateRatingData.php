<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Data;

use InvalidArgumentException;

/**
 * An aggregate of REAL, on-site reviews. Never build one from invented numbers: Google treats fake or self-serving
 * ratings as spam (manual action). Builders omit the rating when none is given.
 */
final readonly class AggregateRatingData
{
    public function __construct(
        public float $ratingValue,
        public int $ratingCount,
        public ?int $reviewCount = null,
        public float $bestRating = 5,
        public float $worstRating = 1,
    ) {
        if ($ratingCount < 1) {
            throw new InvalidArgumentException('An aggregate rating needs at least one real rating.');
        }
        if ($ratingValue < $worstRating || $ratingValue > $bestRating) {
            throw new InvalidArgumentException("Rating {$ratingValue} is outside {$worstRating}–{$bestRating}.");
        }
    }

    /**
     * @return array<string, mixed>
     */
    public function toNode(): array
    {
        return [
            '@type' => 'AggregateRating',
            'ratingValue' => round($this->ratingValue, 1),
            'ratingCount' => $this->ratingCount,
            'reviewCount' => $this->reviewCount,
            'bestRating' => $this->bestRating,
            'worstRating' => $this->worstRating,
        ];
    }
}
