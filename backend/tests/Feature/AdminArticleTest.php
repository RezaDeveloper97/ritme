<?php

namespace Tests\Feature;

use App\Enums\CyclePhase;
use App\Enums\CycleSubphase;
use App\Models\Admin;
use App\Models\Article;
use App\Services\Media\ImageOptimizer;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\UploadedFile;
use Illuminate\Support\Facades\Storage;
use Tests\TestCase;

/**
 * Admin article form: rich-text fields, the phase picker fed by the
 * phase-contents list, and the optimizing cover-image uploader.
 */
class AdminArticleTest extends TestCase
{
    use RefreshDatabase;

    private ?Admin $admin = null;

    private function admin(): Admin
    {
        return $this->admin ??= Admin::create([
            'name' => 'Test Admin',
            'email' => 'articles@ritme.test',
            'password' => 'secret123',
            'role' => Admin::ROLE_SUPER,
            'is_active' => true,
        ]);
    }

    private function article(array $overrides = []): Article
    {
        return Article::create(array_merge([
            'slug' => 'sample-'.uniqid(),
            'title' => ['fa' => 'نمونه', 'en' => 'Sample'],
            'is_published' => true,
        ], $overrides));
    }

    /**
     * @return array<string, mixed>
     */
    private function payload(array $overrides = []): array
    {
        return array_merge([
            'slug' => 'new-article',
            'title' => ['fa' => 'عنوان', 'en' => 'Title'],
            'is_published' => '1',
        ], $overrides);
    }

    // ── Rich-text editor ────────────────────────────────────────

    public function test_excerpt_and_body_render_as_rich_text_editors(): void
    {
        $response = $this->actingAs($this->admin(), 'admin')->get('/admin/articles/create');

        $response->assertOk()
            ->assertSee('data-rich-editor', escape: false)
            ->assertSee('name="excerpt[fa]"', escape: false)
            ->assertSee('name="body[fa]"', escape: false)
            // The editor is served from /admin so a single proxy rule covers it.
            ->assertSee('/admin/ckeditor/ckeditor.js', escape: false);
    }

    public function test_editor_html_is_stored_verbatim(): void
    {
        $this->actingAs($this->admin(), 'admin')->post('/admin/articles', $this->payload([
            'body' => ['fa' => '<p>سلام <strong>دنیا</strong></p>', 'en' => '<p>Hello</p>'],
        ]))->assertRedirect('/admin/articles');

        $this->assertSame('<p>سلام <strong>دنیا</strong></p>', Article::first()->body['fa']);
    }

    public function test_ckeditor_assets_are_served_under_the_admin_prefix(): void
    {
        $this->get('/admin/ckeditor/ckeditor.js')
            ->assertOk()
            ->assertHeader('Content-Type', 'application/javascript');

        $this->get('/admin/ckeditor/skins/moono-lisa/editor.css')->assertOk();
    }

    public function test_ckeditor_asset_route_rejects_path_traversal_and_unknown_types(): void
    {
        $this->get('/admin/ckeditor/../admin.css')->assertNotFound();
        $this->get('/admin/ckeditor/../../../.env')->assertNotFound();
        $this->get('/admin/ckeditor/LICENSE.md')->assertOk();      // known type
        $this->get('/admin/ckeditor/does-not-exist.js')->assertNotFound();
    }

    // ── Phase options (shared with /admin/phase-contents) ───────

    public function test_phase_picker_offers_the_phase_contents_list(): void
    {
        $response = $this->actingAs($this->admin(), 'admin')->get('/admin/articles/create');

        foreach (CycleSubphase::options() as $option) {
            $response->assertSee('value="'.$option['value'].'"', escape: false);
            $response->assertSee($option['label'], escape: false);
        }

        // Aliased sub-phases are left out: they share a label with the phase
        // they collapse onto, so ticking one would be a coin flip.
        $response->assertDontSee('value="menstrual"', escape: false);
        $response->assertDontSee('value="unknown"', escape: false);
    }

    public function test_the_picker_never_shows_the_same_label_twice(): void
    {
        $labels = array_column(CycleSubphase::options(), 'label');

        $this->assertSame($labels, array_values(array_unique($labels)));
    }

