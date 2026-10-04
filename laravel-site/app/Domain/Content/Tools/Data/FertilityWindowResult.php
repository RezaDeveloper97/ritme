<?php

declare(strict_types=1);

namespace App\Domain\Content\Tools\Data;

/**
 * «پنجره تقریبی باروری»: the six days ending on the probable ovulation day, plus the probable next period.
 */
final readonly class FertilityWindowResult
{
    public function __construct(
        public JalaliDay $windowFrom,
        public JalaliDay $windowTo,
        public JalaliDay $ovulation,
        public JalaliDay $nextPeriod,
    ) {}
}
