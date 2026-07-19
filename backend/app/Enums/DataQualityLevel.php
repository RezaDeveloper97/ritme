<?php

namespace App\Enums;

/**
 * Deterministic data-quality grade for an engine output (task.md §28). Assignment
 * precedence is insufficient → poor → good → partial: the first matching grade wins.
 */
enum DataQualityLevel: string
{
    case GOOD = 'good';
    case PARTIAL = 'partial';
    case POOR = 'poor';
    case INSUFFICIENT = 'insufficient';

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
