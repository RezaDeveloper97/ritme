<?php

declare(strict_types=1);

namespace App\Domain\Directory\Data;

use App\Domain\Seo\Schema\Data\AggregateRatingData;

/**
 * Rating of a place from its approved, non-demo reviews: overall average and count plus the average of each aspect
 * that was scored (ReviewAspect value => 1–5). Empty (count 0) when there are no real reviews.
 */
final readonly class RatingSummaryData
{
    /**
     * @param  array<string, float>  $aspects
     */
    public function __construct(
        public float $average = 0.0,
        public int $count = 0,
        public array $aspects = [],
    ) {}

    public function isEmpty(): bool
    {
        return $this->count < 1;
    }

    public function toSchema(): ?AggregateRatingData
    {
        return $this->isEmpty() ? null : new AggregateRatingData($this->average, $this->count, $this->count);
    }

    /**
     * @return array{average: float, count: int, aspects: array<string, float>}
     */
    public function toArray(): array
    {
        return ['average' => $this->average, 'count' => $this->count, 'aspects' => $this->aspects];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    public static function fromArray(array $data): self
    {
        return new self((float) $data['average'], (int) $data['count'], array_map(floatval(...), (array) ($data['aspects'] ?? [])));
    }
}
