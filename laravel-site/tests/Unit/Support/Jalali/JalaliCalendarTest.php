<?php

declare(strict_types=1);

use App\Support\Jalali\JalaliCalendar;

/** Known Gregorian ⇄ Jalali pairs (Nowruz boundaries, leap-year Esfand 30, historic dates). */
dataset('known dates', [
    'nowruz 1403 (Mar 20, 2024)' => [[2024, 3, 20], [1403, 1, 1]],
    'eve of nowruz 1403' => [[2024, 3, 19], [1402, 12, 29]],
    'esfand 30, leap 1403' => [[2025, 3, 20], [1403, 12, 30]],
    'nowruz 1404' => [[2025, 3, 21], [1404, 1, 1]],
    'esfand 29, common 1404' => [[2026, 3, 20], [1404, 12, 29]],
    'nowruz 1405' => [[2026, 3, 21], [1405, 1, 1]],
    'mehr 12, 1405' => [[2026, 10, 4], [1405, 7, 12]],
    'esfand 30, leap 1399' => [[2021, 3, 20], [1399, 12, 30]],
    'shahrivar 31 → mehr 1' => [[2024, 9, 22], [1403, 7, 1]],
    'bahman 22, 1357' => [[1979, 2, 11], [1357, 11, 22]],
    'gregorian leap day' => [[2024, 2, 29], [1402, 12, 10]],
    'start of 2000' => [[2000, 1, 1], [1378, 10, 11]],
    'esfand 30, leap 1408' => [[2030, 3, 20], [1408, 12, 30]],
]);

it('converts Gregorian to Jalali', function (array $gregorian, array $jalali): void {
    expect(JalaliCalendar::toJalali(...$gregorian))->toBe($jalali);
})->with('known dates');

it('converts Jalali to Gregorian', function (array $gregorian, array $jalali): void {
    expect(JalaliCalendar::toGregorian(...$jalali))->toBe($gregorian);
})->with('known dates');

it('knows Jalali leap years', function (): void {
    $leap = array_values(array_filter(range(1390, 1412), JalaliCalendar::isLeapYear(...)));

    expect($leap)->toBe([1391, 1395, 1399, 1403, 1408, 1412])
        ->and(JalaliCalendar::monthLength(1403, 12))->toBe(30)
        ->and(JalaliCalendar::monthLength(1404, 12))->toBe(29)
        ->and(JalaliCalendar::monthLength(1404, 6))->toBe(31)
        ->and(JalaliCalendar::monthLength(1404, 7))->toBe(30);
});

it('validates dates', function (): void {
    expect(JalaliCalendar::isValid(1403, 12, 30))->toBeTrue()
        ->and(JalaliCalendar::isValid(1404, 12, 30))->toBeFalse()
        ->and(JalaliCalendar::isValid(1404, 13, 1))->toBeFalse()
        ->and(JalaliCalendar::isValid(1404, 7, 31))->toBeFalse();
});

it('rejects invalid input', function (callable $call): void {
    $call();
})->throws(InvalidArgumentException::class)->with([
    'jalali esfand 30 in a common year' => [fn () => JalaliCalendar::toGregorian(1404, 12, 30)],
    'gregorian feb 29 in a common year' => [fn () => JalaliCalendar::toJalali(2025, 2, 29)],
    'year out of range' => [fn () => JalaliCalendar::isLeapYear(4000)],
]);

it('round-trips every day from 1300 to 1500', function (): void {
    $day = new DateTimeImmutable('1921-03-21');
    $end = new DateTimeImmutable('2121-03-21');
    $previous = null;

    while ($day < $end) {
        [$jy, $jm, $jd] = JalaliCalendar::toJalali((int) $day->format('Y'), (int) $day->format('n'), (int) $day->format('j'));
        expect(JalaliCalendar::toGregorian($jy, $jm, $jd))->toBe([(int) $day->format('Y'), (int) $day->format('n'), (int) $day->format('j')]);

        if ($previous !== null) {
            [$py, $pm, $pd] = $previous;
            $expected = $pd < JalaliCalendar::monthLength($py, $pm) ? [$py, $pm, $pd + 1] : ($pm < 12 ? [$py, $pm + 1, 1] : [$py + 1, 1, 1]);
            expect([$jy, $jm, $jd])->toBe($expected);
        }

        $previous = [$jy, $jm, $jd];
        $day = $day->modify('+1 day');
    }
});

it('agrees with ICU’s Persian calendar', function (): void {
    $formatter = new IntlDateFormatter('en_US@calendar=persian', IntlDateFormatter::NONE, IntlDateFormatter::NONE, 'UTC', IntlDateFormatter::TRADITIONAL, 'y-M-d');
    $day = new DateTimeImmutable('2010-01-01', new DateTimeZone('UTC'));

    for ($i = 0; $i < 365 * 30; $i += 3) {
        $date = $day->modify("+{$i} days");
        $icu = array_map(intval(...), explode('-', (string) $formatter->format($date)));

        expect(JalaliCalendar::toJalali((int) $date->format('Y'), (int) $date->format('n'), (int) $date->format('j')))->toBe($icu);
    }
})->skip(! class_exists(IntlDateFormatter::class), 'intl not installed');
