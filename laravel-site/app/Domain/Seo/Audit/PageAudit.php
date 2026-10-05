<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit;

/**
 * The audit result of one rendered page.
 */
final readonly class PageAudit
{
    /**
     * @param  list<AuditIssue>  $issues
     */
    public function __construct(
        public string $url,
        public ?string $title,
        public ?string $description,
        public bool $indexable,
        public array $issues,
    ) {}

    public function with(AuditIssue ...$issues): self
    {
        return new self($this->url, $this->title, $this->description, $this->indexable, [...$this->issues, ...array_values($issues)]);
    }

    public function errorCount(): int
    {
        return $this->count(Severity::Error);
    }

    public function warningCount(): int
    {
        return $this->count(Severity::Warning);
    }

    public function noticeCount(): int
    {
        return $this->count(Severity::Notice);
    }

    public function count(Severity $severity): int
    {
        return count(array_filter($this->issues, static fn (AuditIssue $i): bool => $i->severity === $severity));
    }

    /** 100 minus the severity penalties of its findings, never below 0. */
    public function score(): int
    {
        $penalty = 0;
        foreach ($this->issues as $issue) {
            $penalty += $issue->severity->penalty();
        }

        return max(0, 100 - $penalty);
    }
}
