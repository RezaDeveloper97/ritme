<?php

declare(strict_types=1);

namespace App\Domain\Content\Tools\Data;

/**
 * «تاریخ تقریبی زایمان»: the estimate, the usual ±2-week range around it and — when the LMP is 0–299 days ago —
 * how far along the pregnancy is today (whole weeks + remaining days).
 */
final readonly class DueDateResult
{
    public function __construct(
        public JalaliDay $due,
        public JalaliDay $rangeFrom,
        public JalaliDay $rangeTo,
        public ?int $weeks = null,
        public ?int $days = null,
    ) {}
}
