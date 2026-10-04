<?php

declare(strict_types=1);

namespace App\Domain\Seo\Analysis;

/**
 * Outcome of one analyser check. Info is advice that does not move the score.
 */
enum Severity: string
{
    case Pass = 'pass';
    case Info = 'info';
    case Warning = 'warning';
    case Error = 'error';

    public function label(): string
    {
        return match ($this) {
            self::Pass => 'خوب',
            self::Info => 'نکته',
            self::Warning => 'قابل بهبود',
            self::Error => 'مشکل',
        };
    }

    /**
     * Share of the check's weight earned toward the score (null = not scored).
     */
    public function credit(): ?float
    {
        return match ($this) {
            self::Pass => 1.0,
            self::Warning => 0.5,
            self::Error => 0.0,
            self::Info => null,
        };
    }

    /**
     * Sort order for display: errors first.
     */
    public function rank(): int
    {
        return match ($this) {
            self::Error => 0,
            self::Warning => 1,
            self::Info => 2,
            self::Pass => 3,
        };
    }
}
