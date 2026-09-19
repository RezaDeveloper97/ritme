<?php

namespace App\Services\HealthEngine;

use Carbon\Carbon;
use Closure;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Log;

/**
 * Per-user cache of cycle-engine results (`/cycle/month`, `/cycle/today`,
 * `/cycle/date/{date}`), which are otherwise recomputed on every request.
 *
 * The key is content-addressed. Besides the user, `calculation_version`, the
 * locale, today's date (Tehran and app timezone: the engine reads "today" for the
 * cycle view and the age factor) and the caller's scope, it hashes the engine's
 * actual inputs: the profile row, every cycle_histories row, the daily logs in
 * the calculated range and, when tips are part of the result, the admin
 * recommendations. Those inputs are loaded anyway (and reused by the engine on a
 * miss), so a cached result can never outlive a write, whichever code path made
 * it — including writes inside the same second, or while a recalculation job is
 * still running and `calculation_version` has not moved yet. The hit saves the
 * engine's CPU, which is the cost that matters (the month view is ~1.5 ms/day).
 *
 * Cache failures (e.g. Redis down) fall back to computing the result directly.
 */
class CycleEngineCache
{
    /** Bump when the cached payload shape changes, so old entries are ignored. */
    private const SCHEMA = 1;

    /** Keys roll over with the date anyway; the TTL only bounds dead entries. */
    private const TTL_SECONDS = 86400;

    /**
     * @param  string  $scope  what is being computed, incl. its arguments (e.g. "month:2026-09:calendar")
     * @param  bool  $withContent  whether the result carries daily tips (then the recommendations are part of the key)
     * @param  Closure(): array<string, mixed>  $compute
     * @return array<string, mixed>
     */
    public function remember(
        HealthDataEngine $engine,
        string $scope,
        Carbon $from,
        Carbon $to,
        bool $withContent,
        Closure $compute,
    ): array {
        $key = $this->key($engine, $scope, $from, $to, $withContent);

        try {
            $cached = Cache::get($key);
        } catch (\Throwable $e) {
            $this->warn($e);

            return $compute();
        }

        if (is_array($cached)) {
            return $cached;
        }

        $value = $compute();

        try {
            Cache::put($key, $value, self::TTL_SECONDS);
        } catch (\Throwable $e) {
            $this->warn($e);
        }

        return $value;
    }

    public function key(HealthDataEngine $engine, string $scope, Carbon $from, Carbon $to, bool $withContent): string
    {
        $user = $engine->user();
        $profile = $user->profile;

        $inputs = [
            self::SCHEMA,
            $profile?->getAttributes(),
            $engine->cycleHistories()->map(fn (Model $row): array => $row->getAttributes())->all(),
            array_map(fn (?Model $log): ?array => $log?->getAttributes(), $engine->dailyLogsBetween($from, $to)),
            $withContent ? $engine->recommendations()->signature() : null,
        ];

        return implode(':', [
            'cycle-engine',
            $user->getKey(),
            (int) ($profile?->calculation_version ?? 0),
            $engine->locale(),
            Carbon::now('Asia/Tehran')->toDateString(),
            Carbon::today()->toDateString(),
            $scope,
            hash('xxh128', serialize($inputs)),
        ]);
    }

    private function warn(\Throwable $e): void
    {
        Log::warning('CycleEngineCache: cache unavailable, computing directly', ['error' => $e->getMessage()]);
    }
}
