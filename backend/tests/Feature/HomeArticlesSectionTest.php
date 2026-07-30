<?php

namespace Tests\Feature;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Models\Article;
use App\Models\User;
use App\Models\UserProfile;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\UploadedFile;
use Illuminate\Support\Facades\Storage;
use Laravel\Passport\Passport;
use Tests\TestCase;

/**
 * Locks the contract the home "بر اساس سیکل فعلی شما" row renders from: the
 * articles an admin published for the phase the user is in today, plus the
 * general ones — never a hardcoded list.
 */
class HomeArticlesSectionTest extends TestCase
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

    private function article(array $overrides = []): Article
    {
        static $n = 0;
        $n++;

        return Article::create(array_merge([
            'slug' => 'article-'.$n,
            'title' => ['fa' => 'عنوان '.$n, 'en' => 'Title '.$n],
            'is_published' => true,
        ], $overrides));
    }

    /** The section as the Persian app requests it (locale comes from the header). */
    private function section(): ?array
    {
        return $this->getJson('/api/v1/home/sections/articles', ['Accept-Language' => 'fa'])
            ->assertOk()
            ->json('data.section');
    }

    public function test_section_is_absent_while_nothing_is_published(): void
    {
        $this->actingUser();

        $this->assertNull($this->section());
    }

    public function test_it_returns_articles_for_the_current_phase_and_general_ones(): void
    {
        $this->actingUser();

        $phaseTagged = $this->article(['cycle_phases' => [CyclePhase::MENSTRUATION->value]]);
        $subphaseTagged = $this->article(['cycle_phases' => [CycleSubphase::MENSTRUATION->value]]);
        $general = $this->article(['cycle_phases' => null]);
        $otherPhase = $this->article(['cycle_phases' => [CycleSubphase::LATE_LUTEAL->value]]);
        $draft = $this->article(['cycle_phases' => null, 'is_published' => false]);
        // Tagged for two phases — one of them is where the user is today.
        $multiPhase = $this->article(['cycle_phases' => [
            CycleSubphase::LATE_LUTEAL->value,
            CycleSubphase::MENSTRUATION->value,
        ]]);

        $ids = collect($this->section()['data']['items'])->pluck('id');

        $this->assertTrue($ids->contains($phaseTagged->id));
        $this->assertTrue($ids->contains($subphaseTagged->id));
        $this->assertTrue($ids->contains($general->id));
        $this->assertTrue($ids->contains($multiPhase->id));
        $this->assertFalse($ids->contains($otherPhase->id));
        $this->assertFalse($ids->contains($draft->id));
    }

    /**
     * The engine can report an aliased sub-phase (v1.1 spellings such as
     * "menstrual"), while the admin panel only offers the canonical key. An
     * article tagged with the canonical phase must still surface.
     */
    public function test_an_aliased_subphase_matches_the_canonical_tag(): void
    {
        $this->actingUser();
        $article = $this->article(['cycle_phases' => [CycleSubphase::MENSTRUATION->value]]);

        $this->assertSame(
            CycleSubphase::MENSTRUATION,
            CycleSubphase::MENSTRUAL->canonical(),
        );

        $ids = collect($this->section()['data']['items'])->pluck('id');
        $this->assertTrue($ids->contains($article->id));
    }

    public function test_items_carry_what_the_card_renders(): void
    {
        Storage::fake('public');
        $this->actingUser();

        $article = $this->article([
            'excerpt' => ['fa' => '<p>خلاصه <strong>کوتاه</strong></p>', 'en' => '<p>Short</p>'],
            'read_time_minutes' => 4,
            'category' => 'سلامت',
            'image_path' => UploadedFile::fake()->image('cover.jpg', 800, 600)->store('articles', 'public'),
        ]);

        $item = collect($this->section()['data']['items'])->firstWhere('id', $article->id);

        $this->assertSame('عنوان '.substr($article->slug, 8), $item['title']);
        // The editor's markup is stripped for the card summary.
        $this->assertSame('خلاصه کوتاه', $item['excerpt']);
        $this->assertSame(4, $item['read_time_minutes']);
        $this->assertSame('سلامت', $item['category']);
        $this->assertSame([], $item['cycle_phases']);
        $this->assertStringContainsString($article->image_path, $item['image_url']);
    }

    public function test_it_localizes_to_the_requested_locale(): void
    {
        $this->actingUser();
        $this->article(['title' => ['fa' => 'فارسی', 'en' => 'English']]);

        $section = $this->getJson('/api/v1/home/sections/articles', ['Accept-Language' => 'en'])
            ->assertOk()
            ->json('data.section');

        $this->assertSame('Based on your current cycle', $section['title']);
        $this->assertSame('English', $section['data']['items'][0]['title']);
    }
}
