<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Crawl;

use App\Domain\Seo\Analysis\ContentType;

/**
 * What an audited URL is in the admin: the analyser content type (null = not analysed), its edit page, the focus
 * keyword set there and the record's own body HTML (what the admin analysis panel analyses; null = none).
 */
final readonly class ContentTarget
{
    public function __construct(
        public ?ContentType $type = null,
        public ?string $editUrl = null,
        public string $focusKeyword = '',
        public ?string $contentHtml = null,
    ) {}
}
