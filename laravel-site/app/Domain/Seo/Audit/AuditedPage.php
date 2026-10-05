<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit;

use App\Domain\Seo\Analysis\ContentType;

/**
 * One audited URL of a run: the rule findings (PageAudit) plus what the report shows next to them.
 */
final readonly class AuditedPage
{
    public const SOURCE_ROUTE = 'route';

    public const SOURCE_SITEMAP = 'sitemap';

    public const SOURCE_LINK = 'link';

    public const SOURCE_PATH = 'path';

    public function __construct(
        public PageAudit $audit,
        public int $status,
        public string $source,
        public bool $inSitemap = false,
        public int $milliseconds = 0,
        public int $htmlBytes = 0,
        public int $requests = 0,
        public ?string $routeName = null,
        public ?ContentType $contentType = null,
        public ?int $analysisScore = null,
        public int $inbound = 0,
        public ?string $editUrl = null,
        public ?string $contentHash = null,
    ) {}

    public function path(): string
    {
        return $this->audit->url;
    }

    public function withAudit(PageAudit $audit): self
    {
        return new self($audit, $this->status, $this->source, $this->inSitemap, $this->milliseconds, $this->htmlBytes,
            $this->requests, $this->routeName, $this->contentType, $this->analysisScore, $this->inbound, $this->editUrl, $this->contentHash);
    }

    public function with(AuditIssue ...$issues): self
    {
        return $this->withAudit($this->audit->with(...$issues));
    }

    public function withInbound(int $inbound): self
    {
        return new self($this->audit, $this->status, $this->source, $this->inSitemap, $this->milliseconds, $this->htmlBytes,
            $this->requests, $this->routeName, $this->contentType, $this->analysisScore, $inbound, $this->editUrl, $this->contentHash);
    }

    public function withAnalysisScore(?int $score): self
    {
        return new self($this->audit, $this->status, $this->source, $this->inSitemap, $this->milliseconds, $this->htmlBytes,
            $this->requests, $this->routeName, $this->contentType, $score, $this->inbound, $this->editUrl, $this->contentHash);
    }
}
