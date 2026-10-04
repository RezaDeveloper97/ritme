<?php

declare(strict_types=1);

namespace App\Domain\Directory\Data;

/**
 * A visitor's review as validated by the delivery layer (L5-03 form request): rating 1–5, optional aspect scores.
 */
final readonly class ReviewSubmission
{
    /**
     * @param  array<string, int>  $aspects  ReviewAspect value => 1–5
     */
    public function __construct(
        public string $placeSlug,
        public string $authorName,
        public int $rating,
        public string $body,
        public array $aspects = [],
        public ?string $ip = null,
    ) {}
}
