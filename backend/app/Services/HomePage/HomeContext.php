<?php

namespace App\Services\HomePage;

use App\Models\DailyHealthLog;
use App\Models\PregnancyProfile;
use App\Models\User;
use App\Models\UserProfile;
use App\Services\HealthEngine\CycleMetrics;
use App\Services\HealthEngine\CycleMetricsCalculator;
use App\Services\HealthEngine\HealthDataEngine;
use App\Services\HomePage\Support\CycleHistoryDigest;
use App\Services\MessageSystem\Core\MessageManager;
use App\Services\MessageSystem\Core\MessageResult;
use App\Services\MessageSystem\Enums\MessageMode;
use Carbon\Carbon;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\Log;

/**
 * Shared, request-scoped context handed to every section.
 *
 * Immutable inputs are promoted readonly; expensive derived data
 * (cycle calculation, daily log, message result, history) is memoized so the
 * heavy HealthDataEngine runs at most once per date even though many sections
 * read from it.
 */
final class HomeContext
{
    private ?HealthDataEngine $engine = null;

    /** @var array<string, array> cycle calculation memoized per Y-m-d */
    private array $cycleDataByDate = [];

    private bool $dailyLogLoaded = false;

    private ?DailyHealthLog $dailyLog = null;

    /** @var array<int, Collection> recent logs memoized per window size */
    private array $recentLogsByWindow = [];

    /** @var array<string, Collection> logs memoized per "from..to" range */
    private array $logsByRange = [];

    /**
     * Days of daily_health_logs loaded up front, ending at the context date. Every
     * log read the sections make (the day's log, the week strip, the 3/7-day and
     * previous-week windows) falls inside it and is served from this one query
     * instead of a SELECT per day and per window.
     */
    private const LOG_WINDOW_PAST_DAYS = 29;

    /** @var Collection<int, DailyHealthLog>|null */
    private ?Collection $windowLogs = null;

    /** @var array<string, array> calendar-only calculations memoized per Y-m-d */
    private array $calendarDataByDate = [];

    private ?Collection $cycleHistories = null;

    private ?CycleHistoryDigest $cycleHistoryDigest = null;

    private ?CycleMetrics $cycleMetrics = null;

    private bool $messagesLoaded = false;

    private ?MessageResult $messages = null;

    public function __construct(
        public readonly User $user,
        public readonly ?UserProfile $profile,
        public readonly ?PregnancyProfile $pregnancyProfile,
        public readonly Carbon $date,
        public readonly string $locale,
        public readonly MessageMode $mode,
    ) {}

    public function isFa(): bool
    {
        return $this->locale === 'fa';
    }

    /**
     * Tiny inline bilingual helper.
     */
    public function t(string $fa, string $en): string
    {
        return $this->isFa() ? $fa : $en;
    }

    /**
     * A number written in the locale's own digits, for interpolation into
     * sentences the client shows verbatim — a Persian sentence carrying Latin
     * digits reads as a rendering glitch next to the rest of the app.
     */
    public function num(int|float $value): string
    {
        $text = (string) $value;

        return $this->isFa()
            ? strtr($text, ['0' => '۰', '1' => '۱', '2' => '۲', '3' => '۳', '4' => '۴', '5' => '۵', '6' => '۶', '7' => '۷', '8' => '۸', '9' => '۹'])
            : $text;
    }

    public function isCycleMode(): bool
    {
        return $this->mode === MessageMode::CYCLE;
    }

    public function isPregnancyMode(): bool
    {
        return $this->mode === MessageMode::PREGNANCY;
    }

    /**
     * Whether enough profile data exists to run cycle predictions.
     */
    public function hasCycleData(): bool
    {
        return $this->profile && $this->profile->last_period_start;
    }

    public function healthEngine(): HealthDataEngine
    {
        if ($this->engine === null) {
            $this->engine = new HealthDataEngine($this->user, $this->locale);
            [$from, $to] = $this->logWindow();
            $this->engine->preloadDailyLogs($from, $to, $this->windowLogs());
        }

        return $this->engine;
    }

    /**
     * Calendar fields only (phase, cycle day, windows — no tips/text) for a date,
     * for sections that colour days. Reuses a full calculation when one exists.
     */
    public function calendarDataFor(Carbon $date): array
    {
        $key = $date->toDateString();

        if (isset($this->cycleDataByDate[$key])) {
            return $this->cycleDataByDate[$key];
        }

        return $this->calendarDataByDate[$key] ??= $this->healthEngine()->calculateForDate($date, false);
    }

    /**
     * Cycle calculation for an arbitrary date (memoized per date).
     */
    public function cycleDataFor(Carbon $date): array
    {
        $key = $date->toDateString();

        return $this->cycleDataByDate[$key] ??= $this->healthEngine()->calculateForDate($date);
    }

    /**
     * Cycle calculation for the context date, or null if profile incomplete.
     */
    public function cycleData(): ?array
    {
        if (! $this->hasCycleData()) {
            return null;
        }

        return $this->cycleDataFor($this->date);
    }

    public function phase(): ?string
    {
        return $this->cycleData()['phase'] ?? null;
    }

    /**
     * 1-based day of the current cycle for the context date, or null when the
     * cycle isn't known — the granularity challenge content is tagged with.
     */
    public function cycleDay(): ?int
    {
        $day = $this->cycleData()['cycle_day'] ?? null;

        return $day === null ? null : (int) $day;
    }

