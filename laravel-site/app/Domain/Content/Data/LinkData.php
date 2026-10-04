<?php

declare(strict_types=1);

namespace App\Domain\Content\Data;

/**
 * A plain labelled link (footer columns, socials).
 */
final readonly class LinkData
{
    public function __construct(
        public string $label,
        public string $url,
        public bool $external = false,
    ) {}
}
