<?php

namespace App\Services\Challenges;

use App\Models\Challenge;
use App\Models\DailyHealthLog;
use App\Models\User;
use App\Models\UserChallengeCompletion;
use Carbon\Carbon;
use Illuminate\Support\Collection;

/**
 * Picks "چالش امروز" for a user and answers everything the UI needs about it
 * (completion state, streak, weekly progress).
 *
 * Selection is layered, strongest signal first:
 *
 *  1. **Cycle phase** — challenges tagged for another phase are excluded.
 *  2. **Recent-log signal** — poor sleep, pain, low energy or a logging gap
 *     steer the pick toward a matching category when the pool has one.
 *  3. **Difficulty ladder** — the pool is capped by the user's current streak
 *     so a returning user is not handed a "hard" challenge on day one.
 *  4. **No repeats** — anything the user completed in the previous
 *     {@see self::REPEAT_WINDOW_DAYS} days drops out of the pool.
 *  5. **Deterministic per-user pick** — a hash of (user, date) selects from
 *     what's left, so the challenge is stable all day but differs per user.
 *
 * Every filter is *soft*: if it would empty the pool it is skipped, so a user
 * always gets a challenge as long as one active challenge exists.
 *
 * Note on stability: the no-repeat window deliberately ends *yesterday*, so
 * completing today's challenge cannot change today's pick.
 */
final class DailyChallengeService
{
    /** Days a completed challenge stays out of the pool. */
    private const REPEAT_WINDOW_DAYS = 14;

    /** Log window inspected for the category signal. */
    private const SIGNAL_WINDOW_DAYS = 3;

    /** Streak needed before a difficulty tier is offered. */
    private const DIFFICULTY_UNLOCK = [
        'easy' => 0,
        'medium' => 3,
        'hard' => 7,
    ];

    /**
     * The full "challenge" payload for a day, or null when no challenge is
     * available (empty/inactive pool).
     *
     * @param  Collection<int, DailyHealthLog>|null  $recentLogs  pre-loaded logs
     *                                                            (HomeContext memoizes them); fetched on demand when omitted.
     * @return array<string, mixed>|null
     */
    public function payload(
        User $user,
        Carbon $date,
        string $locale,
        ?string $phase = null,
        ?Collection $recentLogs = null,
    ): ?array {
        $challenge = $this->challengeFor($user, $date, $phase, $recentLogs);

        if (! $challenge) {
            return null;
        }

        $isCompleted = $this->isCompleted($user, $challenge, $date);
        $streak = $this->currentStreak($user, $date);

        return [
            'id' => $challenge->id,
            'title' => $challenge->localized('title', $locale),
            'description' => $challenge->localized('description', $locale),
            'category' => $challenge->category,
            'difficulty' => $challenge->difficulty,
            'is_completed' => $isCompleted,
            'streak' => $streak,
            'longest_streak' => $this->longestStreak($user),
            'week_days' => $this->weekDays($user, $date),
            'status_message' => $this->statusMessage($challenge, $isCompleted, $streak, $locale),
        ];
    }

    /**
     * Flip today's completion and return the new state.
     */
    public function toggle(User $user, Challenge $challenge, Carbon $date): bool
    {
        $existing = UserChallengeCompletion::query()
            ->where('user_id', $user->id)
            ->where('challenge_id', $challenge->id)
            ->whereDate('completion_date', $date)
            ->first();

        if ($existing) {
            $existing->delete();

            return false;
        }

        UserChallengeCompletion::create([
            'user_id' => $user->id,
            'challenge_id' => $challenge->id,
            'completion_date' => $date->toDateString(),
            'completed_at' => now(),
        ]);

        return true;
    }

    /**
     * The challenge this user should see on this date.
     */
    public function challengeFor(
        User $user,
        Carbon $date,
        ?string $phase = null,
        ?Collection $recentLogs = null,
    ): ?Challenge {
        $pool = Challenge::query()
            ->active()
            ->forPhase($phase)
            ->orderBy('sort_order')
            ->orderBy('id')
            ->get();

        if ($pool->isEmpty()) {
            return null;
        }

        $pool = $this->withoutRecentlyCompleted($pool, $user, $date);
        $pool = $this->cappedByDifficulty($pool, $this->currentStreak($user, $date));
        $pool = $this->preferSignalCategory($pool, $user, $date, $recentLogs);

        return $pool[$this->seed($user, $date) % $pool->count()];
    }

