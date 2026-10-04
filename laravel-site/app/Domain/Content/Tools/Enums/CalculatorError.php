<?php

declare(strict_types=1);

namespace App\Domain\Content\Tools\Enums;

/**
 * Why a calculator could not answer. The value is the key under `tools.errors.*` (lang/fa/tools.php) and the code
 * resources/js/lib/jalali.js returns, so both sides show the same Persian message.
 */
enum CalculatorError: string
{
    case InvalidDate = 'invalid_date';
    case FutureDate = 'future_date';
    case CycleOutOfRange = 'cycle_range';

    /** The input the message belongs to (`lmp` or `cycle`). */
    public function field(): string
    {
        return $this === self::CycleOutOfRange ? 'cycle' : 'lmp';
    }
}
