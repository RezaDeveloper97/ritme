<?php

namespace App\Enums;

/**
 * Where the resolved `current_period_end` anchor came from (task.md §10.2). An
 * assumed end is derived from the effective period length and is never stored as
 * an actual log; `current_period_end_is_confirmed` must always accompany it.
 */
enum PeriodEndSource: string
{
    case USER_LOGGED = 'user_logged';
    case ASSUMED_FROM_EFFECTIVE_DURATION = 'assumed_from_effective_duration';
    case UNKNOWN = 'unknown';

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
