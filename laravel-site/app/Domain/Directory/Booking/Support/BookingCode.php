<?php

declare(strict_types=1);

namespace App\Domain\Directory\Booking\Support;

/**
 * The public code of a booking request — the only key of its /directory/booked/{code} page, so it must not be
 * guessable: 12 random symbols from a 31-letter alphabet without look-alikes (no 0/O, 1/I/L), ~59 bits, grouped
 * 4-4-4 for reading out on the phone (`K3M9-QX2T-D7XA`). Lookups are case-insensitive (`normalize`).
 */
final class BookingCode
{
    public const ALPHABET = '23456789ABCDEFGHJKMNPQRSTUVWXYZ';

    public const PATTERN = '/^[2-9A-HJKMNP-Z]{4}-[2-9A-HJKMNP-Z]{4}-[2-9A-HJKMNP-Z]{4}$/';

    public static function generate(): string
    {
        $max = strlen(self::ALPHABET) - 1;
        $symbols = '';
        for ($i = 0; $i < 12; $i++) {
            $symbols .= self::ALPHABET[random_int(0, $max)];
        }

        return implode('-', str_split($symbols, 4));
    }

    /** Upper-cased and trimmed; anything longer than a code is cut so lookups stay cheap. */
    public static function normalize(string $code): string
    {
        return strtoupper(substr(trim($code), 0, 20));
    }
}
