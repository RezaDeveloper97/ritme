<?php

declare(strict_types=1);

namespace App\Domain\Directory\Booking\Support;

use App\Domain\Directory\Enums\Weekday;
use App\Domain\Directory\Support\OpeningHours;
use Carbon\CarbonImmutable;
use Carbon\CarbonInterface;

/**
 * Which days a parent may ask for. Not availability: the place confirms the real time. A day is bookable from today
 * up to HORIZON_DAYS ahead, unless the place's opening hours say it is closed on that weekday (unknown hours = any
 * day). The place page offers the next OFFERED open days starting TOMORROW — the page is full-page cached, and a
 * list that starts tomorrow is still valid (tomorrow → today) for the whole cache TTL after midnight.
 */
final class BookingDays
{
    public const HORIZON_DAYS = 60;

    public const OFFERED = 10;

    public const TIMEZONE = 'Asia/Tehran';

    public static function today(?CarbonInterface $now = null): CarbonImmutable
    {
        return CarbonImmutable::instance($now ?? CarbonImmutable::now())->setTimezone(self::TIMEZONE)->startOfDay();
    }

    public static function isBookable(CarbonInterface $date, OpeningHours $hours, ?CarbonInterface $now = null): bool
    {
        $day = CarbonImmutable::instance($date)->setTimezone(self::TIMEZONE)->startOfDay();
        $today = self::today($now);

        return $day->greaterThanOrEqualTo($today)
            && $day->lessThanOrEqualTo($today->addDays(self::HORIZON_DAYS))
            && ! self::isClosed($day, $hours);
    }

    /**
     * The next open days from tomorrow (relative to $now), at most OFFERED of them within the horizon.
     *
     * @return list<CarbonImmutable>
     */
    public static function offered(OpeningHours $hours, ?CarbonInterface $now = null, int $count = self::OFFERED): array
    {
        $today = self::today($now);
        $days = [];
        for ($i = 1; $i <= self::HORIZON_DAYS && count($days) < $count; $i++) {
            $day = $today->addDays($i);
            if (! self::isClosed($day, $hours)) {
                $days[] = $day;
            }
        }

        return $days;
    }

    /** Closed = the place lists hours for the week and this weekday has none. */
    private static function isClosed(CarbonImmutable $day, OpeningHours $hours): bool
    {
        if ($hours->isEmpty()) {
            return false;
        }
        $weekday = Weekday::fromDate($day);

        return $hours->isKnown($weekday) && $hours->ranges($weekday) === [];
    }
}
