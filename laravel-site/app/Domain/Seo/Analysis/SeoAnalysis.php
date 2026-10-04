<?php

declare(strict_types=1);

namespace App\Domain\Seo\Analysis;

/**
 * The analyser result: 0–100 score and the checklist (errors first).
 */
final readonly class SeoAnalysis
{
    public const GOOD_SCORE = 80;

    public const NEEDS_WORK_SCORE = 60;

    /**
     * @param  list<SeoCheck>  $checks
     */
    public function __construct(
        public int $score,
        public array $checks,
    ) {}

    public function check(string $key): ?SeoCheck
    {
        foreach ($this->checks as $check) {
            if ($check->key === $key) {
                return $check;
            }
        }

        return null;
    }

    /**
     * @return list<SeoCheck>
     */
    public function withSeverity(Severity $severity): array
    {
        return array_values(array_filter($this->checks, static fn (SeoCheck $c): bool => $c->severity === $severity));
    }

    public function count(Severity $severity): int
    {
        return count($this->withSeverity($severity));
    }

    /**
     * Below the threshold or carrying an error (red line, cut title, missing alt …).
     */
    public function needsWork(): bool
    {
        return $this->score < self::NEEDS_WORK_SCORE || $this->count(Severity::Error) > 0;
    }

    /**
     * success | warning | danger — Filament colour of the score badge.
     */
    public function color(): string
    {
        return match (true) {
            $this->needsWork() => 'danger',
            $this->score >= self::GOOD_SCORE => 'success',
            default => 'warning',
        };
    }
}
