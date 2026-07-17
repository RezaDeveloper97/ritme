<?php

namespace Tests\Feature;

use App\Models\CycleHistory;
use App\Models\User;
use App\Models\UserProfile;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * The rich, render-ready daily payload (spec §19) served under data.cycle_view on
 * /cycle/today and /cycle/date/{date}: daily card, predictions, three-layer values,
 * data status and confidence.
 */
class CycleViewApiTest extends TestCase
{
    use RefreshDatabase;

    private function actingUser(array $profileOverrides = []): User
    {
        $user = User::factory()->create(['mobile' => '09121234567']);

        UserProfile::create(array_merge([
            'user_id' => $user->id,
            'birthday' => '1995-05-15',
            'period_duration' => 5,
            'cycle_duration' => 28,
            'last_period_start' => now()->subDays(10)->toDateString(),
        ], $profileOverrides));

        Passport::actingAs($user);

        return $user;
    }

    public function test_today_returns_the_rich_cycle_view_payload(): void
    {
        $this->actingUser();

        $view = $this->getJson('/api/v1/cycle/today')->assertOk()->json('data.cycle_view');

        $this->assertIsInt($view['cycle_day']);
        $this->assertArrayHasKey('title', $view['daily_card']);
        $this->assertArrayHasKey('fertility_level', $view['daily_card']);
        $this->assertContains($view['data_status'], ['actual', 'predicted', 'needs_confirmation', 'incomplete']);
        $this->assertContains($view['fertility_level'], ['very_low', 'low', 'medium', 'high', 'very_high']);
        $this->assertArrayHasKey('next_period_start', $view['predictions']);
        $this->assertArrayHasKey('estimated_ovulation_date', $view['predictions']);
        $this->assertContains($view['data_quality']['confidence'], ['high', 'medium', 'low']);
        $this->assertIsArray($view['data_quality']['confidence_reasons']);
    }

    public function test_effective_values_fall_back_to_the_profile_without_enough_cycles(): void
    {
        $this->actingUser();

        $view = $this->getJson('/api/v1/cycle/today')->assertOk()->json('data.cycle_view');

        $this->assertSame(28, $view['effective_values']['cycle_length']);
        $this->assertSame(28, $view['profile_values']['cycle_length']);
        $this->assertNull($view['calculated_values']['cycle_length']); // fewer than 3 valid cycles
        $this->assertSame('profile', $view['effective_values']['source']);
    }

    public function test_a_day_inside_a_logged_period_is_actual(): void
    {
        $user = $this->actingUser(['last_period_start' => now()->subDays(2)->toDateString()]);

        CycleHistory::create([
            'user_id' => $user->id,
            'period_start_date' => now()->subDays(2)->toDateString(),
            'period_end_date' => now()->toDateString(),
            'bleeding_length' => 3,
            'is_confirmed' => true,
            'source' => 'user_logged',
        ]);

        $view = $this->getJson('/api/v1/cycle/today')->assertOk()->json('data.cycle_view');

        $this->assertSame('actual', $view['data_status']);
        $this->assertNotEmpty($view['daily_card']['title']);
    }

    public function test_daily_card_is_localised_to_persian(): void
    {
        $this->actingUser();

        $view = $this->getJson('/api/v1/cycle/today', ['Accept-Language' => 'fa'])
            ->assertOk()->json('data.cycle_view');

        $this->assertNotEmpty($view['daily_card']['fertility_label']);
        // Persian output contains non-ASCII characters.
        $this->assertMatchesRegularExpression('/[^\x00-\x7F]/u', $view['daily_card']['title']);
    }

    public function test_incomplete_profile_still_carries_three_layer_values(): void
    {
        $user = User::factory()->create(['mobile' => '09120000009']);
        UserProfile::create([
            'user_id' => $user->id,
            'period_duration' => 5,
            'cycle_duration' => 30,
            // no last_period_start → the engine has no anchor
        ]);
        Passport::actingAs($user);

        $view = $this->getJson('/api/v1/cycle/today')->assertOk()->json('data.cycle_view');

        $this->assertNull($view['cycle_day']);
        $this->assertNull($view['daily_card']);
        $this->assertSame(30, $view['profile_values']['cycle_length']);
        $this->assertSame(30, $view['effective_values']['cycle_length']);
    }

    public function test_a_forgotten_open_period_does_not_cover_days_after_a_newer_period(): void
    {
        $user = $this->actingUser(['last_period_start' => now()->subDays(5)->toDateString()]);

        // An old open period the user forgot to close…
        CycleHistory::create([
            'user_id' => $user->id,
            'period_start_date' => now()->subDays(30)->toDateString(),
            'is_confirmed' => true,
            'source' => 'user_logged',
        ]);
        // …and a newer, properly closed period after it.
        CycleHistory::create([
            'user_id' => $user->id,
            'period_start_date' => now()->subDays(10)->toDateString(),
            'period_end_date' => now()->subDays(6)->toDateString(),
            'bleeding_length' => 5,
            'is_confirmed' => true,
            'source' => 'user_logged',
        ]);

        // A day after the newer period must not read as "inside" the old open one.
        $date = now()->subDays(4)->toDateString();
        $view = $this->getJson("/api/v1/cycle/date/{$date}")->assertOk()->json('data.cycle_view');

        $this->assertNotSame('incomplete', $view['data_status']);
        $this->assertNotSame('actual', $view['data_status']);
    }
}
