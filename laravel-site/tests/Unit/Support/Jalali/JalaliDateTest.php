<?php

declare(strict_types=1);

use App\Support\Jalali\JalaliDate;

it('formats in Tehran time with Persian digits by default', function (): void {
    // 21:00 UTC on Mar 20 is already 00:30 on Nowruz in Tehran (+03:30).
    $date = JalaliDate::fromDateTime(new DateTimeImmutable('2024-03-20 21:00:00', new DateTimeZone('UTC')));

    expect([$date->year, $date->month, $date->day])->toBe([1403, 1, 2])
        ->and($date->format())->toBe('۲ فروردین ۱۴۰۳')
        ->and($date->format('Y/m/d H:i'))->toBe('۱۴۰۳/۰۱/۰۲ ۰۰:۳۰');
});

it('supports the format tokens', function (string $format, string $expected): void {
    $date = JalaliDate::fromDateTime(new DateTimeImmutable('2026-10-04 14:05:09', new DateTimeZone('Asia/Tehran')));

    expect($date->format($format, false))->toBe($expected);
})->with([
    'default' => [JalaliDate::DEFAULT_FORMAT, '12 مهر 1405'],
    'numeric' => ['Y/m/d', '1405/07/12'],
    'short' => ['y/n/j', '05/7/12'],
    'weekday' => ['l j F', 'یکشنبه 12 مهر'],
    'short weekday' => ['D', 'ی'],
    'time' => ['H:i:s', '14:05:09'],
    '12h' => ['g:i A', '2:05 ب.ظ'],
    'escaped' => ['\\Y Y', 'Y 1405'],
    'literal text' => ['j F، ساعت G', '12 مهر، ساعت 14'],
]);

it('builds from a Jalali day and converts back', function (): void {
    $date = JalaliDate::fromJalali(1403, 12, 30);

    expect($date->toDateTime()->format('Y-m-d H:i e'))->toBe('2025-03-20 00:00 Asia/Tehran')
        ->and($date->isLeapYear())->toBeTrue()
        ->and($date->monthName())->toBe('اسفند')
        ->and($date->weekdayName())->toBe('پنج‌شنبه')
        ->and($date->toDateString())->toBe('1403/12/30');
});

it('parses timestamps and strings', function (): void {
    expect(JalaliDate::parse('2024-03-20')->format('Y/m/d', false))->toBe('1403/01/01')
        ->and(JalaliDate::parse(1710880200)->format('Y/m/d H:i', false))->toBe('1403/01/01 00:00');
});

it('rejects unparseable strings', function (): void {
    JalaliDate::parse('not a date');
})->throws(InvalidArgumentException::class);

it('exposes the global jdate() helper', function (): void {
    expect(jdate(new DateTimeImmutable('2026-10-04 09:30', new DateTimeZone('Asia/Tehran'))))->toBe('۱۲ مهر ۱۴۰۵')
        ->and(jdate('2026-10-04 09:30', 'Y/m/d H:i'))->toBe('۱۴۰۵/۰۷/۱۲ ۰۹:۳۰')
        ->and(jdate('2026-10-04', 'Y', false))->toBe('1405')
        ->and(jdate())->toBe(JalaliDate::now()->format());
});
