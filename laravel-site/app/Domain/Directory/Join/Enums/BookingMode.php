<?php

declare(strict_types=1);

namespace App\Domain\Directory\Join\Enums;

/**
 * How mothers reach the place (directory-business FAQ «اگر رزرو آنلاین نخواهم چه؟»): online booking, or phone only —
 * the place page then shows the number instead of the booking button.
 */
enum BookingMode: string
{
    case Online = 'online';
    case Phone = 'phone';

    public function label(): string
    {
        return match ($this) {
            self::Online => 'رزرو آنلاین',
            self::Phone => 'فقط تماس تلفنی',
        };
    }

    public function description(): string
    {
        return match ($this) {
            self::Online => 'وقت‌های خالی را اعلام می‌کنی و مادرها همین‌جا رزرو می‌کنند.',
            self::Phone => 'به‌جای دکمه رزرو، شماره مجموعه نمایش داده می‌شود.',
        };
    }
}