    public function isCompleted(User $user, Challenge $challenge, Carbon $date): bool
    {
        return UserChallengeCompletion::query()
            ->where('user_id', $user->id)
            ->where('challenge_id', $challenge->id)
            ->whereDate('completion_date', $date)
            ->exists();
    }

    /**
     * Consecutive days (ending today, or yesterday when today isn't done yet)
     * on which the user completed at least one challenge.
     */
    public function currentStreak(User $user, Carbon $date): int
    {
        $days = $this->completedDates($user, $date->copy()->subDays(365), $date);

        if ($days->isEmpty()) {
            return 0;
        }

        $cursor = $date->copy();

        // Today not done yet doesn't break a streak that ran through yesterday.
        if (! $days->contains($cursor->toDateString())) {
            $cursor->subDay();
        }

        $streak = 0;

        while ($days->contains($cursor->toDateString())) {
            $streak++;
            $cursor->subDay();
        }

        return $streak;
    }

    /**
     * Longest run of consecutive completion days the user has ever had.
     */
    public function longestStreak(User $user): int
    {
        $days = $this->completedDates($user)->sort()->values();

        $longest = 0;
        $run = 0;
        $previous = null;

        foreach ($days as $day) {
            $current = Carbon::parse($day);
            $run = ($previous && $previous->copy()->addDay()->isSameDay($current)) ? $run + 1 : 1;
            $longest = max($longest, $run);
            $previous = $current;
        }

        return $longest;
    }

    /**
     * The last 7 days (oldest → newest) as `{date, is_completed, is_today}` —
     * the strip of dots under the challenge card.
     *
     * @return array<int, array<string, mixed>>
     */
    public function weekDays(User $user, Carbon $date): array
    {
        $from = $date->copy()->subDays(6);
        $days = $this->completedDates($user, $from, $date);

        $out = [];

        for ($cursor = $from->copy(); $cursor <= $date; $cursor->addDay()) {
            $out[] = [
                'date' => $cursor->toDateString(),
                'is_completed' => $days->contains($cursor->toDateString()),
                'is_today' => $cursor->isSameDay($date),
            ];
        }

        return $out;
    }

    /**
     * Distinct `Y-m-d` strings on which the user completed something.
     *
     * @return Collection<int, string>
     */
    private function completedDates(User $user, ?Carbon $from = null, ?Carbon $to = null): Collection
    {
        return UserChallengeCompletion::query()
            ->where('user_id', $user->id)
            ->when($from, fn ($q) => $q->whereDate('completion_date', '>=', $from))
            ->when($to, fn ($q) => $q->whereDate('completion_date', '<=', $to))
            ->pluck('completion_date')
            ->map(fn ($d) => Carbon::parse($d)->toDateString())
            ->unique()
            ->values();
    }

    /**
     * Drop challenges completed in the previous {@see self::REPEAT_WINDOW_DAYS}
     * days (today excluded — see the class note on pick stability).
     *
     * @param  Collection<int, Challenge>  $pool
     * @return Collection<int, Challenge>
     */
    private function withoutRecentlyCompleted(Collection $pool, User $user, Carbon $date): Collection
    {
        $recent = UserChallengeCompletion::query()
            ->where('user_id', $user->id)
            ->whereDate('completion_date', '>=', $date->copy()->subDays(self::REPEAT_WINDOW_DAYS))
            ->whereDate('completion_date', '<', $date)
            ->pluck('challenge_id')
            ->unique();

        $filtered = $pool->reject(fn (Challenge $c) => $recent->contains($c->id))->values();

        return $filtered->isEmpty() ? $pool : $filtered;
    }

    /**
     * Keep only difficulties unlocked at the user's current streak.
     *
     * @param  Collection<int, Challenge>  $pool
     * @return Collection<int, Challenge>
     */
    private function cappedByDifficulty(Collection $pool, int $streak): Collection
    {
        $allowed = array_keys(array_filter(
            self::DIFFICULTY_UNLOCK,
            fn (int $needed) => $streak >= $needed,
        ));

        $filtered = $pool
            ->filter(fn (Challenge $c) => $c->difficulty === null || in_array($c->difficulty, $allowed, true))
            ->values();

        return $filtered->isEmpty() ? $pool : $filtered;
    }

