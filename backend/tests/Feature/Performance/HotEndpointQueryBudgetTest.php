<?php

namespace Tests\Feature\Performance;

use App\Enums\CyclePhase;
use App\Enums\RecommendationType;
use App\Models\CycleHistory;
use App\Models\DailyHealthLog;
use App\Models\Recommendation;
use App\Models\User;
use App\Models\UserProfile;
use Carbon\Carbon;
use Illuminate\Database\Events\QueryExecuted;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\DB;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * Query budgets for the hot read endpoints (T-M1-12, perf baseline §2.1–§2.2),
 * so the per-day N+1s and duplicate reads that were removed cannot come back.
 *
 * The user is seeded like the baseline's: a profile, a year of logged periods
 * and two months of daily logs. Budgets count every SQL statement the request
 * runs, including the language-registry read (cached in Redis in production).
 * `Passport::actingAs` skips the three token/client/user lookups a real request
 * makes, so a production request runs these budgets + 3 − 1.
 *
 * Baseline (production-like, incl. auth): /home 37, week_calendar 14,
 * /messages/daily 14, /cycle/today 8, /cycle/month 7.
 */
class HotEndpointQueryBudgetTest extends TestCase
{
    use RefreshDatabase;

    private User $user;

    protected function setUp(): void
    {
        parent::setUp();

        Carbon::setTestNow(Carbon::parse('2026-09-19 10:00:00'));

        $this->user = User::factory()->create(['mobile' => '09120000012']);
        UserProfile::create([
            'user_id' => $this->user->id,
            'birthday' => '1995-03-10',
            'weight' => 60,
            'height' => 165,
            'period_duration' => 5,
            'cycle_duration' => 29,
            'last_period_start' => '2026-09-05',
            'user_goal' => 'non_ttc',
        ]);

        $start = Carbon::parse('2025-09-20');
        foreach ([0, 28, 30, 29, 27, 31, 29, 28, 30, 29, 28, 30, 29] as $length) {
            $start->addDays($length);
            CycleHistory::create([
                'user_id' => $this->user->id,
                'period_start_date' => $start->toDateString(),
                'period_end_date' => $start->copy()->addDays(4)->toDateString(),
                'bleeding_length' => 5,
                'is_confirmed' => true,
                'source' => 'user_logged',
            ]);
        }

        $moods = ['happy', 'calm', 'anxious', 'sad', 'bored'];
        for ($i = 59; $i >= 0; $i--) {
            DailyHealthLog::create([
                'user_id' => $this->user->id,
                'log_date' => Carbon::parse('2026-09-19')->subDays($i)->toDateString(),
                'moods' => [$moods[$i % 5]],
                'headache_intensity' => $i % 7 === 0 ? 'mild' : null,
            ]);
        }

        // Installs always carry admin recommendations; without one the engine
        // adds an existence probe that production never runs.
        Recommendation::create([
            'type' => RecommendationType::NUTRITION->value,
            'text' => ['fa' => 'متن', 'en' => 'Text'],
            'cycle_phase' => CyclePhase::LUTEAL->value,
            'is_active' => true,
            'sort_order' => 0,
        ]);

        Passport::actingAs($this->user->fresh());
    }

    protected function tearDown(): void
    {
        Carbon::setTestNow();

        parent::tearDown();
    }

    public function test_home_has_no_per_day_log_queries_and_no_duplicates(): void
    {
        $queries = $this->queriesFor('/api/v1/home');

        $this->assertLessThanOrEqual(15, count($queries), $this->dump($queries));
        $this->assertNoDuplicates($queries);
        // One preloaded window (day log, week strip, 3/7-day and previous-week
        // windows) plus the message system's 90-day pattern window.
        $this->assertSame(2, $this->countOn($queries, 'daily_health_logs'), $this->dump($queries));
        $this->assertSame(1, $this->countOn($queries, 'cycle_histories'), $this->dump($queries));
        $this->assertSame(1, $this->countOn($queries, 'message_contents'), $this->dump($queries));
    }

