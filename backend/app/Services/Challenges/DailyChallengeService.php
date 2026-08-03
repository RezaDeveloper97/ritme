<?php

namespace App\Services\Challenges;

use App\Models\Challenge;
use App\Models\DailyHealthLog;
use App\Models\User;
use App\Models\UserChallengeCompletion;
use Carbon\Carbon;
use Illuminate\Support\Collection;

/**
 * Picks "چالش امروز" for a user: one task from the admin-authored pool, which
 * the user can tick off. That is the whole feature — there is no streak, no
 * scoring and no progression, by product decision.
 *
 * Selection is layered, strongest signal first:
 *
 *  1. **Cycle day** — challenges are authored for a day range (1..35, see
 *     {@see Challenge::MAX_CYCLE_DAY}). Ranges that don't contain today's cycle
 *     day drop out, and when any challenge *does* target today specifically it
 *     outranks the untargeted ones — that is the whole point of authoring by
 *     day.
 *  2. **Recent-log signal** — poor sleep, pain, low energy or a logging gap
 *     steer the pick toward a matching category when the pool has one.
 *  3. **No repeats** — anything the user completed in the previous
 *     {@see self::REPEAT_WINDOW_DAYS} days drops out of the pool.
 *  4. **Deterministic per-user pick** — a hash of (user, date) selects from
 *     what's left, so the challenge is stable all day but differs per user.
 *
 * Every filter is *soft*: if it would empty the pool it is skipped, so a user
 * always gets a challenge as long as one active challenge exists — including
 * users with no cycle data at all, who simply never see day-targeted content.
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

    /**
     * The full "challenge" payload for a day, or null when no challenge is
     * available (empty/inactive pool).
     *
     * @param  int|null  $cycleDay  the user's cycle day for `$date`, or null when
     *                              her cycle isn't known (no profile / pregnancy mode).
     * @param  Collection<int, DailyHealthLog>|null  $recentLogs  pre-loaded logs
     *                                                            (HomeContext memoizes them); fetched on demand when omitted.
     * @return array<string, mixed>|null
     */
    public function payload(
        User $user,
        Carbon $date,
        string $locale,
        ?int $cycleDay = null,
        ?Collection $recentLogs = null,
    ): ?array {
        $challenge = $this->challengeFor($user, $date, $cycleDay, $recentLogs);

        if (! $challenge) {
            return null;
        }

        return [
            'id' => $challenge->id,
            'title' => $challenge->localized('title', $locale),
            'description' => $challenge->localized('description', $locale),
            'category' => $challenge->category,
            // Both the day the pick was made for and the range it was authored
            // for, so the card can say "روز ۷ چرخه" without a second request.
            'cycle_day' => $cycleDay,
            'cycle_day_from' => $challenge->cycle_day_from,
            'cycle_day_to' => $challenge->cycle_day_to,
            'is_completed' => $this->isCompleted($user, $challenge, $date),
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
        ?int $cycleDay = null,
        ?Collection $recentLogs = null,
    ): ?Challenge {
        $pool = Challenge::query()
            ->active()
            ->orderBy('sort_order')
            ->orderBy('id')
            ->get();

        if ($pool->isEmpty()) {
            return null;
        }

        $pool = $this->narrowToCycleDay($pool, $cycleDay);
        $pool = $this->withoutRecentlyCompleted($pool, $user, $date);
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
     * Keep the challenges that may run on this cycle day, preferring the ones
     * authored *for* it.
     *
     * Content written for a specific stretch of the cycle only pays off if it
     * actually surfaces there, and a handful of day-targeted rows would almost
     * never win a uniform draw against a large untargeted pool — so a matching
     * targeted challenge takes precedence over the general ones.
     *
     * @param  Collection<int, Challenge>  $pool
     * @return Collection<int, Challenge>
     */
    private function narrowToCycleDay(Collection $pool, ?int $cycleDay): Collection
    {
        $eligible = $pool->filter(fn (Challenge $c) => $c->appliesToCycleDay($cycleDay))->values();

        if ($eligible->isEmpty()) {
            return $pool;
        }

        $targeted = $eligible->filter(fn (Challenge $c) => $c->isDayTargeted())->values();

        return $targeted->isEmpty() ? $eligible : $targeted;
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
}