    public function test_an_article_can_be_tagged_with_several_phases(): void
    {
        $phases = [CycleSubphase::HIGH_FERTILITY->value, CycleSubphase::OVULATION_LIKELY->value];

        $this->actingAs($this->admin(), 'admin')
            ->post('/admin/articles', $this->payload(['cycle_phases' => $phases]))
            ->assertRedirect('/admin/articles');

        $this->assertSame($phases, Article::first()->cycle_phases);
    }

    public function test_ticking_no_phase_stores_a_general_article(): void
    {
        $this->actingAs($this->admin(), 'admin')
            ->post('/admin/articles', $this->payload())
            ->assertRedirect('/admin/articles');

        // null, not [] — "general" must stay one representation (see the scope).
        $this->assertNull(Article::first()->cycle_phases);
    }

    public function test_a_duplicated_phase_is_stored_once(): void
    {
        $duplicated = array_fill(0, 2, CycleSubphase::MID_LUTEAL->value);

        $this->actingAs($this->admin(), 'admin')
            ->post('/admin/articles', $this->payload(['cycle_phases' => $duplicated]))
            ->assertRedirect('/admin/articles');

        $this->assertSame([CycleSubphase::MID_LUTEAL->value], Article::first()->cycle_phases);
    }

    public function test_an_unknown_phase_is_rejected(): void
    {
        $this->actingAs($this->admin(), 'admin')
            ->post('/admin/articles', $this->payload([
                'cycle_phases' => [CycleSubphase::MID_LUTEAL->value, 'not_a_phase'],
            ]))
            ->assertSessionHasErrors('cycle_phases.1');

        $this->assertSame(0, Article::count());
    }

    public function test_a_legacy_main_phase_stays_selectable_when_editing(): void
    {
        $article = $this->article(['cycle_phases' => [CyclePhase::FOLLICULAR->value]]);

        $this->actingAs($this->admin(), 'admin')
            ->get('/admin/articles/'.$article->id.'/edit')
            ->assertOk()
            ->assertSee('value="follicular"', escape: false)
            ->assertSee('(قدیمی)', escape: false);

        $this->actingAs($this->admin(), 'admin')
            ->put('/admin/articles/'.$article->id, $this->payload([
                'slug' => $article->slug,
                'cycle_phases' => [CyclePhase::FOLLICULAR->value, CycleSubphase::MID_FOLLICULAR->value],
            ]))
            ->assertRedirect('/admin/articles');

        $this->assertSame(
            [CyclePhase::FOLLICULAR->value, CycleSubphase::MID_FOLLICULAR->value],
            $article->fresh()->cycle_phases,
        );
    }

    public function test_articles_are_matched_by_subphase_or_main_phase(): void
    {
        $sub = $this->article(['cycle_phases' => [CycleSubphase::HIGH_FERTILITY->value]]);
        $main = $this->article(['cycle_phases' => [CyclePhase::OVULATION->value]]);
        $general = $this->article(['cycle_phases' => null]);
        $other = $this->article(['cycle_phases' => [CycleSubphase::LATE_LUTEAL->value]]);
        // Tagged for several phases: matches through any one of them.
        $multi = $this->article(['cycle_phases' => [
            CycleSubphase::LATE_LUTEAL->value,
            CycleSubphase::HIGH_FERTILITY->value,
        ]]);

        $ids = Article::query()
            ->published()
            ->forPhase([CycleSubphase::HIGH_FERTILITY->value, CyclePhase::OVULATION->value])
            ->pluck('id');

        $this->assertTrue($ids->contains($sub->id));
        $this->assertTrue($ids->contains($main->id));
        $this->assertTrue($ids->contains($general->id));
        $this->assertTrue($ids->contains($multi->id));
        $this->assertFalse($ids->contains($other->id));
    }

    // ── Cover image upload ──────────────────────────────────────

