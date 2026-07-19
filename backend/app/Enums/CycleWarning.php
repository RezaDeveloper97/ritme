<?php

namespace App\Enums;

/**
 * Stable warning codes attached to an engine output (task.md §32). Warnings are
 * soft signals for the client — they never block the resolve itself.
 */
enum CycleWarning: string
{
    case PERIOD_END_MISSING = 'period_end_missing';
    case PERIOD_END_MISSING_WARNING_CAP_EXCEEDED = 'period_end_missing_warning_cap_exceeded';
    case PERIOD_END_MISSING_HARD_CAP_EXCEEDED = 'period_end_missing_hard_cap_exceeded';

    case LOW_HISTORY = 'low_history';
    case HIGH_CYCLE_VARIABILITY = 'high_cycle_variability';

    case SHORT_CYCLE_OUTLIER_DETECTED = 'short_cycle_outlier_detected';
    case LONG_CYCLE_OUTLIER_DETECTED = 'long_cycle_outlier_detected';

    case ONBOARDING_DATA_USED = 'onboarding_data_used';
    case DEFAULT_VALUES_USED = 'default_values_used';
    case PREDICTION_BASED_OUTPUT = 'prediction_based_output';

    case PREDICTED_PERIOD_OVERDUE = 'predicted_period_overdue';
    case CYCLE_UNRESOLVED_AFTER_EXPECTED_PERIOD = 'cycle_unresolved_after_expected_period';

    case INSUFFICIENT_ANCHOR_DATA = 'insufficient_anchor_data';

    /** Warnings that cannot be cleared without the user logging or confirming data (§33). */
    public function requiresUserInput(): bool
    {
        return match ($this) {
            self::PERIOD_END_MISSING_WARNING_CAP_EXCEEDED,
            self::PERIOD_END_MISSING_HARD_CAP_EXCEEDED,
            self::CYCLE_UNRESOLVED_AFTER_EXPECTED_PERIOD,
            self::INSUFFICIENT_ANCHOR_DATA => true,
            default => false,
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
