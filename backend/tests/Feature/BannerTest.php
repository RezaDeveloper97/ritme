<?php

namespace Tests\Feature;

use App\Models\Admin;
use App\Models\Banner;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\UploadedFile;
use Illuminate\Support\Facades\Storage;
use Laravel\Passport\Passport;
use Tests\TestCase;

class BannerTest extends TestCase
{
    use RefreshDatabase;

    private function admin(): Admin
    {
        return Admin::create([
            'name' => 'Test Admin',
            'email' => 'super@ritme.test',
            'password' => 'secret123',
            'role' => Admin::ROLE_SUPER,
            'is_active' => true,
        ]);
    }

    private function banner(array $overrides = []): Banner
    {
        return Banner::create(array_merge([
            'title' => ['fa' => 'بنر', 'en' => 'Banner'],
            'image_path' => 'banners/sample.jpg',
            'position' => 'home_top',
            'is_active' => true,
            'sort_order' => 0,
        ], $overrides));
    }

    // ── Model scope ─────────────────────────────────────────────

    public function test_active_scope_excludes_inactive_and_out_of_window_banners(): void
    {
        $live = $this->banner();
        $this->banner(['is_active' => false]);
        $this->banner(['starts_at' => now()->addDay()]);   // not started yet
        $this->banner(['ends_at' => now()->subDay()]);      // already ended

        $active = Banner::active()->get();

        $this->assertCount(1, $active);
        $this->assertTrue($active->first()->is($live));
    }

    public function test_active_scope_includes_banners_inside_their_window(): void
    {
        $this->banner(['starts_at' => now()->subDay(), 'ends_at' => now()->addDay()]);

        $this->assertCount(1, Banner::active()->get());
    }

    // ── Public API ──────────────────────────────────────────────

    public function test_banners_endpoint_requires_authentication(): void
    {
        $this->getJson('/api/v1/banners')->assertUnauthorized();
    }

    public function test_banners_endpoint_groups_active_banners_by_position(): void
    {
        Passport::actingAs(User::factory()->create());

        $second = $this->banner(['position' => 'home_top', 'sort_order' => 2]);
        $first = $this->banner(['position' => 'home_top', 'sort_order' => 1]);
        $this->banner(['position' => 'home_middle']);
        $this->banner(['position' => 'home_top', 'is_active' => false]); // excluded

        $response = $this->getJson('/api/v1/banners')->assertOk();

        $response->assertJsonPath('success', true);
        $response->assertJsonCount(2, 'data.positions.home_top');
        $response->assertJsonCount(1, 'data.positions.home_middle');
        $response->assertJsonCount(0, 'data.positions.home_bottom');

        // Ordered by sort_order ascending — the lower sort_order comes first.
        $top = $response->json('data.positions.home_top');
        $this->assertSame($first->id, $top[0]['id']);
        $this->assertSame($second->id, $top[1]['id']);
    }

    public function test_banner_payload_exposes_only_client_fields(): void
    {
        Passport::actingAs(User::factory()->create());
        $this->banner([
            'title' => ['fa' => 'جشنواره', 'en' => 'Sale'],
            'link_url' => 'https://example.com',
            'link_type' => 'external',
        ]);

        $banner = $this->getJson('/api/v1/banners', ['Accept-Language' => 'fa'])
            ->json('data.positions.home_top.0');

        $this->assertSame(['id', 'title', 'image_url', 'position', 'link_url', 'link_type'], array_keys($banner));
        $this->assertSame('جشنواره', $banner['title']); // localized per Accept-Language
        $this->assertStringContainsString('banners/sample.jpg', $banner['image_url']);
        $this->assertSame('external', $banner['link_type']);
    }

    public function test_position_query_restricts_to_one_slot(): void
    {
        Passport::actingAs(User::factory()->create());
        $this->banner(['position' => 'home_top']);
        $this->banner(['position' => 'home_middle']);

        $data = $this->getJson('/api/v1/banners?position=home_middle')->json('data.positions');

        $this->assertArrayHasKey('home_middle', $data);
        $this->assertArrayNotHasKey('home_top', $data);
        $this->assertCount(1, $data['home_middle']);
    }

    // ── Admin CRUD + image upload ───────────────────────────────

    public function test_admin_can_create_a_banner_with_an_uploaded_image(): void
    {
        Storage::fake('public');
        $this->actingAs($this->admin(), 'admin');

        $file = UploadedFile::fake()->image('promo.jpg', 1080, 540);

        $this->post('/admin/banners', [
            'title' => ['fa' => 'تخفیف', 'en' => 'Discount'],
            'image' => $file,
            'position' => 'home_top',
            'link_type' => 'internal',
            'link_url' => '/calendar',
            'sort_order' => 3,
            'is_active' => '1',
        ])->assertRedirect(route('admin.banners.index'));

        $banner = Banner::firstOrFail();
        $this->assertSame('home_top', $banner->position);
        $this->assertSame('/calendar', $banner->link_url);
        $this->assertSame('internal', $banner->link_type);
        $this->assertTrue($banner->is_active);
        Storage::disk('public')->assertExists($banner->image_path);
    }

    public function test_creating_a_banner_requires_an_image(): void
    {
        $this->actingAs($this->admin(), 'admin');

        $this->from(route('admin.banners.create'))
            ->post('/admin/banners', ['position' => 'home_top'])
            ->assertSessionHasErrors('image');

        $this->assertSame(0, Banner::count());
    }

    public function test_external_link_must_be_a_valid_url(): void
    {
        Storage::fake('public');
        $this->actingAs($this->admin(), 'admin');

        $this->from(route('admin.banners.create'))->post('/admin/banners', [
            'image' => UploadedFile::fake()->image('p.jpg', 1080, 540),
            'position' => 'home_top',
            'link_type' => 'external',
            'link_url' => 'not-a-url',
        ])->assertSessionHasErrors('link_url');
    }

    public function test_updating_replaces_the_image_and_deletes_the_old_one(): void
    {
        Storage::fake('public');
        $this->actingAs($this->admin(), 'admin');

        $old = UploadedFile::fake()->image('old.jpg', 1080, 540)->store('banners', 'public');
        $banner = $this->banner(['image_path' => $old]);

        $this->put(route('admin.banners.update', $banner), [
            'image' => UploadedFile::fake()->image('new.jpg', 1080, 540),
            'position' => 'home_bottom',
            'is_active' => '1',
        ])->assertRedirect(route('admin.banners.index'));

        $banner->refresh();
        $this->assertSame('home_bottom', $banner->position);
        Storage::disk('public')->assertMissing($old);
        Storage::disk('public')->assertExists($banner->image_path);
    }

    public function test_destroy_removes_the_row_and_the_file(): void
    {
        Storage::fake('public');
        $this->actingAs($this->admin(), 'admin');

        $path = UploadedFile::fake()->image('x.jpg', 1080, 540)->store('banners', 'public');
        $banner = $this->banner(['image_path' => $path]);

        $this->delete(route('admin.banners.destroy', $banner))
            ->assertRedirect();

        $this->assertModelMissing($banner);
        Storage::disk('public')->assertMissing($path);
    }

    public function test_toggle_flips_active_state(): void
    {
        $this->actingAs($this->admin(), 'admin');
        $banner = $this->banner(['is_active' => true]);

        $this->post(route('admin.banners.toggle', $banner))->assertRedirect();

        $this->assertFalse($banner->refresh()->is_active);
    }
}
