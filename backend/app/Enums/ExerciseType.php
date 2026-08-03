<?php

namespace App\Enums;

/**
 * Kind of physical activity recorded on a daily health log
 * (daily_health_logs.exercise_type).
 */
enum ExerciseType: string
{
    case WALKING = 'walking';
    case RUNNING = 'running';
    case CYCLING = 'cycling';
    case GYM = 'gym';
    case YOGA = 'yoga';
    case SWIMMING = 'swimming';
    case DANCE = 'dance';
    case TEAM_SPORT = 'team_sport';
    case OTHER = 'other';

    public function label(string $locale = 'fa'): string
    {
        return $locale === 'fa'
            ? match ($this) {
                self::WALKING => 'پیاده‌روی',
                self::RUNNING => 'دویدن',
                self::CYCLING => 'دوچرخه',
                self::GYM => 'باشگاه / تمرین قدرتی',
                self::YOGA => 'یوگا / پیلاتس',
                self::SWIMMING => 'شنا',
                self::DANCE => 'رقص',
                self::TEAM_SPORT => 'ورزش تیمی',
                self::OTHER => 'سایر',
            }
        : match ($this) {
            self::WALKING => 'Walking',
            self::RUNNING => 'Running',
            self::CYCLING => 'Cycling',
            self::GYM => 'Gym / strength training',
            self::YOGA => 'Yoga / pilates',
            self::SWIMMING => 'Swimming',
            self::DANCE => 'Dance',
            self::TEAM_SPORT => 'Team sport',
            self::OTHER => 'Other',
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
