<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit;

/**
 * How far one audit run may go. A run with explicit `paths` audits only those (no sitemap, no link discovery, no
 * orphan check) — the quick CI / developer mode. Otherwise it crawls every parameterless GET route, every sitemap URL
 * (posts, places, products, landings …) and the internal links found on the way, within `maxPages` HTML pages and
 * `timeLimit` seconds. `priority` paths are audited first (the previous run's failing / new pages), so a run cut short
 * by its bounds still covers what most likely changed. `assumeProduction` renders with production robots (outside
 * production every page is forced noindex, which would hide every indexability problem).
 */
final readonly class AuditOptions
{
    public const DEFAULT_MAX_PAGES = 500;

    public const DEFAULT_TIME_LIMIT = 240;

    /**
     * @param  list<string>  $paths
     * @param  list<string>  $priority
     */
    public function __construct(
        public array $paths = [],
        public int $maxPages = self::DEFAULT_MAX_PAGES,
        public int $timeLimit = self::DEFAULT_TIME_LIMIT,
        public bool $followLinks = true,
        public bool $includeSitemap = true,
        public bool $assumeProduction = true,
        public array $priority = [],
    ) {}

    public function isFullCrawl(): bool
    {
        return $this->paths === [];
    }

    /**
     * @param  list<string>  $priority
     */
    public function withPriority(array $priority): self
    {
        return new self($this->paths, $this->maxPages, $this->timeLimit, $this->followLinks, $this->includeSitemap, $this->assumeProduction, $priority);
    }
}