    public function test_week_calendar_reads_logs_once(): void
    {
        $queries = $this->queriesFor('/api/v1/home/sections/week_calendar');

        $this->assertLessThanOrEqual(5, count($queries), $this->dump($queries));
        $this->assertNoDuplicates($queries);
        $this->assertSame(1, $this->countOn($queries, 'daily_health_logs'), $this->dump($queries));
    }

    public function test_daily_messages_read_message_contents_once(): void
    {
        $queries = $this->queriesFor('/api/v1/messages/daily');

        $this->assertLessThanOrEqual(8, count($queries), $this->dump($queries));
        $this->assertNoDuplicates($queries);
        $this->assertSame(1, $this->countOn($queries, 'message_contents'), $this->dump($queries));
        $this->assertSame(2, $this->countOn($queries, 'daily_health_logs'), $this->dump($queries));
    }

    public function test_smart_tip_reads_message_contents_once(): void
    {
        $queries = $this->queriesFor('/api/v1/home/sections/smart_tip');

        $this->assertLessThanOrEqual(8, count($queries), $this->dump($queries));
        $this->assertNoDuplicates($queries);
        $this->assertSame(1, $this->countOn($queries, 'message_contents'), $this->dump($queries));
    }

    public function test_cycle_today_and_date_read_history_once(): void
    {
        foreach (['/api/v1/cycle/today', '/api/v1/cycle/date/2026-09-19', '/api/v1/cycle/date/2026-10-05'] as $uri) {
            $queries = $this->queriesFor($uri);

            $this->assertLessThanOrEqual(5, count($queries), $uri.$this->dump($queries));
            $this->assertNoDuplicates($queries);
            $this->assertSame(1, $this->countOn($queries, 'cycle_histories'), $uri.$this->dump($queries));
        }
    }

    public function test_cycle_month_reads_each_input_once_in_both_views(): void
    {
        foreach (['/api/v1/cycle/month/2026/9', '/api/v1/cycle/month/2026/9?view=calendar'] as $uri) {
            $queries = $this->queriesFor($uri);

            $this->assertLessThanOrEqual(5, count($queries), $uri.$this->dump($queries));
            $this->assertNoDuplicates($queries);
            $this->assertSame(1, $this->countOn($queries, 'daily_health_logs'), $uri.$this->dump($queries));
            $this->assertSame(1, $this->countOn($queries, 'cycle_histories'), $uri.$this->dump($queries));
        }
    }

    /**
     * @return array<int, array{sql: string, bindings: array}>
     */
    private function queriesFor(string $uri): array
    {
        $queries = [];
        DB::listen(function (QueryExecuted $query) use (&$queries): void {
            $queries[] = ['sql' => $query->sql, 'bindings' => $query->bindings];
        });

        $this->getJson($uri)->assertOk();

        return $queries;
    }

    /** @param array<int, array{sql: string, bindings: array}> $queries */
    private function assertNoDuplicates(array $queries): void
    {
        $seen = [];
        foreach ($queries as $query) {
            $key = $query['sql'].json_encode($query['bindings']);
            $this->assertArrayNotHasKey($key, $seen, 'Duplicate query: '.$query['sql']);
            $seen[$key] = true;
        }
    }

    /** @param array<int, array{sql: string, bindings: array}> $queries */
    private function countOn(array $queries, string $table): int
    {
        return count(array_filter(
            $queries,
            fn (array $query): bool => (bool) preg_match('/\bfrom\s+["`]?'.$table.'["`]?\s/i', $query['sql'].' '),
        ));
    }

    /** @param array<int, array{sql: string, bindings: array}> $queries */
    private function dump(array $queries): string
    {
        return "\n".count($queries)." queries:\n".implode("\n", array_column($queries, 'sql'));
    }
}
