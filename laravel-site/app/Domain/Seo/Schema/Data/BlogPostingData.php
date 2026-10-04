<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Data;

use App\Domain\Seo\Data\SeoImage;

/**
 * A magazine article. Dates are ISO 8601 with timezone (`2026-10-04T09:30:00+03:30`).
 */
final readonly class BlogPostingData
{
    /**
     * @param  list<PersonData>  $authors
     * @param  list<string>  $keywords
     */
    public function __construct(
        public string $url,
        public string $headline,
        public string $datePublished,
        public array $authors,
        public ?string $dateModified = null,
        public ?string $description = null,
        public ?SeoImage $image = null,
        public ?string $section = null,
        public array $keywords = [],
        public ?int $wordCount = null,
    ) {}
}
