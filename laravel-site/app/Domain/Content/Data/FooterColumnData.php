<?php

declare(strict_types=1);

namespace App\Domain\Content\Data;

/**
 * One footer link column, rendered as `<nav aria-label="{title}">` with a list.
 */
final readonly class FooterColumnData
{
    /**
     * @param  list<LinkData>  $links
     */
    public function __construct(
        public string $title,
        public array $links,
    ) {}
}
