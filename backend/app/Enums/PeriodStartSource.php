<?php

namespace App\Enums;

/**
 * Where the resolved `current_period_start` anchor came from (task.md §10.1).
 * Every start anchor in the API must be accompanied by its source.
 */
enum PeriodStartSource: string
{
    case USER_LOGGED = 'user_logged';
    case ONBOARDING_DECLARED = 'onboarding_declared';
    case PREDICTED_REFERENCE = 'predicted_reference';
    case UNKNOWN = 'unknown';

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
