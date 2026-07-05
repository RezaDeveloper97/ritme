<?php

namespace App\Enums;

/**
 * Type of an in-app user notification (drives the header bell badge & list).
 */
enum NotificationType: string
{
    case REMINDER = 'reminder';
    case ALERT = 'alert';
    case TIP = 'tip';
    case ACHIEVEMENT = 'achievement';
    case SYSTEM = 'system';

    public function label(string $locale = 'fa'): string
    {
        return $locale === 'fa'
            ? match ($this) {
                self::REMINDER => 'یادآوری',
                self::ALERT => 'هشدار',
                self::TIP => 'نکته',
                self::ACHIEVEMENT => 'دستاورد',
                self::SYSTEM => 'سیستمی',
            }
            : match ($this) {
                self::REMINDER => 'Reminder',
                self::ALERT => 'Alert',
                self::TIP => 'Tip',
                self::ACHIEVEMENT => 'Achievement',
                self::SYSTEM => 'System',
            };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
