<?php

namespace Tests\Feature;

use App\Http\Middleware\TestClock;
use App\Models\User;
use App\Services\Sms\Providers\LogSmsProvider;
use Carbon\Carbon;
use Database\Seeders\ContractFixtureSeeder;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Route;
use Illuminate\Support\Facades\Schema;
use Laravel\Passport\Passport;
use RuntimeException;
use Tests\TestCase;

/**
 * The pieces that make Laravel's responses reproducible for the Go contract
 * goldens: the fixture seeder, the X-Test-Now clock and the log SMS provider.
 */
class ContractFixtureTest extends TestCase
{
    use RefreshDatabase;

    private const NOW = '2026-09-23T10:00:00+03:30';

    /** Far from the real date, so "ignored" can't pass by coincidence. */
    private const FAR = '2020-02-02T10:00:00+03:30';

    protected function setUp(): void
    {
        parent::setUp();

        Route::middleware('api')->get('/api/_contract/now', fn () => response()->json([
            'now' => Carbon::now()->toIso8601String(),
            'today' => Carbon::today()->toDateString(),
        ]));
    }

    protected function tearDown(): void
    {
        $this->setClockFlag(null);

        parent::tearDown();
    }

    public function test_seeder_is_idempotent(): void
    {
        $this->seed(ContractFixtureSeeder::class);
        $first = $this->rowCounts();

        $this->seed(ContractFixtureSeeder::class);
        $second = $this->rowCounts();

        $this->assertSame($first, $second);
        $this->assertSame(count(ContractFixtureSeeder::PERSONAS), $first['users']);
        $this->assertGreaterThan(0, $first['pregnancy_weekly_content']);
        $this->assertSame(3, $first['banners']);
        $this->assertSame(['fa', 'en', 'ar'], DB::table('languages')->where('is_active', true)->orderBy('sort_order')->pluck('code')->all());
        $this->assertSame(6, DB::table('cycle_histories')->where('user_id', ContractFixtureSeeder::PERSONAS['regular'][0])->count());
    }

    public function test_seeder_is_independent_of_the_real_clock(): void
    {
        Carbon::setTestNow('2031-01-01 12:00:00');
        $this->seed(ContractFixtureSeeder::class);
        Carbon::setTestNow();

        $this->assertSame('2026-09-14', DB::table('user_profiles')->where('user_id', ContractFixtureSeeder::PERSONAS['regular'][0])->value('last_period_start'));
        $this->assertSame(ContractFixtureSeeder::CONTRACT_NOW, (string) DB::table('articles')->orderBy('id')->value('published_at'));
    }

    public function test_clock_header_freezes_now_when_enabled(): void
    {
        $this->setClockFlag('true');

        $this->getJson('/api/_contract/now', [TestClock::HEADER => self::NOW])
            ->assertOk()
            ->assertExactJson(['now' => self::NOW, 'today' => '2026-09-23']);

        // Restored after the request.
        $this->assertNull(Carbon::getTestNow());
    }

    public function test_invalid_clock_header_is_rejected_when_enabled(): void
    {
        $this->setClockFlag('true');

        $this->getJson('/api/_contract/now', [TestClock::HEADER => 'not-a-date'])->assertStatus(400);
    }

    public function test_clock_header_is_ignored_when_flag_is_off(): void
    {
        $this->setClockFlag(null);

        $this->getJson('/api/_contract/now', [TestClock::HEADER => self::FAR])
            ->assertOk()
            ->assertJsonMissing(['today' => '2020-02-02']);
    }

    public function test_clock_header_is_ignored_in_production_even_with_flag_on(): void
    {
        $this->setClockFlag('true');
        $this->app['env'] = 'production';

        $this->assertFalse(TestClock::isEnabled());
        $this->getJson('/api/_contract/now', [TestClock::HEADER => self::FAR])
            ->assertOk()
            ->assertJsonMissing(['today' => '2020-02-02']);
    }

    public function test_cycle_today_for_regular_persona_is_reproducible(): void
    {
        $this->seed(ContractFixtureSeeder::class);
        $this->setClockFlag('true');
        Passport::actingAs(User::findOrFail(ContractFixtureSeeder::PERSONAS['regular'][0]));

        $first = $this->getJson('/api/v1/cycle/today', [TestClock::HEADER => self::NOW])->assertOk();
        $second = $this->getJson('/api/v1/cycle/today', [TestClock::HEADER => self::NOW])->assertOk();

        $this->assertSame($first->getContent(), $second->getContent());
        // Last period started 2026-09-14 → cycle day 10 on the fixed clock.
        $this->assertStringContainsString('2026-09-14', $first->getContent());
    }

    public function test_log_sms_provider_logs_and_refuses_production(): void
    {
        $this->assertTrue((new LogSmsProvider)->sendOtp('09900000001', '1234', 'login_otp'));

        $this->app['env'] = 'production';
        $this->expectException(RuntimeException::class);
        new LogSmsProvider;
    }

    /** @return array<string, int> */
    private function rowCounts(): array
    {
        $tables = [
            'users', 'user_profiles', 'cycle_histories', 'daily_health_logs', 'pregnancy_profiles',
            'pregnancy_symptom_logs', 'pregnancy_weekly_logs', 'pregnancy_fetal_movements', 'pregnancy_alerts',
            'reminders', 'user_notifications', 'user_task_completions', 'user_challenge_completions',
            'languages', 'banners', 'message_contents', 'recommendations', 'pregnancy_weekly_content',
            'phase_contents', 'info_sections', 'challenges', 'task_templates', 'articles', 'affirmations',
        ];

        $counts = [];
        foreach ($tables as $table) {
            $this->assertTrue(Schema::hasTable($table), $table);
            $counts[$table] = DB::table($table)->count();
        }

        return $counts;
    }

    private function setClockFlag(?string $value): void
    {
        if ($value === null) {
            unset($_ENV['TEST_CLOCK_ENABLED'], $_SERVER['TEST_CLOCK_ENABLED']);
            putenv('TEST_CLOCK_ENABLED');

            return;
        }

        $_ENV['TEST_CLOCK_ENABLED'] = $_SERVER['TEST_CLOCK_ENABLED'] = $value;
        putenv("TEST_CLOCK_ENABLED={$value}");
    }
}
