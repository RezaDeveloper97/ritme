<?php

namespace Tests\Feature;

use App\Models\User;
use App\Models\UserProfile;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Http;
use Laravel\Passport\Passport;
use Tests\TestCase;

class ProfileNameUpdateTest extends TestCase
{
    use RefreshDatabase;

    private function actingUser(): User
    {
        $user = User::factory()->create(['mobile' => '09121234567', 'name' => 'Sara']);

        UserProfile::create([
            'user_id' => $user->id,
            'period_duration' => 5,
            'cycle_duration' => 28,
            'last_period_start' => now()->subDays(10)->toDateString(),
        ]);

        Passport::actingAs($user);

        return $user;
    }

    private function enableTelegram(): void
    {
        config([
            'services.telegram.token' => 'test-token',
            'services.telegram.chat_id' => '12345',
        ]);
    }

    public function test_it_updates_the_name(): void
    {
        $user = $this->actingUser();
        Http::fake();

        $this->postJson('/api/v1/profile', ['name' => 'Mina'])
            ->assertOk()
            ->assertJsonPath('success', true);

        $this->assertSame('Mina', $user->fresh()->name);
    }

    public function test_it_rejects_a_name_longer_than_the_limit(): void
    {
        $this->actingUser();

        $this->postJson('/api/v1/profile', ['name' => str_repeat('a', 256)])
            ->assertStatus(422);
    }

    public function test_it_announces_the_change_on_telegram(): void
    {
        $user = $this->actingUser();
        $this->enableTelegram();
        Http::fake(['api.telegram.org/*' => Http::response(['ok' => true])]);

        $this->postJson('/api/v1/profile', ['name' => 'Mina'])->assertOk();

        Http::assertSent(function ($request) use ($user) {
            return str_contains($request->url(), 'api.telegram.org/bottest-token/sendMessage')
                && $request['chat_id'] === '12345'
                && str_contains($request['text'], 'Sara')
                && str_contains($request['text'], 'Mina')
                && str_contains($request['text'], (string) $user->id);
        });
    }

    public function test_it_does_not_announce_when_the_name_is_unchanged(): void
    {
        $this->actingUser();
        $this->enableTelegram();
        Http::fake();

        $this->postJson('/api/v1/profile', ['name' => 'Sara'])->assertOk();

        Http::assertNothingSent();
    }

    public function test_a_telegram_failure_does_not_break_the_update(): void
    {
        $user = $this->actingUser();
        $this->enableTelegram();
        Http::fake(['api.telegram.org/*' => Http::response('nope', 500)]);

        $this->postJson('/api/v1/profile', ['name' => 'Mina'])->assertOk();

        $this->assertSame('Mina', $user->fresh()->name);
    }
}
