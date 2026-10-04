<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Data;

/**
 * One breadcrumb step. The same list feeds the visible `x-ui.breadcrumbs` and the BreadcrumbList node; the current
 * page is the last item and may have no URL.
 */
final readonly class BreadcrumbItem
{
    public function __construct(
        public string $name,
        public ?string $url = null,
    ) {}
}
