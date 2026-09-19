<?php

namespace Tests\Feature\Performance;

use App\Models\CycleHistory;
use App\Models\DailyHealthLog;
use App\Models\MessageContent;
use App\Models\User;
use App\Models\UserProfile;
use App\Services\HealthEngine\CycleEngineCache;
use App\Services\HealthEngine\HealthDataEngine;
use App\Services\MessageSystem\Support\MessageContentRepository;
use Carbon\Carbon;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * The opt-in slim `/cycle/month?view=calendar` payload, the per-user engine
 * cache (T-M1-12) and the per-request message-content load.
 */
class CycleEngineCacheTest extends TestCase
{
    use RefreshDatabase;

    private const CALENDAR_FIELDS = [
        'calculation_date', 'cycle_day', 'phase', 'subphase', 'estimated_ovulation_day',
        'cycle_length_used', 'is_fertile_window', 'is_pms_window', 'is_period_tomorrow',
        'final_probability', 'cycle_variability',
    ];

    protected function setUp(): void
    {
        parent::setUp();

        Carbon::setTestNow(Carbon::parse('2026-09-19 10:00:00'));
    }

    protected function tearDown(): void
    {
        Carbon::setTestNow();

        parent::tearDown();
    }

    private function userWithCycle(string $mobile = '09120000031', string $lmp = '2026-09-05'): User
    {
        $user = User::factory()->create(['mobile' => $mobile]);
        UserProfile::create([
            'user_id' => $user->id,
            'birthday' => '1995-03-10',
            'weight' => 60,
            'height' => 165,
            'period_duration' => 5,
            'cycle_duration' => 28,
            'last_period_start' => $lmp,
        ]);
        CycleHistory::create([
            'user_id' => $user->id,
            'period_start_date' => $lmp,
            'period_end_date' => Carbon::parse($lmp)->addDays(4)->toDateString(),
            'bleeding_length' => 5,
            'is_confirmed' => true,
            'source' => 'user_logged',
        ]);

        return $user;
    }

    public function test_month_default_view_keeps_the_full_payload(): void
    {
        Passport::actingAs($this->userWithCycle());

        $day = $this->getJson('/api/v1/cycle/month/2026/9')->assertOk()->json('data.calculations.10');

        foreach (['text_flags', 'daily_tips', 'source_profile_data', 'source_daily_log_data', 'cycle_score', 'age_factor'] as $key) {
            $this->assertArrayHasKey($key, $day);
        }
        $this->assertSame(
            $this->getJson('/api/v1/cycle/month/2026/9')->json(),
            $this->getJson('/api/v1/cycle/month/2026/9?view=full')->json(),
        );
    }

    public function test_calendar_view_returns_only_the_calendar_fields_with_the_same_values(): void
    {
        Passport::actingAs($this->userWithCycle());

        $full = $this->getJson('/api/v1/cycle/month/2026/9')->assertOk()->json('data');
        $calendar = $this->getJson('/api/v1/cycle/month/2026/9?view=calendar')
            ->assertOk()
            ->assertJsonPath('success', true)
            ->assertJsonStructure(['data' => ['calculations', 'calculation_status', 'is_recalculating', 'month_summary']])
            ->json('data');

        $this->assertCount(30, $calendar['calculations']);
        foreach ($calendar['calculations'] as $i => $day) {
            $this->assertSame(self::CALENDAR_FIELDS, array_keys($day));
            $this->assertSame(array_intersect_key($full['calculations'][$i], $day), $day);
        }
        $this->assertSame($full['month_summary'], $calendar['month_summary']);
    }

    public function test_calendar_view_keeps_the_empty_day_sentinel_for_an_incomplete_profile(): void
    {
        $user = User::factory()->create(['mobile' => '09120000032']);
        Passport::actingAs($user);

        $day = $this->getJson('/api/v1/cycle/month/2026/9?view=calendar')->assertOk()->json('data.calculations.0');

        $this->assertNull($day['cycle_day']);
        $this->assertArrayNotHasKey('text_flags', $day);
    }

    public function test_unknown_view_is_rejected(): void
    {
        Passport::actingAs($this->userWithCycle());

        $this->getJson('/api/v1/cycle/month/2026/9?view=everything')
            ->assertStatus(422)
            ->assertJsonPath('success', false);
    }

