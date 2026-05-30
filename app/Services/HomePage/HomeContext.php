<?php

namespace App\Services\HomePage;

use App\Models\DailyHealthLog;
use App\Models\PregnancyProfile;
use App\Models\User;
use App\Models\UserProfile;
use App\Services\HealthEngine\HealthDataEngine;
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

    private ?Collection $cycleHistories = null;

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
        return $this->engine ??= new HealthDataEngine($this->user, $this->locale);
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
        if (!$this->hasCycleData()) {
            return null;
        }

        return $this->cycleDataFor($this->date);
    }

    public function phase(): ?string
    {
        return $this->cycleData()['phase'] ?? null;
    }

    public function dailyLog(): ?DailyHealthLog
    {
        if (!$this->dailyLogLoaded) {
            $this->dailyLog = $this->user->dailyHealthLogs()
                ->whereDate('log_date', $this->date)
                ->first();
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
        return $this->recentLogsByWindow[$days] ??= $this->user->dailyHealthLogs()
            ->whereDate('log_date', '>=', $this->date->copy()->subDays($days - 1))
            ->whereDate('log_date', '<=', $this->date)
            ->orderBy('log_date')
            ->get();
    }

    /**
     * Confirmed/recorded period history, newest first.
     *
     * @return Collection<int, \App\Models\CycleHistory>
     */
    public function cycleHistories(): Collection
    {
        return $this->cycleHistories ??= $this->user->cycleHistories()
            ->orderByDesc('period_start_date')
            ->get();
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
            if ($this->isCycleMode() && !$this->hasCycleData()) {
                return $this->messages = null;
            }

            $manager = new MessageManager($this->user, $this->locale);
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
