<?php

namespace App\Enums;

enum CycleVariability: string
{
    case REGULAR = 'regular';
    case SEMI_IRREGULAR = 'semi_irregular';
    case IRREGULAR = 'irregular';

    public function label(string $locale = 'en'): string
    {
        return match($locale) {
            'fa' => match($this) {
                self::REGULAR => 'منظم',
                self::SEMI_IRREGULAR => 'نیمه‌منظم',
                self::IRREGULAR => 'نامنظم',
            },
            default => match($this) {
                self::REGULAR => 'Regular',
                self::SEMI_IRREGULAR => 'Semi-Irregular',
                self::IRREGULAR => 'Irregular',
            },
        };
    }

    public function uncertaintyRange(): int
    {
        return match($this) {
            self::REGULAR => 1,
            self::SEMI_IRREGULAR => 2,
            self::IRREGULAR => 3,
        };
    }

    public static function fromStdDev(float $stdDev): self
    {
        if ($stdDev <= 4) {
            return self::REGULAR;
        } elseif ($stdDev <= 7) {
            return self::SEMI_IRREGULAR;
        }
        return self::IRREGULAR;
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
