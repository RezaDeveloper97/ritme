<?php

namespace Tests\Feature;

use App\Models\User;
use App\Models\UserProfile;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

class DailyMessagesApiTest extends TestCase
{
    use RefreshDatabase;

    private function actingUser(bool $withProfile = true): User
    {
        $user = User::factory()->create(['mobile' => '09121234567']);

        if ($withProfile) {
            UserProfile::create([
                'user_id' => $user->id,
                'birthday' => '1995-05-15',
                'weight' => 60,
                'height' => 165,
                'period_duration' => 5,
                'cycle_duration' => 28,
                'last_period_start' => now()->subDays(10)->toDateString(),
            ]);
        }

        Passport::actingAs($user);

        return $user;
    }

    public function test_daily_returns_messages_for_complete_profile(): void
    {
        $this->actingUser();

        $response = $this->getJson('/api/v1/messages/daily');

        $response->assertOk()
            ->assertJsonPath('success', true)
            ->assertJsonPath('data.mode', 'cycle')
            ->assertJsonStructure([
                'data' => ['date', 'context_info', 'primary_message' => ['short_message']],
            ]);
    }

    public function test_daily_normalizes_browser_accept_language_header(): void
    {
        $this->actingUser();

        $response = $this->getJson('/api/v1/messages/daily', [
            'Accept-Language' => 'fa-IR,fa;q=0.9,en-US;q=0.8',
        ]);

        $response->assertOk();
        // Persian labels, not the English fallback that the raw header produced.
        $this->assertSame('فولیکولار', $response->json('data.context_info.phase_label'));
    }

    public function test_daily_rejects_invalid_date_with_422(): void
    {
        $this->actingUser();

        $this->getJson('/api/v1/messages/daily?date=bogus')
            ->assertStatus(422)
            ->assertJsonPath('success', false);
    }

    public function test_daily_rejects_invalid_mode_with_422(): void
    {
        $this->actingUser();

        $this->getJson('/api/v1/messages/daily?mode=nope')->assertStatus(422);
    }

    public function test_daily_requires_completed_profile(): void
    {
        $this->actingUser(withProfile: false);

        $this->getJson('/api/v1/messages/daily')->assertStatus(400);
    }

    public function test_daily_requires_authentication(): void
    {
        $this->getJson('/api/v1/messages/daily')->assertStatus(401);
    }
}
