<?php

declare(strict_types=1);

namespace App\Domain\Content\Tools;

use App\Domain\Content\Tools\Data\FertilityWindowResult;
use App\Domain\Content\Tools\Data\JalaliDay;

/**
 * Calendar estimate for regular cycles: ovulation ≈ 14 days before the next period (LMP + cycle − 14), fertile window
 * = the 5 days before it + that day. Less reliable for irregular cycles and never a contraception method.
 * Twin of `fertilityWindow()` in resources/js/lib/jalali.js.
 */
final class FertilityWindowCalculator
{
    /** Luteal phase length assumed by the calendar method. */
    public const LUTEAL_DAYS = 14;

    /** Days before ovulation that still count as fertile. */
    public const WINDOW_LEAD_DAYS = 5;

    public function calculate(JalaliDay $lmp, int $cycle): FertilityWindowResult
    {
        $ovulation = $lmp->addDays($cycle - self::LUTEAL_DAYS);

        return new FertilityWindowResult(
            windowFrom: $ovulation->addDays(-self::WINDOW_LEAD_DAYS),
            windowTo: $ovulation,
            ovulation: $ovulation,
            nextPeriod: $lmp->addDays($cycle),
        );
    }
}
