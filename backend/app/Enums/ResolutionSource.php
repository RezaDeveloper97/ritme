<?php

namespace App\Enums;

/**
 * The primary source the engine relied on to resolve the target date's status
 * (task.md §30). This — not the backward-compat `is_predicted` flag — is the
 * authoritative answer to "where did this output come from".
 */
enum ResolutionSource: string
{
    case USER_LOGGED = 'user_logged';
    case USER_LOGGED_WITH_ASSUMED_END = 'user_logged_with_assumed_end';
    case PREDICTION = 'prediction';
    case ONBOARDING_BASED = 'onboarding_based';
    case DEFAULT_BASED = 'default_based';
    case UNKNOWN = 'unknown';

    /**
     * Backward-compat mapping (task.md §31): everything except a fully
     * user-logged resolution counts as predicted.
     */
    public function isPredicted(): bool
    {
        return $this !== self::USER_LOGGED;
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
