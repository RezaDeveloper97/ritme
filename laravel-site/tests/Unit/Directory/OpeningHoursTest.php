<?php

declare(strict_types=1);

use App\Domain\Directory\Enums\Weekday;
use App\Domain\Directory\Support\OpeningHours;
use App\Domain\Seo\Schema\Data\OpeningHoursData;
use App\Domain\Seo\Schema\Enums\DayOfWeek;
use Carbon\CarbonImmutable;

beforeEach(function (): void {
    $this->previousTimezone = date_default_timezone_get();
    date_default_timezone_set('Asia/Tehran');
});

afterEach(function (): void {
    date_default_timezone_set($this->previousTimezone);
});

function abPariHours(): OpeningHours
{
    $day = [['opens' => '09:00', 'closes' => '20:00']];

    return OpeningHours::fromArray([
        'saturday' => $day, 'sunday' => $day, 'monday' => $day, 'tuesday' => $day, 'wednesday' => $day,
        'thursday' => [['opens' => '09:00', 'closes' => '14:00']],
        'friday' => [],
    ]);
}

// 2026-10-03 is a Saturday, 2026-10-08 a Thursday, 2026-10-09 a Friday.
function tehran(string $at): CarbonImmutable
{
    return CarbonImmutable::parse($at, 'Asia/Tehran');
}

it('names the Iranian weekdays in Persian, Saturday first', function (): void {
    expect(array_map(static fn (Weekday $d): string => $d->label(), Weekday::cases()))
        ->toBe(['شنبه', 'یکشنبه', 'دوشنبه', 'سه‌شنبه', 'چهارشنبه', 'پنجشنبه', 'جمعه'])
        ->and(Weekday::fromDate(tehran('2026-10-03 12:00')))->toBe(Weekday::Saturday)
        ->and(Weekday::fromDate(tehran('2026-10-09 12:00')))->toBe(Weekday::Friday)
        ->and(Weekday::Saturday->previous())->toBe(Weekday::Friday)
        ->and(Weekday::Friday->next())->toBe(Weekday::Saturday)
        ->and(Weekday::Thursday->schemaDay())->toBe(DayOfWeek::Thursday);
});

it('knows whether a place is open now', function (string $at, bool $open, ?string $closes): void {
    $hours = abPariHours();

    expect($hours->isOpenAt(tehran($at)))->toBe($open)
        ->and($hours->closesAt(tehran($at)))->toBe($closes);
})->with([
    'saturday morning before opening' => ['2026-10-03 08:59', false, null],
    'saturday at opening' => ['2026-10-03 09:00', true, '20:00'],
    'saturday evening' => ['2026-10-03 19:59', true, '20:00'],
    'saturday at closing' => ['2026-10-03 20:00', false, null],
    'thursday noon' => ['2026-10-08 12:00', true, '14:00'],
    'thursday afternoon' => ['2026-10-08 15:00', false, null],
    'friday closed' => ['2026-10-09 12:00', false, null],
]);

it('evaluates in Tehran time whatever the timezone of the given moment', function (): void {
    // 05:30 UTC = 09:00 Tehran (UTC+3:30, no DST since 2022).
    expect(abPariHours()->isOpenAt(CarbonImmutable::parse('2026-10-03 05:30', 'UTC')))->toBeTrue()
        ->and(abPariHours()->isOpenAt(CarbonImmutable::parse('2026-10-03 05:29', 'UTC')))->toBeFalse();
});

it('handles ranges past midnight and split days', function (): void {
    $hours = OpeningHours::fromArray([
        'saturday' => [['opens' => '18:00', 'closes' => '02:00']],
        'sunday' => [['opens' => '09:00', 'closes' => '12:00'], ['opens' => '16:00', 'closes' => '20:00']],
        'monday' => [['00:00', '24:00']],
    ]);

    expect($hours->isOpenAt(tehran('2026-10-03 23:00')))->toBeTrue()          // saturday night
        ->and($hours->closesAt(tehran('2026-10-04 01:30')))->toBe('02:00')     // early sunday, still saturday's range
        ->and($hours->isOpenAt(tehran('2026-10-04 02:00')))->toBeFalse()
        ->and($hours->isOpenAt(tehran('2026-10-04 13:00')))->toBeFalse()       // lunch break
        ->and($hours->closesAt(tehran('2026-10-04 16:30')))->toBe('20:00')
        ->and($hours->isOpenAt(tehran('2026-10-05 23:59')))->toBeTrue()        // monday all day
        ->and($hours->isOpenAt(tehran('2026-10-06 00:30')))->toBeFalse();      // tuesday unknown
});