    /**
     * Narrow the pool to the category the user's recent logs point at.
     *
     * @param  Collection<int, Challenge>  $pool
     * @param  Collection<int, DailyHealthLog>|null  $recentLogs
     * @return Collection<int, Challenge>
     */
    private function preferSignalCategory(
        Collection $pool,
        User $user,
        Carbon $date,
        ?Collection $recentLogs,
    ): Collection {
        $category = $this->signalCategory($user, $date, $recentLogs);

        if ($category === null) {
            return $pool;
        }

        $filtered = $pool->where('category', $category)->values();

        return $filtered->isEmpty() ? $pool : $filtered;
    }

    /**
     * Read the last few daily logs and name the category that would help most,
     * or null when nothing stands out.
     *
     * @param  Collection<int, DailyHealthLog>|null  $recentLogs
     */
    private function signalCategory(User $user, Carbon $date, ?Collection $recentLogs): ?string
    {
        $logs = $recentLogs
            ?? $user->dailyHealthLogs()
                ->whereDate('log_date', '>=', $date->copy()->subDays(self::SIGNAL_WINDOW_DAYS - 1))
                ->whereDate('log_date', '<=', $date)
                ->get();

        $logs = $logs->filter(
            fn (DailyHealthLog $log) => Carbon::parse($log->log_date)
                ->gte($date->copy()->subDays(self::SIGNAL_WINDOW_DAYS - 1)),
        );

        // Nothing logged lately — nudge the habit itself.
        if ($logs->isEmpty()) {
            return 'tracking';
        }

        if ($logs->contains(fn (DailyHealthLog $log) => $this->hasPoorSleep($log))) {
            return 'sleep';
        }

        if ($logs->contains(fn (DailyHealthLog $log) => $this->hasPainOrLowMood($log))) {
            return 'mindfulness';
        }

        if ($logs->contains(fn (DailyHealthLog $log) => in_array($log->energy_level, ['very_low', 'low'], true))) {
            return 'nutrition';
        }

        if ($logs->contains(fn (DailyHealthLog $log) => in_array($log->energy_level, ['high', 'very_high'], true))) {
            return 'exercise';
        }

        return null;
    }

    private function hasPoorSleep(DailyHealthLog $log): bool
    {
        return $log->sleep_quality === 'bad'
            || in_array($log->sleep_duration, ['0_3', '3_6'], true);
    }

    private function hasPainOrLowMood(DailyHealthLog $log): bool
    {
        $pains = [
            $log->headache_intensity,
            $log->stomach_ache_intensity,
            $log->pelvic_pain_intensity,
            $log->back_pain_intensity,
        ];

        if (in_array('high', $pains, true) || in_array('medium', $pains, true)) {
            return true;
        }

        $moods = is_array($log->moods) ? $log->moods : [];

        return (bool) array_intersect($moods, ['anxious', 'sad', 'angry', 'frustrated']);
    }

    /**
     * Stable per-(user, day) seed — same challenge all day, different users get
     * different challenges on the same day.
     */
    private function seed(User $user, Carbon $date): int
    {
        return crc32($user->id.':'.$date->toDateString());
    }

    private function statusMessage(Challenge $challenge, bool $isCompleted, int $streak, string $locale): ?string
    {
        return $isCompleted
            ? $this->completedMessage($streak, $locale)
            : $challenge->localized('description', $locale);
    }

    /**
     * Encouragement shown once today's challenge is ticked off.
     */
    public function completedMessage(int $streak, string $locale): string
    {
        $fa = $locale === 'fa';

        if ($streak >= 2) {
            return $fa
                ? 'عالی! '.$this->num($streak).' روز پیاپی چالش‌هات رو انجام دادی، همین‌طور ادامه بده'
                : "Great! {$streak} days in a row — keep it going";
        }

        return $fa
            ? 'عالی! امروز این چالش رو انجام دادی، با همین روند ادامه بده'
            : "Great! You completed today's challenge, keep it up";
    }

    /** Persian digits for numbers embedded in Persian copy. */
    private function num(int $value): string
    {
        return strtr((string) $value, ['0' => '۰', '1' => '۱', '2' => '۲', '3' => '۳', '4' => '۴', '5' => '۵', '6' => '۶', '7' => '۷', '8' => '۸', '9' => '۹']);
    }
}