    public function test_month_preload_includes_the_first_day_of_the_month(): void
    {
        $user = $this->userWithCycle();
        DailyHealthLog::create(['user_id' => $user->id, 'log_date' => '2026-09-01', 'moods' => ['sad']]);
        Passport::actingAs($user);

        $firstDay = $this->getJson('/api/v1/cycle/month/2026/9')->json('data.calculations.0');

        $this->assertSame('2026-09-01', $firstDay['source_daily_log_data']['log_date'] ?? null);
    }

    public function test_a_write_is_visible_on_the_next_read_even_within_the_same_second(): void
    {
        $user = $this->userWithCycle();
        Passport::actingAs($user);

        $before = $this->getJson('/api/v1/cycle/month/2026/9')->json('data.calculations.14.source_daily_log_data');
        $this->assertNull($before);

        // Same frozen second: a timestamp-based key would not notice this write.
        DailyHealthLog::create(['user_id' => $user->id, 'log_date' => '2026-09-15', 'moods' => ['happy']]);

        $after = $this->getJson('/api/v1/cycle/month/2026/9')->json('data.calculations.14.source_daily_log_data');
        $this->assertSame(['happy'], $after['moods'] ?? null);

        // A period edit moves every day's cycle_day on the next read too.
        $dayBefore = $this->getJson('/api/v1/cycle/today')->json('data.calculation.cycle_day');
        CycleHistory::where('user_id', $user->id)->update(['period_start_date' => '2026-09-10 00:00:00']);
        $dayAfter = $this->getJson('/api/v1/cycle/today')->json('data.calculation.cycle_day');
        $this->assertNotSame($dayBefore, $dayAfter);
    }

    public function test_engine_results_are_cached_per_inputs_and_per_user(): void
    {
        $alice = $this->userWithCycle('09120000033', '2026-09-05');
        $bob = $this->userWithCycle('09120000034', '2026-09-12');
        $cache = new CycleEngineCache;
        $day = Carbon::parse('2026-09-19');
        $runs = 0;
        $compute = function (HealthDataEngine $engine) use (&$runs, $day): array {
            $runs++;

            return $engine->calculateForDate($day, false);
        };
        $remember = fn (User $user) => $cache->remember(
            $engine = new HealthDataEngine($user->fresh(), 'fa'),
            'test-day',
            $day,
            $day,
            false,
            fn () => $compute($engine),
        );

        $first = $remember($alice);
        $this->assertSame($first, $remember($alice));
        $this->assertSame(1, $runs, 'the second identical read is served from the cache');

        // Another user with other data never receives Alice's entry.
        $this->assertNotSame($first['cycle_day'], $remember($bob)['cycle_day']);
        $this->assertSame(2, $runs);

        // Changing an input (a new log in the range) is a new key.
        DailyHealthLog::create(['user_id' => $alice->id, 'log_date' => '2026-09-19', 'moods' => ['calm']]);
        $remember($alice);
        $this->assertSame(3, $runs);
    }

    public function test_message_contents_are_read_once_per_locale_and_an_admin_edit_shows_next_request(): void
    {
        $row = MessageContent::create([
            'group' => 'sleep_cycle', 'item_key' => 'luteal', 'locale' => 'fa',
            'payload' => ['tip' => 'قدیمی'], 'is_active' => true, 'is_approved' => true,
        ]);
        MessageContent::create([
            'group' => 'nutrition_cycle', 'item_key' => 'luteal', 'locale' => 'fa',
            'payload' => ['tip' => 'غذا'], 'is_active' => true, 'is_approved' => false,
        ]);

        $repository = new MessageContentRepository;
        $this->assertSame(['tip' => 'قدیمی'], $repository->payload('sleep_cycle', 'luteal', 'fa'));
        // Unapproved rows are never served; the caller's fallback applies.
        $this->assertSame(['tip' => 'fallback'], $repository->resolve('nutrition_cycle', 'luteal', 'fa', ['tip' => 'fallback']));
        $this->assertNull($repository->payload('sleep_cycle', 'luteal', 'en'));

        $row->update(['payload' => ['tip' => 'جدید']]);

        // The next request (a fresh singleton) sees the edit immediately.
        $this->assertSame(['tip' => 'جدید'], (new MessageContentRepository)->payload('sleep_cycle', 'luteal', 'fa'));
    }
}
