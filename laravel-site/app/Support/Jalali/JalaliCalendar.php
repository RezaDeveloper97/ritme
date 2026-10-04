<?php

declare(strict_types=1);

namespace App\Support\Jalali;

use InvalidArgumentException;

/**
 * Jalali (Solar Hijri) ⇄ Gregorian day conversion — a port of the jalaali-js algorithm (Borkowski's break years),
 * exact for Jalali years -61…3177. Pure integer maths, no intl / package dependency.
 */
final class JalaliCalendar
{
    /** Years where the 33-year leap cycle is re-anchored. */
    private const BREAKS = [-61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210, 1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178];

    /** @return array{0: int, 1: int, 2: int} [jy, jm, jd] */
    public static function toJalali(int $gy, int $gm, int $gd): array
    {
        if (! checkdate($gm, $gd, $gy)) {
            throw new InvalidArgumentException("Invalid Gregorian date {$gy}-{$gm}-{$gd}.");
        }

        return self::d2j(self::g2d($gy, $gm, $gd));
    }

    /** @return array{0: int, 1: int, 2: int} [gy, gm, gd] */
    public static function toGregorian(int $jy, int $jm, int $jd): array
    {
        if (! self::isValid($jy, $jm, $jd)) {
            throw new InvalidArgumentException("Invalid Jalali date {$jy}-{$jm}-{$jd}.");
        }

        return self::d2g(self::j2d($jy, $jm, $jd));
    }

    /**
     * Julian Day Number of a Jalali date — for day arithmetic (add N days, days between two dates).
     */
    public static function toDayNumber(int $jy, int $jm, int $jd): int
    {
        if (! self::isValid($jy, $jm, $jd)) {
            throw new InvalidArgumentException("Invalid Jalali date {$jy}-{$jm}-{$jd}.");
        }

        return self::j2d($jy, $jm, $jd);
    }

    /** @return array{0: int, 1: int, 2: int} [jy, jm, jd] of a Julian Day Number. */
    public static function fromDayNumber(int $jdn): array
    {
        return self::d2j($jdn);
    }

    public static function isValid(int $jy, int $jm, int $jd): bool
    {
        return $jy >= self::BREAKS[0] && $jy < self::BREAKS[count(self::BREAKS) - 1]
            && $jm >= 1 && $jm <= 12 && $jd >= 1 && $jd <= self::monthLength($jy, $jm);
    }

    public static function isLeapYear(int $jy): bool
    {
        return self::cal($jy)['leap'] === 0;
    }

    public static function monthLength(int $jy, int $jm): int
    {
        if ($jm <= 6) {
            return 31;
        }
        if ($jm <= 11) {
            return 30;
        }

        return self::isLeapYear($jy) ? 30 : 29;
    }

    /** @return array{leap: int, gy: int, march: int} */
    private static function cal(int $jy): array
    {
        $count = count(self::BREAKS);
        if ($jy < self::BREAKS[0] || $jy >= self::BREAKS[$count - 1]) {
            throw new InvalidArgumentException("Jalali year {$jy} is out of range.");
        }

        $gy = $jy + 621;
        $leapJ = -14;
        $jp = self::BREAKS[0];
        $jump = 0;
        for ($i = 1; $i < $count; $i++) {
            $jm = self::BREAKS[$i];
            $jump = $jm - $jp;
            if ($jy < $jm) {
                break;
            }
            $leapJ += intdiv($jump, 33) * 8 + intdiv($jump % 33, 4);
            $jp = $jm;
        }

        $n = $jy - $jp;
        $leapJ += intdiv($n, 33) * 8 + intdiv($n % 33 + 3, 4);
        if ($jump % 33 === 4 && $jump - $n === 4) {
            $leapJ++;
        }

        $leapG = intdiv($gy, 4) - intdiv((intdiv($gy, 100) + 1) * 3, 4) - 150;
        $march = 20 + $leapJ - $leapG;

        if ($jump - $n < 6) {
            $n = $n - $jump + intdiv($jump + 4, 33) * 33;
        }
        $leap = (($n + 1) % 33 - 1) % 4;
        if ($leap === -1) {
            $leap = 4;
        }

        return ['leap' => $leap, 'gy' => $gy, 'march' => $march];
    }

    private static function j2d(int $jy, int $jm, int $jd): int
    {
        $r = self::cal($jy);

        return self::g2d($r['gy'], 3, $r['march']) + ($jm - 1) * 31 - intdiv($jm, 7) * ($jm - 7) + $jd - 1;
    }

    /** @return array{0: int, 1: int, 2: int} */
    private static function d2j(int $jdn): array
    {
        $gy = self::d2g($jdn)[0];
        $jy = $gy - 621;
        $r = self::cal($jy);
        $k = $jdn - self::g2d($gy, 3, $r['march']);

        if ($k >= 0) {
            if ($k <= 185) {
                return [$jy, 1 + intdiv($k, 31), $k % 31 + 1];
            }
            $k -= 186;
        } else {
            $jy--;
            $k += 179;
            if ($r['leap'] === 1) {
                $k++;
            }
        }

        return [$jy, 7 + intdiv($k, 30), $k % 30 + 1];
    }

    /** Gregorian date → Julian Day Number. */
    private static function g2d(int $gy, int $gm, int $gd): int
    {
        $d = intdiv(($gy + intdiv($gm - 8, 6) + 100100) * 1461, 4)
            + intdiv(153 * (($gm + 9) % 12) + 2, 5)
            + $gd - 34840408;

        return $d - intdiv(intdiv($gy + 100100 + intdiv($gm - 8, 6), 100) * 3, 4) + 752;
    }

    /** @return array{0: int, 1: int, 2: int} Julian Day Number → Gregorian date. */
    private static function d2g(int $jdn): array
    {
        $j = 4 * $jdn + 139361631;
        $j += intdiv(intdiv(4 * $jdn + 183187720, 146097) * 3, 4) * 4 - 3908;
        $i = intdiv($j % 1461, 4) * 5 + 308;
        $gd = intdiv($i % 153, 5) + 1;
        $gm = intdiv($i, 153) % 12 + 1;
        $gy = intdiv($j, 1461) - 100100 + intdiv(8 - $gm, 6);

        return [$gy, $gm, $gd];
    }
}
