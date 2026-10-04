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

    public function with(AuditIssue $issue): self
    {
        return new self($this->url, $this->title, $this->description, $this->indexable, [...$this->issues, $issue]);
    }

    public function errorCount(): int
    {
        return count(array_filter($this->issues, static fn (AuditIssue $i): bool => $i->severity === Severity::Error));
    }

    public function warningCount(): int
    {
        return count($this->issues) - $this->errorCount();
    }
}
