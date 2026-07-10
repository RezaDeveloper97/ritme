<?php

namespace App\Enums;

/**
 * WHO body-mass-index bands. Boundaries follow the standard cut-offs used across
 * the app: underweight < 18.5, normal 18.5–25, overweight 25–30, obese ≥ 30.
 * A value lands in the higher band exactly on a boundary (e.g. 25.0 => overweight).
 */
enum BmiCategory: string
{
    case UNDERWEIGHT = 'underweight';
    case NORMAL = 'normal';
    case OVERWEIGHT = 'overweight';
    case OBESE = 'obese';

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }

    /** Classify a computed BMI value into its band. */
    public static function fromBmi(float $bmi): self
    {
        return match (true) {
            $bmi < 18.5 => self::UNDERWEIGHT,
            $bmi < 25.0 => self::NORMAL,
            $bmi < 30.0 => self::OVERWEIGHT,
            default => self::OBESE,
        };
    }

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::UNDERWEIGHT => 'کم‌وزن',
                self::NORMAL => 'طبیعی',
                self::OVERWEIGHT => 'اضافه‌وزن',
                self::OBESE => 'چاق',
            },
            default => match ($this) {
                self::UNDERWEIGHT => 'Underweight',
                self::NORMAL => 'Normal',
                self::OVERWEIGHT => 'Overweight',
                self::OBESE => 'Obese',
            },
        };
    }
}
