<?php

namespace App\Enums;

/**
 * Type of a user reminder shown on the home page
 * (یادآور دکتر / یادآور دارو and future reminder kinds).
 */
enum ReminderType: string
{
    case DOCTOR = 'doctor';
    case MEDICATION = 'medication';
    case APPOINTMENT = 'appointment';
    case CUSTOM = 'custom';

    public function label(string $locale = 'fa'): string
    {
        return $locale === 'fa'
            ? match ($this) {
                self::DOCTOR => 'یادآور دکتر',
                self::MEDICATION => 'یادآور دارو',
                self::APPOINTMENT => 'یادآور نوبت',
                self::CUSTOM => 'یادآور',
            }
            : match ($this) {
                self::DOCTOR => 'Doctor reminder',
                self::MEDICATION => 'Medication reminder',
                self::APPOINTMENT => 'Appointment reminder',
                self::CUSTOM => 'Reminder',
            };
    }

    public function icon(): string
    {
        return match ($this) {
            self::DOCTOR => 'stethoscope',
            self::MEDICATION => 'pill',
            self::APPOINTMENT => 'calendar-clock',
            self::CUSTOM => 'bell',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