    public function test_uploaded_cover_image_is_optimized_and_linked(): void
    {
        Storage::fake('public');

        $this->actingAs($this->admin(), 'admin')->post('/admin/articles', $this->payload([
            'image' => UploadedFile::fake()->image('cover.jpg', 3000, 2000),
        ]))->assertRedirect('/admin/articles');

        $article = Article::first();

        $this->assertNotNull($article->image_path);
        Storage::disk('public')->assertExists($article->image_path);
        $this->assertStringStartsWith('articles/', $article->image_path);

        // Scaled into the mobile-sized box, keeping the aspect ratio.
        $size = getimagesizefromstring(Storage::disk('public')->get($article->image_path));
        $this->assertSame(ImageOptimizer::MAX_WIDTH, $size[0]);
        $this->assertSame(720, $size[1]);

        // image_url resolves through the disk, so the host follows APP_URL.
        $this->assertStringContainsString($article->image_path, $article->image_url);
    }

    public function test_a_small_image_keeps_its_dimensions(): void
    {
        Storage::fake('public');

        $this->actingAs($this->admin(), 'admin')->post('/admin/articles', $this->payload([
            'image' => UploadedFile::fake()->image('small.png', 400, 300),
        ]))->assertRedirect('/admin/articles');

        $size = getimagesizefromstring(Storage::disk('public')->get(Article::first()->image_path));

        $this->assertSame([400, 300], [$size[0], $size[1]]);
    }

    public function test_non_image_uploads_are_rejected(): void
    {
        Storage::fake('public');

        $this->actingAs($this->admin(), 'admin')->post('/admin/articles', $this->payload([
            'image' => UploadedFile::fake()->create('report.pdf', 120, 'application/pdf'),
        ]))->assertSessionHasErrors('image');

        $this->assertSame(0, Article::count());
    }

    public function test_a_disallowed_image_format_is_rejected(): void
    {
        Storage::fake('public');

        $this->actingAs($this->admin(), 'admin')->post('/admin/articles', $this->payload([
            'image' => UploadedFile::fake()->create('animation.gif', 40, 'image/gif'),
        ]))->assertSessionHasErrors('image');

        $this->assertSame(0, Article::count());
    }

    public function test_replacing_the_image_deletes_the_previous_file(): void
    {
        Storage::fake('public');

        $article = $this->article(['image_path' => UploadedFile::fake()->image('old.jpg', 800, 600)->store('articles', 'public')]);
        $old = $article->image_path;

        $this->actingAs($this->admin(), 'admin')->put('/admin/articles/'.$article->id, $this->payload([
            'slug' => $article->slug,
            'image' => UploadedFile::fake()->image('new.jpg', 800, 600),
        ]))->assertRedirect('/admin/articles');

        $article->refresh();

        $this->assertNotSame($old, $article->image_path);
        Storage::disk('public')->assertMissing($old);
        Storage::disk('public')->assertExists($article->image_path);
    }

    public function test_the_image_can_be_removed(): void
    {
        Storage::fake('public');

        $article = $this->article(['image_path' => UploadedFile::fake()->image('cover.jpg', 800, 600)->store('articles', 'public')]);
        $path = $article->image_path;

        $this->actingAs($this->admin(), 'admin')->put('/admin/articles/'.$article->id, $this->payload([
            'slug' => $article->slug,
            'remove_image' => '1',
        ]))->assertRedirect('/admin/articles');

        $this->assertNull($article->fresh()->image_path);
        Storage::disk('public')->assertMissing($path);
    }

    public function test_deleting_an_article_removes_its_image(): void
    {
        Storage::fake('public');

        $article = $this->article(['image_path' => UploadedFile::fake()->image('cover.jpg', 800, 600)->store('articles', 'public')]);
        $path = $article->image_path;

        $this->actingAs($this->admin(), 'admin')
            ->delete('/admin/articles/'.$article->id)
            ->assertRedirect();

        Storage::disk('public')->assertMissing($path);
        $this->assertSame(0, Article::count());
    }

    public function test_an_external_url_is_used_when_no_file_was_uploaded(): void
    {
        $this->actingAs($this->admin(), 'admin')->post('/admin/articles', $this->payload([
            'image_url' => 'https://cdn.example.com/a.jpg',
        ]))->assertRedirect('/admin/articles');

        $this->assertSame('https://cdn.example.com/a.jpg', Article::first()->image_url);
    }
}
