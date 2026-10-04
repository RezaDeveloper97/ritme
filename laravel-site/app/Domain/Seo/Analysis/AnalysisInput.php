<?php

declare(strict_types=1);

namespace App\Domain\Seo\Analysis;

/**
 * Everything the analyser looks at, already resolved by the caller (no I/O inside the analyser):
 *
 *  - $title        the full <title> as Google sees it (SEO title or fallback, through the title template)
 *  - $heading      the page's visible h1 (post title, product / place name)
 *  - $description  the effective meta description
 *  - $slug         last URL segment (Persian or Latin, URL-encoded is fine)
 *  - $contentHtml  body HTML or plain text; null = the page body lives in a template (static pages) → content checks skipped
 *  - $ownHosts     hosts counted as internal links
 *  - $duplicateTitle / $duplicateDescription  another page already uses it; null = not checked
 */
final readonly class AnalysisInput
{
    /**
     * @param  list<string>  $ownHosts
     */
    public function __construct(
        public ContentType $type,
        public string $title,
        public string $heading = '',
        public string $description = '',
        public string $slug = '',
        public string $focusKeyword = '',
        public ?string $contentHtml = null,
        public array $ownHosts = [],
        public bool $cornerstone = false,
        public ?bool $duplicateTitle = null,
        public ?bool $duplicateDescription = null,
    ) {}
}
