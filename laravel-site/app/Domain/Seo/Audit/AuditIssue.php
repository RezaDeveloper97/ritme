<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit;

final readonly class AuditIssue
{
    public function __construct(
        public Severity $severity,
        public string $code,
        public string $message,
    ) {}

    public static function error(string $code, string $message): self
    {
        return new self(Severity::Error, $code, $message);
    }

    public static function warning(string $code, string $message): self
    {
        return new self(Severity::Warning, $code, $message);
    }
}
