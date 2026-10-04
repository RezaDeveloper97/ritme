<?php

declare(strict_types=1);

namespace App\Domain\Content\Tools;

use App\Domain\Content\Tools\Data\DueDateResult;
use App\Domain\Content\Tools\Data\JalaliDay;

/**
 * Naegele's rule adjusted for cycle length: LMP + 280 days + (cycle − 28). An estimate only — most births happen in
 * the weeks around it and an ultrasound may move it. Twin of `dueDate()` in resources/js/lib/jalali.js.
 */
final class DueDateCalculator
{
    public const PREGNANCY_DAYS = 280;

    /** Half-width of the «بازه معمول» shown around the estimate. */
    public const RANGE_DAYS = 14;

    /** «الان حدود … هفته» is shown only while today is this many days (or fewer) after the LMP. */
    public const PROGRESS_MAX_DAYS = 299;

    /** `$today` null = no «how far along» line. */
    public function calculate(JalaliDay $lmp, int $cycle, ?JalaliDay $today = null): DueDateResult
    {
        $due = $lmp->addDays(self::PREGNANCY_DAYS + $cycle - CalculatorInput::DEFAULT_CYCLE);
        $elapsed = $today === null ? -1 : $today->number - $lmp->number;
        $showProgress = $elapsed >= 0 && $elapsed <= self::PROGRESS_MAX_DAYS;

        return new DueDateResult(
            due: $due,
            rangeFrom: $due->addDays(-self::RANGE_DAYS),
            rangeTo: $due->addDays(self::RANGE_DAYS),
            weeks: $showProgress ? intdiv($elapsed, 7) : null,
            days: $showProgress ? $elapsed % 7 : null,
        );
    }
}
