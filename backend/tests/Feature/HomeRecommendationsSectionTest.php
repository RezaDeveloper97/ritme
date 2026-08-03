<?php

namespace Tests\Feature;

use App\Enums\CyclePhase;
use App\Enums\RecommendationType;
use App\Models\Recommendation;
use App\Models\User;
use App\Models\UserProfile;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * Locks the contract the home "توصیه‌های امروز" card renders from: the
 * recommendations an admin defined for the phase the user is in today, each
 * carrying the category's icon and the resolved heading — never a hardcoded
 * list.
 */
class HomeRecommendationsSectionTest extends TestCase
{
    use RefreshDatabase;

    /** A user bleeding today, so the engine puts them in the menstrual phase. */
    private function actingUser(): User
    {
        $user = User::factory()->create(['mobile' => '09121234567']);
        UserProfile::create([
            'user_id' => $user->id,
            'last_period_start' => now()->toDateString(),
            'cycle_duration' => 28,
            'period_duration' => 5,
        ]);
        Passport::actingAs($user);

        return $user;
    }

    private function recommendation(array $overrides = []): Recommendation
    {
        return Recommendation::create(array_merge([
            'type' => RecommendationType::NUTRITION->value,
            'text' => ['fa' => 'آهن بخورید', 'en' => 'Eat iron'],
            'cycle_phase' => CyclePhase::MENSTRUATION->value,
            'is_active' => true,
            'sort_order' => 0,
        ], $overrides));
    }

    private function section(): ?array
    {
        return $this->getJson('/api/v1/home/sections/recommendations', ['Accept-Language' => 'fa'])
            ->assertOk()
            ->json('data.section');
    }

    public function test_it_returns_the_phase_recommendations_with_icon_and_title(): void
    {
        $this->actingUser();
        $this->recommendation();

        $section = $this->section();

        $this->assertSame('recommendations', $section['type']);
        $this->assertSame([[
            'type' => 'nutrition',
            'icon' => RecommendationType::NUTRITION->icon(),
            'title' => RecommendationType::NUTRITION->label('fa'),
            'text' => 'آهن بخورید',
        ]], $section['data']['items']);
        $this->assertSame([['type' => 'nutrition', 'icon' => RecommendationType::NUTRITION->icon()]], $section['data']['tags']);
    }

    /** An admin's own heading replaces the category label. */
    public function test_an_admin_title_wins_over_the_category_label(): void
    {
        $this->actingUser();
        $this->recommendation(['title' => ['fa' => 'عنوان دلخواه', 'en' => 'Custom']]);

        $this->assertSame('عنوان دلخواه', $this->section()['data']['items'][0]['title']);
    }

    public function test_recommendations_for_another_phase_are_left_out(): void
    {
        $this->actingUser();
        $this->recommendation(['cycle_phase' => CyclePhase::LUTEAL->value]);

        // Every row belongs to a phase the user is not in, so the engine
        // resolves nothing and the card is dropped rather than rendered empty.
        $this->assertNull($this->section());
    }

    public function test_section_is_absent_when_every_recommendation_is_inactive(): void
    {
        $this->actingUser();
        $this->recommendation(['is_active' => false]);

        $this->assertNull($this->section());
    }
}
