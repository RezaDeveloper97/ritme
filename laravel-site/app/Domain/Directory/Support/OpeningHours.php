<?php

declare(strict_types=1);

namespace App\Domain\Directory\Support;

use App\Domain\Directory\Enums\Weekday;
use App\Domain\Seo\Schema\Data\OpeningHoursData;
use App\Support\Text\PersianDigits;
use Carbon\CarbonImmutable;
use Carbon\CarbonInterface;

/**
 * Weekly opening hours of a place (the `opening_hours` JSON), evaluated in the app timezone (Asia/Tehran):
 *
 *   ['saturday' => [['opens' => '09:00', 'closes' => '20:00']], 'friday' => []]
 *
 * A day with `[]` is closed; a missing day is unknown. A range whose `closes` is earlier than `opens` runs past
 * midnight (it is open early the next day). `23:59` as closing time means "until the end of the day". Input ranges
 * may also be `['09:00', '20:00']` lists and `9:00`-style times; invalid or empty ranges are dropped.
 */
final readonly class OpeningHours
{
    private const END_OF_DAY = '23:59';

    /**
     * @param  array<string, list<array{opens: string, closes: string}>>  $days  keyed by Weekday value, week order
     */
    private function __construct(private array $days) {}

    public static function fromArray(mixed $value): self
    {
        $days = [];
        if (! is_array($value)) {
            return new self($days);
        }

        foreach (Weekday::cases() as $day) {
            if (! array_key_exists($day->value, $value)) {
                continue;
            }

            $ranges = [];
            foreach ((array) $value[$day->value] as $range) {
                $normalized = self::range($range);
                if ($normalized !== null) {
                    $ranges[] = $normalized;
                }
            }
            usort($ranges, static fn (array $a, array $b): int => strcmp($a['opens'], $b['opens']));
            $days[$day->value] = $ranges;
        }

        return new self($days);
    }

    /**
     * @return array<string, list<array{opens: string, closes: string}>>
     */
    public function toArray(): array
    {
        return $this->days;
    }

    /** True when no day has any opening range (unknown hours or closed all week). */
    public function isEmpty(): bool
    {
        foreach ($this->days as $ranges) {
            if ($ranges !== []) {
                return false;
            }
        }

        return true;
    }

    public function isKnown(Weekday $day): bool
    {
        return array_key_exists($day->value, $this->days);
    }

    /**
     * @return list<array{opens: string, closes: string}>
     */
    public function ranges(Weekday $day): array
    {
        return $this->days[$day->value] ?? [];
    }

    public function isOpenAt(CarbonInterface $at): bool
    {
        return $this->closesAt($at) !== null;
    }

    /**
     * Closing time (HH:MM) of the range that is open at $at, or null when closed.
     */
    public function closesAt(CarbonInterface $at): ?string
    {
        $local = self::local($at);
        $day = Weekday::fromDate($local);
        $time = $local->format('H:i');

        foreach ($this->ranges($day) as $range) {
            if (self::overnight($range)) {
                if ($time >= $range['opens']) {
                    return $range['closes'];
                }
            } elseif ($time >= $range['opens'] && ($time < $range['closes'] || $range['closes'] === self::END_OF_DAY)) {
                return $range['closes'];
            }
        }

        foreach ($this->ranges($day->previous()) as $range) {
            if (self::overnight($range) && $time < $range['closes']) {
                return $range['closes'];
            }
        }

        return null;
    }

    /**
     * The next time the place opens after $at (today later on, or one of the next seven days), or null when no hours
     * are known. `daysAhead` 0 = today, 1 = tomorrow…
     *
     * @return array{day: Weekday, opens: string, daysAhead: int}|null
     */
    public function nextOpening(CarbonInterface $at): ?array
    {
        $local = self::local($at);
        $day = Weekday::fromDate($local);
        $time = $local->format('H:i');

        for ($ahead = 0; $ahead <= 7; $ahead++) {
            foreach ($this->ranges($day) as $range) {
                if ($ahead > 0 || $range['opens'] > $time) {
                    return ['day' => $day, 'opens' => $range['opens'], 'daysAhead' => $ahead];
                }
            }
            $day = $day->next();
        }

        return null;
    }

    /**
     * The weekly table for the place page: Persian weekday names, hours like «۹ تا ۲۰» (several ranges joined with
     * «، »), «تعطیل» for closed days, null hours for unknown days, and today's row flagged.
     *
     * @return list<array{day: Weekday, label: string, hours: string|null, closed: bool, today: bool}>
     */
    public function rows(?CarbonInterface $today = null): array
    {
        $current = $today === null ? null : Weekday::fromDate(self::local($today));
        $rows = [];

        foreach (Weekday::cases() as $day) {
            $ranges = $this->ranges($day);
            $closed = $this->isKnown($day) && $ranges === [];

            $rows[] = [
                'day' => $day,
                'label' => $day->label(),
                'hours' => $closed ? 'تعطیل' : ($ranges === [] ? null : implode('، ', array_map(
                    static fn (array $range): string => self::time($range['opens']).' تا '.self::time($range['closes']),
                    $ranges,
                ))),
                'closed' => $closed,
                'today' => $current === $day,
            ];
        }

        return $rows;
    }

    /**
     * schema.org OpeningHoursSpecification entries: days with identical hours are grouped, closed and unknown days
     * are omitted.
     *
     * @return list<OpeningHoursData>
     */
    public function toSchema(): array
    {
        /** @var array<string, array{ranges: list<array{opens: string, closes: string}>, days: list<Weekday>}> $groups */
        $groups = [];
        foreach (Weekday::cases() as $day) {
            $ranges = $this->ranges($day);
            if ($ranges === []) {
                continue;
            }
            $signature = json_encode($ranges, JSON_THROW_ON_ERROR);
            $groups[$signature] ??= ['ranges' => $ranges, 'days' => []];
            $groups[$signature]['days'][] = $day;
        }

        $specs = [];
        foreach ($groups as $group) {
            $days = array_map(static fn (Weekday $day) => $day->schemaDay(), $group['days']);
            foreach ($group['ranges'] as $range) {
                $specs[] = new OpeningHoursData($days, $range['opens'], $range['closes']);
            }
        }

        return $specs;
    }

    /**
     * «۹», «۹:۳۰», «۲۴» (end of day) in Persian digits.
     */
    public static function time(string $time): string
    {
        if ($time === self::END_OF_DAY) {
            return PersianDigits::toPersian(24);
        }

        [$hour, $minute] = explode(':', $time);

        return PersianDigits::toPersian($minute === '00' ? (string) (int) $hour : ((int) $hour).':'.$minute);
    }

    /**
     * @return array{opens: string, closes: string}|null
     */
    private static function range(mixed $range): ?array
    {
        if (! is_array($range)) {
            return null;
        }

        $opens = self::clock($range['opens'] ?? $range[0] ?? null);
        $closes = self::clock($range['closes'] ?? $range[1] ?? null);

        return $opens === null || $closes === null || $opens === $closes ? null : ['opens' => $opens, 'closes' => $closes];
    }

    private static function clock(mixed $value): ?string
    {
        if (! is_string($value)) {
            return null;
        }

        $value = PersianDigits::toLatin(trim($value));
        if ($value === '24:00') {
            return self::END_OF_DAY;
        }
        if (preg_match('/^(\d{1,2}):(\d{2})(?::\d{2})?$/', $value, $m) !== 1 || (int) $m[1] > 23 || (int) $m[2] > 59) {
            return null;
        }

        return sprintf('%02d:%02d', (int) $m[1], (int) $m[2]);
    }

    /**
     * @param  array{opens: string, closes: string}  $range
     */
    private static function overnight(array $range): bool
    {
        return $range['closes'] < $range['opens'];
    }

    private static function local(CarbonInterface $at): CarbonImmutable
    {
        return CarbonImmutable::instance($at)->setTimezone(date_default_timezone_get());
    }
}
