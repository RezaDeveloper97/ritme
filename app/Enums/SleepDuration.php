<?php

namespace App\Enums;

enum SleepDuration: string
{
    case HOURS_0_3 = '0_3';
    case HOURS_3_6 = '3_6';
    case HOURS_6_9 = '6_9';
    case HOURS_9_PLUS = '9_plus';

    public function label(): string
    {
        return match($this) {
            self::HOURS_0_3 => '۰ تا ۳ ساعت',
            self::HOURS_3_6 => '۳ تا ۶ ساعت',
            self::HOURS_6_9 => '۶ تا ۹ ساعت',
            self::HOURS_9_PLUS => 'بیشتر از ۹ ساعت',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
