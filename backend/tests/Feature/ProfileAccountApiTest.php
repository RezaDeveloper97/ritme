<?php

namespace Tests\Feature;

use App\Models\User;
use App\Models\UserProfile;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

class ProfileAccountApiTest extends TestCase
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

    public function test_export_returns_all_data_groups(): void
    {
        $this->actingUser();

        $this->getJson('/api/v1/profile/export')
            ->assertOk()
            ->assertJsonPath('success', true)
            ->assertJsonStructure([
                'data' => [
                    'exported_at', 'account', 'profile', 'health_logs',
                    'cycle_histories', 'pregnancy', 'reminders',
                ],
            ]);
    }

    public function test_delete_account_removes_user_and_data(): void
    {
        $user = $this->actingUser();

        $this->deleteJson('/api/v1/account')
            ->assertOk()
            ->assertJsonPath('success', true);

        $this->assertDatabaseMissing('users', ['id' => $user->id]);
        $this->assertDatabaseMissing('user_profiles', ['user_id' => $user->id]);
    }

    public function test_reminder_crud_lifecycle(): void
    {
        $this->actingUser();

        $created = $this->postJson('/api/v1/reminders', [
            'type' => 'medication',
            'title' => 'ویتامین D',
            'recurrence' => 'daily',
            'recurrence_time' => '16:00',
        ])->assertCreated()->json('data');

        $this->getJson('/api/v1/reminders')
            ->assertOk()
            ->assertJsonCount(1, 'data');

        $this->putJson("/api/v1/reminders/{$created['id']}", ['is_active' => false])
            ->assertOk()
            ->assertJsonPath('data.is_active', false);

        $this->deleteJson("/api/v1/reminders/{$created['id']}")->assertOk();

        $this->getJson('/api/v1/reminders')->assertJsonCount(0, 'data');
    }

    public function test_reminder_validation_rejects_bad_payload(): void
    {
        $this->actingUser();

        $this->postJson('/api/v1/reminders', ['type' => 'bogus'])
            ->assertStatus(422);
    }

    public function test_reminders_are_scoped_to_owner(): void
    {
        $this->actingUser();
        $other = User::factory()->create(['mobile' => '09129999999']);
        $foreign = $other->reminders()->create(['type' => 'custom', 'title' => 'x']);

        $this->putJson("/api/v1/reminders/{$foreign->id}", ['title' => 'y'])
            ->assertStatus(404);
        $this->deleteJson("/api/v1/reminders/{$foreign->id}")->assertStatus(404);
    }
}