it('finds the next opening time', function (): void {
    $hours = abPariHours();

    expect($hours->nextOpening(tehran('2026-10-03 07:00')))->toBe(['day' => Weekday::Saturday, 'opens' => '09:00', 'daysAhead' => 0])
        ->and($hours->nextOpening(tehran('2026-10-08 15:00')))->toBe(['day' => Weekday::Saturday, 'opens' => '09:00', 'daysAhead' => 2])
        ->and(OpeningHours::fromArray([])->nextOpening(tehran('2026-10-03 07:00')))->toBeNull();
});

it('builds the weekly table with Persian names and digits, flagging today', function (): void {
    $rows = abPariHours()->rows(tehran('2026-10-08 10:00'));

    expect(array_column($rows, 'label'))->toBe(['شنبه', 'یکشنبه', 'دوشنبه', 'سه‌شنبه', 'چهارشنبه', 'پنجشنبه', 'جمعه'])
        ->and($rows[0]['hours'])->toBe('۹ تا ۲۰')
        ->and($rows[5]['hours'])->toBe('۹ تا ۱۴')
        ->and($rows[5]['today'])->toBeTrue()
        ->and($rows[0]['today'])->toBeFalse()
        ->and($rows[6]['hours'])->toBe('تعطیل')
        ->and($rows[6]['closed'])->toBeTrue();

    $split = OpeningHours::fromArray(['sunday' => [['16:00', '20:30'], ['09:00', '12:00']]])->rows();
    expect($split[1]['hours'])->toBe('۹ تا ۱۲، ۱۶ تا ۲۰:۳۰')
        ->and($split[0]['hours'])->toBeNull()   // unknown day
        ->and($split[0]['closed'])->toBeFalse()
        ->and(OpeningHours::time('23:59'))->toBe('۲۴');
});

it('normalises input and drops invalid ranges', function (): void {
    $hours = OpeningHours::fromArray([
        'saturday' => [['opens' => '9:00', 'closes' => '۲۰:۰۰'], ['opens' => '25:00', 'closes' => '26:00'], ['10:00', '10:00'], 'junk'],
        'holiday' => [['09:00', '10:00']],
        'friday' => [],
    ]);

    expect($hours->toArray())->toBe([
        'saturday' => [['opens' => '09:00', 'closes' => '20:00']],
        'friday' => [],
    ])
        ->and($hours->isKnown(Weekday::Friday))->toBeTrue()
        ->and($hours->isKnown(Weekday::Monday))->toBeFalse()
        ->and(OpeningHours::fromArray(null)->isEmpty())->toBeTrue()
        ->and(OpeningHours::fromArray(['friday' => []])->isEmpty())->toBeTrue()
        ->and($hours->isEmpty())->toBeFalse();
});

it('groups identical days into schema.org OpeningHoursSpecification', function (): void {
    $specs = abPariHours()->toSchema();

    expect($specs)->toHaveCount(2)
        ->and($specs[0])->toBeInstanceOf(OpeningHoursData::class)
        ->and($specs[0]->days)->toBe([DayOfWeek::Saturday, DayOfWeek::Sunday, DayOfWeek::Monday, DayOfWeek::Tuesday, DayOfWeek::Wednesday])
        ->and([$specs[0]->opens, $specs[0]->closes])->toBe(['09:00', '20:00'])
        ->and($specs[1]->days)->toBe([DayOfWeek::Thursday])
        ->and([$specs[1]->opens, $specs[1]->closes])->toBe(['09:00', '14:00']);
});
