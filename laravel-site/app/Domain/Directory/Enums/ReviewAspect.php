<?php

declare(strict_types=1);

namespace App\Domain\Directory\Enums;

/**
 * Optional per-aspect scores of a review (directory-place.html «نظر مادرها» bars).
 */
enum ReviewAspect: string
{
    case Cleanliness = 'cleanliness';
    case Staff = 'staff';
    case Facilities = 'facilities';
    case Access = 'access';

    public function label(): string
    {
        return match ($this) {
            self::Cleanliness => 'تمیزی',
            self::Staff => 'مربی',
            self::Facilities => 'امکانات',
            self::Access => 'رفت‌وآمد',
        };
    }
}
