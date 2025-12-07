<?php

namespace App\Enums;

enum SexualActivity: string
{
    case HIGH_DESIRE = 'high_desire';
    case PROTECTED_INTERCOURSE = 'protected_intercourse';
    case UNPROTECTED_INTERCOURSE = 'unprotected_intercourse';
    case NO_DESIRE = 'no_desire';
    case DRYNESS = 'dryness';
    case BURNING = 'burning';
    case PAIN_DURING_INTERCOURSE = 'pain_during_intercourse';
    case BLEEDING_AFTER_INTERCOURSE = 'bleeding_after_intercourse';
    case LUBRICANT_USE = 'lubricant_use';

    public function label(): string
    {
        return match($this) {
            self::HIGH_DESIRE => 'High sexual desire',
            self::PROTECTED_INTERCOURSE => 'Protected intercourse',
            self::UNPROTECTED_INTERCOURSE => 'Unprotected intercourse',
            self::NO_DESIRE => 'No sexual desire',
            self::DRYNESS => 'Dryness',
            self::BURNING => 'Burning',
            self::PAIN_DURING_INTERCOURSE => 'Pain during intercourse',
            self::BLEEDING_AFTER_INTERCOURSE => 'Bleeding after intercourse',
            self::LUBRICANT_USE => 'Lubricant use',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
