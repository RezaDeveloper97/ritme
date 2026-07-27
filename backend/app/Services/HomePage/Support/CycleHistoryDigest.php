<?php

namespace App\Services\HomePage\Support;

use App\Models\CycleHistory;
use Carbon\Carbon;
use Illuminate\Support\Collection;

/**
 * Turns a user's raw period history into the per-cycle facts the "my cycles"
 * and "cycle summary" widgets show: how long each cycle actually ran and how
 * many days each bleed lasted.
 *
 * Both numbers are derived from the logged timeline rather than read from the
 * stored `cycle_length` / `bleeding_length` columns, which are only populated
 * for some rows — a cycle's length is the gap to the *next* period start, so it
 * is knowable only once that next period exists. That mirrors how
 * {@see \App\Services\HealthEngine\CycleMetricsCalculator} derives its medians,
 * keeping the summary consistent with the predictions built from the same data.
 */
final class CycleHistoryDigest
{
    /** A gap outside this range is a logging artefact, not a real cycle (§7). */
    private const VALID_CYCLE_MIN = 21;

    private const VALID_CYCLE_MAX = 45;

    private const VALID_DURATION_MIN = 2;

    private const VALID_DURATION_MAX = 10;

    /**
     * @param  array<int, array<string, mixed>>  $cycles  newest first
     */
    private function __construct(private readonly array $cycles) {}

    /**
     * @param  Collection<int, CycleHistory>  $histories  any order
     */
    public static function fromHistories(Collection $histories): self
    {
        // Oldest→newest first, so each entry can see the start that closes it.
        $ordered = $histories
            ->filter(fn (CycleHistory $h) => $h->period_start_date !== null)
            ->sortBy(fn (CycleHistory $h) => $h->period_start_date->timestamp)
            ->values();

        $cycles = [];

        foreach ($ordered as $index => $history) {
            $start = $history->period_start_date->copy()->startOfDay();
            $end = $history->period_end_date?->copy()->startOfDay();
            $nextStart = $ordered->get($index + 1)?->period_start_date?->copy()->startOfDay();

            // This cycle's length is start → next start. The newest cycle is still
            // running, so it has none until the following period is logged.
            $cycleLength = $nextStart ? (int) $start->diffInDays($nextStart) : null;
            $periodLength = ($end && $end->gte($start)) ? ((int) $start->diffInDays($end)) + 1 : null;

            $cycles[] = [
                'id' => $history->id,
                'period_start_date' => $start->toDateString(),
                'period_end_date' => $end?->toDateString(),
                'cycle_length' => $cycleLength,
                'period_length' => $periodLength,
                'is_ongoing' => $end === null,
                'is_current' => $nextStart === null,
                'is_confirmed' => (bool) $history->is_confirmed,
                'is_estimated' => (bool) $history->is_estimated,
                'source' => $history->source,
            ];
        }

        return new self(array_reverse($cycles));
    }

    /**
     * Every recorded cycle, newest first.
     *
     * @return array<int, array<string, mixed>>
     */
    public function all(): array
    {
        return $this->cycles;
    }

    public function count(): int
    {
        return count($this->cycles);
    }

    /**
     * The cycle the user is in right now (the most recent period start).
     *
     * @return array<string, mixed>|null
     */
    public function current(): ?array
    {
        return $this->cycles[0] ?? null;
    }

    /**
     * The cycle that started on a given day, when it is on record — used to
     * attach the logged bleed to the engine's notion of the current cycle.
     *
     * @return array<string, mixed>|null
     */
    public function startingOn(Carbon $date): ?array
    {
        $iso = $date->toDateString();

        foreach ($this->cycles as $cycle) {
            if ($cycle['period_start_date'] === $iso) {
                return $cycle;
            }
        }

        return null;
    }

    /**
     * Completed cycles behind the current one, newest first.
     *
     * @return array<int, array<string, mixed>>
     */
    public function previous(?int $limit = null): array
    {
        $previous = array_slice($this->cycles, 1);

        return $limit === null ? $previous : array_slice($previous, 0, $limit);
    }

    /**
     * Length of the cycle that just ended — the gap between the two most recent
     * period starts. Null until a second period has been logged.
     */
    public function lastCycleLength(): ?int
    {
        foreach ($this->cycles as $cycle) {
            if ($cycle['cycle_length'] !== null) {
                return $cycle['cycle_length'];
            }
        }

        return null;
    }

    /**
     * Duration of the most recently *finished* bleed. An ongoing period is
     * skipped — its length isn't known yet, and reporting a partial count would
     * read as an abnormally short period.
     */
    public function lastPeriodLength(): ?int
    {
        foreach ($this->cycles as $cycle) {
            if ($cycle['period_length'] !== null) {
                return $cycle['period_length'];
            }
        }

        return null;
    }

    /**
     * @return array<int, int> valid cycle lengths, newest first
     */
    public function validCycleLengths(): array
    {
        return $this->collectValid('cycle_length', self::VALID_CYCLE_MIN, self::VALID_CYCLE_MAX);
    }

    /**
     * @return array<int, int> valid period durations, newest first
     */
    public function validPeriodLengths(): array
    {
        return $this->collectValid('period_length', self::VALID_DURATION_MIN, self::VALID_DURATION_MAX);
    }

    public function averageCycleLength(): ?int
    {
        return $this->average($this->validCycleLengths());
    }

    public function averagePeriodLength(): ?int
    {
        return $this->average($this->validPeriodLengths());
    }

    public function shortestCycle(): ?int
    {
        $lengths = $this->validCycleLengths();

        return $lengths === [] ? null : min($lengths);
    }

    public function longestCycle(): ?int
    {
        $lengths = $this->validCycleLengths();

        return $lengths === [] ? null : max($lengths);
    }

    /**
     * @return array<int, int>
     */
    private function collectValid(string $field, int $min, int $max): array
    {
        $values = [];

        foreach ($this->cycles as $cycle) {
            $value = $cycle[$field];
            if ($value !== null && $value >= $min && $value <= $max) {
                $values[] = $value;
            }
        }

        return $values;
    }

    /**
     * @param  array<int, int>  $values
     */
    private function average(array $values): ?int
    {
        return $values === [] ? null : (int) round(array_sum($values) / count($values));
    }
}
