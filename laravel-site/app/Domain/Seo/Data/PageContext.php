<?php

declare(strict_types=1);

namespace App\Domain\Seo\Data;

/**
 * What SeoManager needs to know about the current page: its full URL (with query) and route name.
 */
final readonly class PageContext
{
    public function __construct(
        public string $url,
        public ?string $routeName = null,
    ) {}
}
