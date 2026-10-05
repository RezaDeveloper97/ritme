<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Support;

use App\Domain\Shop\Ordering\Data\DeliverySlot;
use App\Domain\Shop\Ordering\Enums\DeliveryWindow;
use Carbon\CarbonImmutable;
use Carbon\CarbonInterface;

/**
 * Preferred delivery slots offered at checkout (design «زمان تحویل»: day + time window). A preference, not a booked
 * courier: the shop confirms the slot on the phone. Offered: the next OFFERED_DAYS days from tomorrow × every window
 * (Tehran calendar; the checkout page is no-store, so the list is always fresh). Accepted on submit: any day from
 * today (a form left open over midnight) up to HORIZON_DAYS ahead.
 */
final class DeliverySlots
{
    public const OFFERED_DAYS = 4;

    public const HORIZON_DAYS = 14;

    public const TIMEZONE = 'Asia/Tehran';

    public static function today(?CarbonInterface $now = null): CarbonImmutable
    {
        return CarbonImmutable::instance($now ?? CarbonImmutable::now())->setTimezone(self::TIMEZONE)->startOfDay();
    }

    /**
     * @return list<DeliverySlot>
     */
    public static function offered(?CarbonInterface $now = null): array
    {
        $today = self::today($now);
        $slots = [];
        for ($i = 1; $i <= self::OFFERED_DAYS; $i++) {
            foreach (DeliveryWindow::cases() as $window) {
                $slots[] = new DeliverySlot($today->addDays($i), $window, $i === 1);
            }
        }

        return $slots;
    }

    public static function isAcceptable(CarbonInterface $date, ?CarbonInterface $now = null): bool
    {
        $day = CarbonImmutable::instance($date)->setTimezone(self::TIMEZONE)->startOfDay();
        $today = self::today($now);

        return $day->greaterThanOrEqualTo($today) && $day->lessThanOrEqualTo($today->addDays(self::HORIZON_DAYS));
    }

    /** `2026-10-08` → that Tehran day, null when malformed. */
    public static function parse(string $value): ?CarbonImmutable
    {
        if (preg_match('/^(\d{4})-(\d{2})-(\d{2})$/', trim($value), $m) !== 1 || ! checkdate((int) $m[2], (int) $m[3], (int) $m[1])) {
            return null;
        }

        return CarbonImmutable::create((int) $m[1], (int) $m[2], (int) $m[3], 0, 0, 0, self::TIMEZONE);
    }
}
