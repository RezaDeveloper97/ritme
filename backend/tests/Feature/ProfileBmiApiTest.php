<?php

namespace Tests\Feature;

use App\Models\MessageContent;
use App\Models\User;
use App\Models\UserProfile;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

class ProfileBmiApiTest extends TestCase
{
    use RefreshDatabase;

    private function actingUser(array $profile = []): User
    {
        $user = User::factory()->create(['mobile' => '09121234567', 'name' => 'Sara']);
        UserProfile::create(array_merge(['user_id' => $user->id], $profile));
        Passport::actingAs($user);

        return $user;
    }

    public function test_bmi_is_computed_and_categorized(): void
    {
        // 60kg at 165cm => 22.0 => normal band.
        $this->actingUser(['weight' => 60, 'height' => 165]);

        // JSON serializes a whole-number float (22.0) back as an int.
        $this->getJson('/api/v1/profile')
            ->assertOk()
            ->assertJsonPath('data.bmi.value', 22)
            ->assertJsonPath('data.bmi.category', 'normal')
            ->assertJsonStructure(['data' => ['bmi' => ['value', 'category', 'category_label', 'message']]]);
    }

    public function test_boundary_value_lands_in_higher_band(): void
    {
        // 68.07kg at 165cm => raw ≈ 25.003 => overweight (band boundary is 25).
        $this->actingUser(['weight' => 68.07, 'height' => 165]);

        $this->getJson('/api/v1/profile')
            ->assertOk()
            ->assertJsonPath('data.bmi.category', 'overweight');
    }

    public function test_value_that_only_rounds_onto_a_boundary_keeps_true_band(): void
    {
        // Raw BMI ≈ 24.96 (normal); it rounds for display to 25.0 but must NOT
        // be reclassified as overweight.
        $this->actingUser(['weight' => 67.95, 'height' => 165]);

        $this->getJson('/api/v1/profile')
            ->assertOk()
            ->assertJsonPath('data.bmi.value', 25)
            ->assertJsonPath('data.bmi.category', 'normal');
    }

    public function test_bmi_is_null_when_metrics_missing(): void
    {
        $this->actingUser(['weight' => null, 'height' => null]);

        $this->getJson('/api/v1/profile')
            ->assertOk()
            ->assertJsonPath('data.bmi', null);
    }

    public function test_message_prefers_editable_db_content(): void
    {
        $this->actingUser(['weight' => 50, 'height' => 175]); // ~16.3 => underweight

        MessageContent::create([
            'group' => 'bmi_message',
            'item_key' => 'underweight',
            'locale' => 'fa',
            'label' => 'bmi_message / underweight',
            'payload' => ['message' => 'متن سفارشی ادمین'],
            'is_active' => true,
            'is_approved' => true,
        ]);

        $this->getJson('/api/v1/profile', ['Accept-Language' => 'fa'])
            ->assertOk()
            ->assertJsonPath('data.bmi.category', 'underweight')
            ->assertJsonPath('data.bmi.message', 'متن سفارشی ادمین');
    }
}
