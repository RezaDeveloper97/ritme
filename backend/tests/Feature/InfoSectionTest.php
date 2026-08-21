<?php

namespace Tests\Feature;

use App\Models\Admin;
use App\Models\InfoSection;
use Database\Seeders\InfoSectionSeeder;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

/**
 * The admin-managed text screens: GET /api/v1/info/{group} serves the boxes an
 * admin maintains under /admin/info-sections.
 */
class InfoSectionTest extends TestCase
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

    private function section(array $overrides = []): InfoSection
    {
        return InfoSection::create(array_merge([
            'group' => InfoSection::GROUP_HELP,
            'heading' => ['fa' => 'عنوان فارسی', 'en' => 'English heading'],
            'body' => ['fa' => 'متن فارسی', 'en' => 'English body'],
            'is_active' => true,
            'sort_order' => 10,
        ], $overrides));
    }

    // ── API ───────────────────────────────────────────────────

    public function test_sections_are_public_and_localized(): void
    {
        $this->section();

        $fa = $this->getJson('/api/v1/info/help', ['Accept-Language' => 'fa-IR,fa;q=0.9'])
            ->assertOk()->json('data.sections');
        $this->assertSame('عنوان فارسی', $fa[0]['heading']);
        $this->assertSame('متن فارسی', $fa[0]['body']);

        $en = $this->getJson('/api/v1/info/help', ['Accept-Language' => 'en-US,en;q=0.9'])
            ->assertOk()->json('data.sections');
        $this->assertSame('English heading', $en[0]['heading']);
        $this->assertSame('English body', $en[0]['body']);

        // The web client sends ?locale= because browsers drop a JS-set header.
        $viaQuery = $this->getJson('/api/v1/info/help?locale=en', ['Accept-Language' => 'fa'])
            ->assertOk()->json('data.sections');
        $this->assertSame('English heading', $viaQuery[0]['heading']);
    }

    public function test_each_group_only_returns_its_own_boxes(): void
    {
        $this->section(['group' => InfoSection::GROUP_HELP, 'heading' => ['fa' => 'راهنما']]);
        $this->section(['group' => InfoSection::GROUP_PRIVACY, 'heading' => ['fa' => 'حریم']]);
        $this->section(['group' => InfoSection::GROUP_TERMS, 'heading' => ['fa' => 'قوانین']]);

        foreach (['help' => 'راهنما', 'privacy' => 'حریم', 'terms' => 'قوانین'] as $group => $heading) {
            $response = $this->getJson("/api/v1/info/{$group}", ['Accept-Language' => 'fa'])->assertOk();
            $this->assertSame($group, $response->json('data.group'));
            $this->assertSame([$heading], array_column($response->json('data.sections'), 'heading'));
        }

        // A group with nothing in it is an empty list, not an error — the
        // client falls back to its bundled copy.
        $this->assertSame([], $this->getJson('/api/v1/info/about')->assertOk()->json('data.sections'));
    }

    public function test_unknown_group_is_not_found(): void
    {
        $this->getJson('/api/v1/info/pricing')->assertNotFound();
    }

    public function test_legacy_privacy_endpoint_still_serves_the_privacy_group(): void
    {
        $this->section(['group' => InfoSection::GROUP_PRIVACY, 'heading' => ['fa' => 'حریم خصوصی']]);
        $this->section(['group' => InfoSection::GROUP_HELP, 'heading' => ['fa' => 'راهنما']]);

        $sections = $this->getJson('/api/v1/privacy', ['Accept-Language' => 'fa'])
            ->assertOk()->json('data.sections');

        $this->assertSame(['حریم خصوصی'], array_column($sections, 'heading'));
    }

    public function test_only_active_sections_are_returned_in_sort_order(): void
    {
        $this->section(['heading' => ['fa' => 'دوم'], 'sort_order' => 20]);
        $this->section(['heading' => ['fa' => 'اول'], 'sort_order' => 10]);
        $this->section(['heading' => ['fa' => 'مخفی'], 'sort_order' => 5, 'is_active' => false]);

        $sections = $this->getJson('/api/v1/info/help', ['Accept-Language' => 'fa'])
            ->assertOk()->json('data.sections');

        $this->assertSame(['اول', 'دوم'], array_column($sections, 'heading'));
    }

    public function test_missing_english_falls_back_to_persian(): void
    {
        $this->section(['heading' => ['fa' => 'فقط فارسی', 'en' => null], 'body' => ['fa' => 'بدنه', 'en' => null]]);

        $sections = $this->getJson('/api/v1/info/help', ['Accept-Language' => 'en'])
            ->assertOk()->json('data.sections');

        $this->assertSame('فقط فارسی', $sections[0]['heading']);
    }

    public function test_contact_link_is_served_only_when_both_halves_are_present(): void
    {
        $complete = $this->section([
            'link_label' => ['fa' => 'ارسال ایمیل', 'en' => 'Email us'],
            'link_url' => 'mailto:support@ritmesalamat.com',
        ]);
        $urlOnly = $this->section(['link_url' => 'mailto:x@y.z', 'sort_order' => 20]);
        $labelOnly = $this->section(['link_label' => ['fa' => 'بدون مقصد'], 'sort_order' => 30]);

        $sections = collect($this->getJson('/api/v1/info/help?locale=fa')->assertOk()->json('data.sections'))
            ->keyBy('id');

        $this->assertSame('ارسال ایمیل', $sections[$complete->id]['link_label']);
        $this->assertSame('mailto:support@ritmesalamat.com', $sections[$complete->id]['link_url']);

        // A caption with no destination — or a destination with no caption —
        // would render an unusable button, so neither half is sent.
        $this->assertNull($sections[$urlOnly->id]['link_label']);
        $this->assertNull($sections[$urlOnly->id]['link_url']);
        $this->assertNull($sections[$labelOnly->id]['link_label']);
        $this->assertNull($sections[$labelOnly->id]['link_url']);
    }

    // ── Admin panel ───────────────────────────────────────────

    public function test_admin_can_create_edit_toggle_and_delete_a_box(): void
    {
        $this->actingAs($this->admin(), 'admin');

        $this->post('/admin/info-sections', [
            'group' => 'help',
            'heading' => ['fa' => 'باکس تازه', 'en' => 'New box'],
            'body' => ['fa' => 'محتوای باکس', 'en' => 'Box body'],
            'link_label' => ['fa' => 'تماس', 'en' => 'Contact'],
            'link_url' => 'mailto:support@ritmesalamat.com',
            'sort_order' => 30,
            'is_active' => '1',
        ])->assertRedirect(route('admin.info-sections.index', ['group' => 'help']));

        $section = InfoSection::firstOrFail();
        $this->assertSame('باکس تازه', $section->heading['fa']);
        $this->assertSame('mailto:support@ritmesalamat.com', $section->link_url);
        $this->assertTrue($section->is_active);

        $this->put("/admin/info-sections/{$section->id}", [
            'group' => 'privacy',
            'heading' => ['fa' => 'ویرایش شد'],
            'body' => ['fa' => 'محتوای جدید'],
            'sort_order' => 5,
        ])->assertRedirect(route('admin.info-sections.index', ['group' => 'privacy']));

        $section->refresh();
        $this->assertSame('ویرایش شد', $section->heading['fa']);
        // Moved to another screen, and the link was cleared out entirely.
        $this->assertSame('privacy', $section->group);
        $this->assertNull($section->link_url);
        $this->assertNull($section->link_label);
        // The checkbox was absent this time, so the box is now hidden from the app.
        $this->assertFalse($section->is_active);

        $this->post("/admin/info-sections/{$section->id}/toggle");
        $this->assertTrue($section->refresh()->is_active);

        $this->delete("/admin/info-sections/{$section->id}");
        $this->assertSame(0, InfoSection::count());
    }

    public function test_admin_index_is_filtered_by_the_selected_group(): void
    {
        $this->actingAs($this->admin(), 'admin');
        $this->section(['group' => InfoSection::GROUP_HELP, 'heading' => ['fa' => 'باکس راهنما']]);
        $this->section(['group' => InfoSection::GROUP_TERMS, 'heading' => ['fa' => 'باکس قوانین']]);

        $this->get('/admin/info-sections?group=terms')
            ->assertOk()
            ->assertSee('باکس قوانین')
            ->assertDontSee('باکس راهنما');

        // No group, or a bogus one, lands on the first tab rather than erroring.
        $this->get('/admin/info-sections')->assertOk()->assertSee('باکس راهنما');
        $this->get('/admin/info-sections?group=pricing')->assertOk()->assertSee('باکس راهنما');
    }

    public function test_admin_form_requires_persian_copy_and_a_known_group(): void
    {
        $this->actingAs($this->admin(), 'admin');

        $this->post('/admin/info-sections', ['group' => 'pricing', 'heading' => ['en' => 'no persian'], 'body' => ['en' => 'x']])
            ->assertSessionHasErrors(['group', 'heading.fa', 'body.fa']);
    }

    public function test_admin_form_rejects_a_link_that_is_not_a_usable_destination(): void
    {
        $this->actingAs($this->admin(), 'admin');

        $payload = [
            'group' => 'help',
            'heading' => ['fa' => 'پشتیبانی'],
            'body' => ['fa' => 'متن'],
            'link_label' => ['fa' => 'تماس'],
        ];

        // Free text and javascript: URLs would both end up in an href.
        $this->post('/admin/info-sections', $payload + ['link_url' => 'support@ritme.app'])
            ->assertSessionHasErrors('link_url');
        $this->post('/admin/info-sections', $payload + ['link_url' => 'javascript:alert(1)'])
            ->assertSessionHasErrors('link_url');

        $this->post('/admin/info-sections', $payload + ['link_url' => 'tel:+982112345678'])
            ->assertSessionDoesntHaveErrors();
    }

    public function test_guests_cannot_reach_the_admin_page(): void
    {
        $this->get('/admin/info-sections')->assertRedirect('/admin/login');
    }

    // ── Seeder ────────────────────────────────────────────────

    public function test_seeder_fills_every_group_and_is_idempotent(): void
    {
        $this->seed(InfoSectionSeeder::class);
        $count = InfoSection::count();
        $this->assertGreaterThan(0, $count);

        foreach (InfoSection::GROUPS as $group) {
            $this->assertGreaterThan(0, InfoSection::inGroup($group)->count(), "group {$group} was not seeded");
        }

        // The support box ships with a working contact button.
        $this->assertNotNull(InfoSection::inGroup(InfoSection::GROUP_HELP)->whereNotNull('link_url')->first());

        InfoSection::query()->update(['is_active' => false]);
        $this->seed(InfoSectionSeeder::class);

        $this->assertSame($count, InfoSection::count());
        // Re-seeding never revives boxes an admin switched off.
        $this->assertSame(0, InfoSection::active()->count());
    }
}
