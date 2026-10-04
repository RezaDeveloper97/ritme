<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Data;

/**
 * One entry of an ItemList (a summary page linking to detail pages: articles, places, products).
 */
final readonly class ListEntry
{
    public function __construct(
        public string $url,
        public ?string $name = null,
        public ?string $imageUrl = null,
    ) {}
}
