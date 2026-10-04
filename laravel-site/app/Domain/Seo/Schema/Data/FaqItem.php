<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Data;

/**
 * A question with its answer (answer may hold simple HTML: p, a, ul/ol/li, strong, em, br — Google's allow-list).
 */
final readonly class FaqItem
{
    public function __construct(
        public string $question,
        public string $answer,
    ) {}
}
