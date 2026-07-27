<?php

namespace Tests\Feature;

use App\Models\DailyHealthLog;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * Locks the contract the home "خلاصه هفته" card renders from: real 7-day
 * averages of mood/sleep/energy, null (not 0) when nothing was logged, and a
 * delta against the previous 7 days.
 */
class WeeklySummarySectionTest extends TestCase
{
    use RefreshDatabase;

    private function actingUser(): User
    {
        $user = User::factory()->create(['mobile' => '09121234567']);
        Passport::actingAs($user);

        return $user;
    }

    private function log(User $user, string $date, array $fields): void
    {
        DailyHealthLog::create(array_merge(['user_id' => $user->id, 'log_date' => $date], $fields));
    }

    private function items(array $json): array
    {
        return collect($json['data']['section']['data']['items'])->keyBy('key')->all();
    }

    /** No logs at all → every metric is null so the card shows dashes, not zeros. */
    public function test_metrics_are_null_without_logs(): void
    {
        $this->actingUser();

        $json = $this->getJson('/api/v1/home/sections/weekly_summary')->assertOk()->json();
        $items = $this->items($json);

        $this->assertNull($items['mood']['percent']);
        $this->assertNull($items['sleep']['percent']);
        $this->assertNull($items['energy']['percent']);
        $this->assertNull($items['mood']['delta']);
        $this->assertSame(0, $json['data']['section']['data']['logged_days']);
        $this->assertNull($json['data']['section']['data']['overall_percent']);
    }

    /** Percentages average the last 7 days only, and count logged days. */
    public function test_averages_cover_the_last_seven_days(): void
    {
        $user = $this->actingUser();

        // In-window: happy (100) and sad (20) → mood 60.
        $this->log($user, now()->toDateString(), ['moods' => ['happy'], 'sleep_quality' => 'good']);
        $this->log($user, now()->subDays(3)->toDateString(), ['moods' => ['sad']]);
        // Out of window — must not move the average.
        $this->log($user, now()->subDays(9)->toDateString(), ['moods' => ['sad']]);

        $json = $this->getJson('/api/v1/home/sections/weekly_summary')->assertOk()->json();
        $data = $json['data']['section']['data'];
        $items = $this->items($json);

        $this->assertSame(60, $items['mood']['percent']);
        $this->assertSame(100, $items['sleep']['percent']);
        $this->assertNull($items['energy']['percent']);
        $this->assertSame(2, $data['logged_days']);
        $this->assertSame(now()->subDays(6)->toDateString(), $data['range']['from']);
        $this->assertSame(now()->toDateString(), $data['range']['to']);
    }

    /** Delta compares this window with the 7 days before it; null when either side is unscored. */
    public function test_delta_compares_with_the_previous_week(): void
    {
        $user = $this->actingUser();

        $this->log($user, now()->toDateString(), ['moods' => ['happy']]);           // 100
        $this->log($user, now()->subDays(8)->toDateString(), ['moods' => ['sad']]); // 20, previous window

        $items = $this->items($this->getJson('/api/v1/home/sections/weekly_summary')->assertOk()->json());

        $this->assertSame(100, $items['mood']['percent']);
        $this->assertSame(20, $items['mood']['previous_percent']);
        $this->assertSame(80, $items['mood']['delta']);
        // Sleep was never logged in either window.
        $this->assertNull($items['sleep']['delta']);
    }

    /** Fatigue stands in for an explicit energy level so the tile is not empty. */
    public function test_energy_falls_back_to_the_fatigue_flag(): void
    {
        $user = $this->actingUser();
        $this->log($user, now()->toDateString(), ['fatigue' => true]);

        $items = $this->items($this->getJson('/api/v1/home/sections/weekly_summary')->assertOk()->json());

        $this->assertSame(30, $items['energy']['percent']);
    }

    /** The card is user-scoped — another account's logs never leak in. */
    public function test_other_users_logs_are_excluded(): void
    {
        $other = User::factory()->create(['mobile' => '09129999999']);
        $this->log($other, now()->toDateString(), ['moods' => ['happy']]);

        $this->actingUser();

        $items = $this->items($this->getJson('/api/v1/home/sections/weekly_summary')->assertOk()->json());

        $this->assertNull($items['mood']['percent']);
    }
}
