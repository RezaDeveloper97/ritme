<?php

declare(strict_types=1);

namespace App\Domain\Seo\Analysis;

/**
 * One line of the analyser checklist: a stable key (tests, UI anchors), a short Persian label and the verdict.
 */
final readonly class SeoCheck
{
    public function __construct(
        public string $key,
        public string $label,
        public Severity $severity,
        public string $message,
        public int $weight = 1,
    ) {}

    public function passed(): bool
    {
        return $this->severity === Severity::Pass;
    }
}
