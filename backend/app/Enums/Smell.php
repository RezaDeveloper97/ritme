<?php

namespace App\Enums;

enum Smell: string
{
    case NORMAL = 'normal';
    case SLIGHTLY_UNUSUAL = 'slightly_unusual';
    case STRONG_UNPLEASANT = 'strong_unpleasant';

    public function label(): string
    {
        return match($this) {
            self::NORMAL => 'طبیعی',
            self::SLIGHTLY_UNUSUAL => 'کمی غیرطبیعی',
            self::STRONG_UNPLEASANT => 'تند',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
