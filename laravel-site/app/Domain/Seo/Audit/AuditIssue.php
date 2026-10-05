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

    public static function notice(string $code, string $message): self
    {
        return new self(Severity::Notice, $code, $message);
    }

    public function withSeverity(Severity $severity): self
    {
        return new self($severity, $this->code, $this->message);
    }
}