    /**
     * Fine-grained sub-phase (CycleSubphase key) for the context date — the
     * granularity admin content is tagged with.
     */
    public function subphase(): ?string
    {
        return $this->cycleData()['subphase'] ?? null;
    }

    public function dailyLog(): ?DailyHealthLog
    {
        if (! $this->dailyLogLoaded) {
            $this->dailyLog = $this->healthEngine()->dailyLogFor($this->date);
            $this->dailyLogLoaded = true;
        }

        return $this->dailyLog;
    }

    /**
     * Logs for the last $days days (inclusive of the context date), ordered ascending.
     *
     * @return Collection<int, DailyHealthLog>
     */
    public function recentLogs(int $days = 7): Collection
    {
        return $this->recentLogsByWindow[$days] ??= $this->logsBetween(
            $this->date->copy()->subDays($days - 1),
            $this->date
        );
    }

    /**
     * Logs in an arbitrary inclusive date window, ordered ascending. Used by
     * sections that compare a window against the one before it.
     *
     * @return Collection<int, DailyHealthLog>
     */
    public function logsBetween(Carbon $from, Carbon $to): Collection
    {
        $fromKey = $from->toDateString();
        $toKey = $to->toDateString();
        $key = $fromKey.'..'.$toKey;

        if (isset($this->logsByRange[$key])) {
            return $this->logsByRange[$key];
        }

        [$windowFrom, $windowTo] = $this->logWindow();
        if ($fromKey >= $windowFrom->toDateString() && $toKey <= $windowTo->toDateString()) {
            return $this->logsByRange[$key] = $this->windowLogs()
                ->filter(function (DailyHealthLog $log) use ($fromKey, $toKey): bool {
                    $day = Carbon::parse($log->log_date)->toDateString();

                    return $day >= $fromKey && $day <= $toKey;
                })
                ->values();
        }

        return $this->logsByRange[$key] = $this->user->dailyHealthLogs()
            ->whereDate('log_date', '>=', $from)
            ->whereDate('log_date', '<=', $to)
            ->orderBy('log_date')
            ->get();
    }

    /**
     * The preloaded range: LOG_WINDOW_PAST_DAYS before the context date through
     * the end of its Saturday-to-Friday week (the week strip can show later days).
     *
     * @return array{0: Carbon, 1: Carbon}
     */
    private function logWindow(): array
    {
        $from = $this->date->copy()->startOfDay()->subDays(self::LOG_WINDOW_PAST_DAYS);
        $to = $this->date->copy()->startOfDay()->startOfWeek(Carbon::SATURDAY)->addDays(6);

        return [$from, $to->max($this->date->copy()->startOfDay())];
    }

    /**
     * Every log in {@see logWindow()}, ascending — one query per request.
     *
     * @return Collection<int, DailyHealthLog>
     */
    private function windowLogs(): Collection
    {
        if ($this->windowLogs === null) {
            [$from, $to] = $this->logWindow();
            $this->windowLogs = $this->user->dailyHealthLogs()
                ->whereBetween('log_date', [$from->toDateString(), $to->toDateString().' 23:59:59'])
                ->orderBy('log_date')
                ->get();
        }

        return $this->windowLogs;
    }

    /**
     * Confirmed/recorded period history, newest first.
     *
     * @return Collection<int, \App\Models\CycleHistory>
     */
    public function cycleHistories(): Collection
    {
        // The engine's load (same rows, same order), so the page reads the table once.
        return $this->cycleHistories ??= $this->healthEngine()->cycleHistories();
    }

    /**
     * Per-cycle facts (real length and bleed duration of each recorded cycle)
     * derived from that same history. Memoized — several sections read it.
     */
    public function cycleHistoryDigest(): CycleHistoryDigest
    {
        return $this->cycleHistoryDigest ??= CycleHistoryDigest::fromHistories($this->cycleHistories());
    }

    /**
     * The engine's three-layer cycle metrics (effective length/duration,
     * regularity, variability spread) for this user. Sections use it so the
     * numbers they show are the ones predictions were actually built from.
     */
    public function cycleMetrics(): CycleMetrics
    {
        return $this->cycleMetrics ??= (new CycleMetricsCalculator)
            ->calculate($this->cycleHistories(), $this->profile);
    }

    /**
     * Unified MessageSystem result (Layer 1-4 + supplements), or null if it
     * cannot be produced. Failures are swallowed so message issues never break
     * the whole page.
     */
    public function messages(): ?MessageResult
    {
        if ($this->messagesLoaded) {
            return $this->messages;
        }

        $this->messagesLoaded = true;

        try {
            if ($this->isCycleMode() && ! $this->hasCycleData()) {
                return $this->messages = null;
            }

            $manager = new MessageManager($this->user, $this->locale, $this->healthEngine());
            $this->messages = $manager->generateMessages($this->date, $this->mode);
        } catch (\Throwable $e) {
            Log::warning('HomeContext: message generation failed', [
                'user_id' => $this->user->id,
                'error' => $e->getMessage(),
            ]);
            $this->messages = null;
        }

        return $this->messages;
    }

    /**
     * The start date (day 1) of the current cycle, or null if unknown.
     */
    public function currentCycleStart(): ?Carbon
    {
        $cycleDay = $this->cycleData()['cycle_day'] ?? null;

        if ($cycleDay === null) {
            return null;
        }

        return $this->date->copy()->subDays($cycleDay - 1);
    }
}
