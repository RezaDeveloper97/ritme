<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit;

/**
 * Everything one audit run found. `truncated` = stopped by its page or time bound (cross-page checks that need the
 * whole site, like orphan pages, are skipped then).
 */
final readonly class AuditResult
{
    /**
     * @param  list<AuditedPage>  $pages
     * @param  list<array{path: string, status: int, type: string}>  $skipped  non-HTML / redirecting seeds
     */
    public function __construct(
        public array $pages,
        public array $skipped,
        public bool $truncated,
        public ?string $truncatedBy,
        public int $milliseconds,
        public int $fetches,
    ) {}

    public function count(Severity $severity): int
    {
        return array_sum(array_map(static fn (AuditedPage $page): int => $page->audit->count($severity), $this->pages));
    }

    /** Site health 0–100: the mean page score (100 − severity penalties per page). */
    public function score(): int
    {
        if ($this->pages === []) {
            return 100;
        }

        return (int) round(array_sum(array_map(static fn (AuditedPage $page): int => $page->audit->score(), $this->pages)) / count($this->pages));
    }
}
