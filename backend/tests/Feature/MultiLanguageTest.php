<?php

namespace Tests\Feature;

use App\Models\Admin;
use App\Models\Article;
use App\Models\Language;
use App\Models\MessageContent;
use App\Services\Language\LanguageProvisioner;
use App\Services\Language\LanguageRegistry;
use App\Services\Language\TranslationStore;
use Database\Seeders\LanguageSeeder;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\File;
use Tests\TestCase;

/**
 * The app ships whatever languages the `languages` table holds — fa and en are
 * seeded rows, not special cases. Adding one in the admin panel has to grow the
 * content forms, generate the locale's UI-string files and reach the API,
 * without a single hardcoded locale anywhere in between.
 */
class MultiLanguageTest extends TestCase
{
    use RefreshDatabase;

    private ?Admin $admin = null;

    protected function setUp(): void
    {
        parent::setUp();

        $this->seed(LanguageSeeder::class);
        app(LanguageRegistry::class)->flush();
    }

    protected function tearDown(): void
    {
        // Generated bundles live on disk, outside the transaction the database
        // is rolled back with, so they have to be swept up by hand.
        foreach (['ar', 'pt-br'] as $code) {
            File::deleteDirectory(storage_path("app/translations/{$code}"));
            File::deleteDirectory(lang_path($code));
        }

        parent::tearDown();
    }

    private function admin(): Admin
    {
        return $this->admin ??= Admin::create([
            'name' => 'Test Admin',
            'email' => 'languages@ritme.test',
            'password' => 'secret123',
            'role' => Admin::ROLE_SUPER,
            'is_active' => true,
        ]);
    }

    public function test_the_registry_starts_with_the_seeded_locales(): void
    {
        $registry = app(LanguageRegistry::class);

        $this->assertSame(['fa', 'en'], $registry->codes());
        $this->assertSame('fa', $registry->defaultCode());
        $this->assertSame('rtl', $registry->direction('fa')->value);
    }

    public function test_it_resolves_a_full_accept_language_header(): void
    {
        $registry = app(LanguageRegistry::class);

        $this->assertSame('en', $registry->resolve('en-US,en;q=0.9'));
        $this->assertSame('fa', $registry->resolve('fa-IR,fa;q=0.9,en;q=0.8'));
        // Nothing we ship — fall back rather than fail.
        $this->assertSame('fa', $registry->resolve('de-DE'));
        $this->assertSame('fa', $registry->resolve(null));
    }

    public function test_creating_a_language_generates_its_translation_files(): void
    {
        $this->actingAs($this->admin(), 'admin')
            ->post('/admin/languages', [
                'code' => 'ar',
                'name' => 'العربية',
                'english_name' => 'Arabic',
                'direction' => 'rtl',
                'is_active' => '1',
                'copy_from' => 'fa',
            ])
            ->assertRedirect(route('admin.languages.index'));

        $this->assertDatabaseHas('languages', ['code' => 'ar', 'direction' => 'rtl']);

        $store = app(TranslationStore::class);
        $namespaces = $store->namespaces();

        $this->assertNotEmpty($namespaces, 'the default locale must ship a translation seed');

        foreach ($namespaces as $namespace) {
            $this->assertFileExists($store->livePath('ar', $namespace));
        }
    }

    public function test_a_new_language_appears_in_the_public_api_with_a_complete_bundle(): void
    {
        $this->createArabic();

        $this->getJson('/api/v1/languages')
            ->assertOk()
            ->assertJsonPath('data.default', 'fa')
            ->assertJsonFragment(['code' => 'ar', 'direction' => 'rtl']);

        $response = $this->getJson('/api/v1/languages/ar/messages')->assertOk();

        $this->assertSame('ar', $response->json('data.locale'));
        $this->assertSame('rtl', $response->json('data.direction'));
        $this->assertNotEmpty($response->json('data.messages'));
    }

    public function test_an_unknown_locale_falls_back_instead_of_404ing(): void
    {
        // An old client asking for a locale that has since been removed must
        // keep rendering, in the default language.
        $this->getJson('/api/v1/languages/de/messages')
            ->assertOk()
            ->assertJsonPath('data.locale', 'fa');
    }

    public function test_untranslated_keys_fall_back_to_the_default_language(): void
    {
        $this->createArabic();

        $store = app(TranslationStore::class);
        $namespace = $store->namespaces()[0];
        $reference = $store->rawNamespace('fa', $namespace);
        $key = array_key_first($reference);

        $store->writeNamespace('ar', $namespace, [$key => 'مترجم']);

        $messages = $store->namespaceMessages('ar', $namespace);

        $this->assertSame('مترجم', $messages[$key]);
        // Every other key the default locale defines is still present.
        foreach (array_keys($reference) as $referenceKey) {
            $this->assertArrayHasKey($referenceKey, $messages);
        }
    }

    public function test_admin_content_forms_render_an_input_per_active_language(): void
    {
        $this->createArabic();

        $this->actingAs($this->admin(), 'admin')
            ->get('/admin/articles/create')
            ->assertOk()
            ->assertSee('name="title[fa]"', false)
            ->assertSee('name="title[en]"', false)
            ->assertSee('name="title[ar]"', false);
    }

    public function test_content_can_be_saved_in_a_language_added_after_launch(): void
    {
        $this->createArabic();

        $this->actingAs($this->admin(), 'admin')
            ->post('/admin/articles', [
                'slug' => 'multi-locale',
                'title' => ['fa' => 'عنوان', 'en' => 'Title', 'ar' => 'عنوان عربي'],
                'is_published' => '1',
            ])
            ->assertRedirect(route('admin.articles.index'));

        $article = Article::where('slug', 'multi-locale')->firstOrFail();

        $this->assertSame('عنوان عربي', $article->localized('title', 'ar'));
        // A locale with no translation reads the default language's text.
        $this->assertSame('عنوان', $article->localized('title', 'de'));
    }

    public function test_only_the_default_language_is_required_on_a_content_form(): void
    {
        $this->createArabic();

        // Persian filled in, the rest blank — publishable.
        $this->actingAs($this->admin(), 'admin')
            ->post('/admin/articles', [
                'slug' => 'fa-only',
                'title' => ['fa' => 'فقط فارسی', 'en' => '', 'ar' => ''],
            ])
            ->assertSessionHasNoErrors();

        // Persian missing — rejected.
        $this->actingAs($this->admin(), 'admin')
            ->post('/admin/articles', [
                'slug' => 'no-default',
                'title' => ['fa' => '', 'en' => 'English only'],
            ])
            ->assertSessionHasErrors('title.fa');
    }

    public function test_a_new_language_gets_its_own_smart_message_rows_pending_review(): void
    {
        MessageContent::create([
            'group' => 'greeting',
            'item_key' => 'morning',
            'locale' => 'fa',
            'payload' => ['text' => 'صبح بخیر'],
            'is_active' => true,
            'is_approved' => true,
        ]);

        $this->createArabic();

        $row = MessageContent::where('locale', 'ar')->where('item_key', 'morning')->first();

        $this->assertNotNull($row);
        // Still the source language's words — an admin must approve the
        // translation before users can be served it.
        $this->assertFalse($row->is_approved);
    }

    public function test_the_default_language_cannot_be_deleted(): void
    {
        $persian = Language::where('code', 'fa')->firstOrFail();

        $this->actingAs($this->admin(), 'admin')
            ->delete("/admin/languages/{$persian->id}")
            ->assertSessionHas('error');

        $this->assertDatabaseHas('languages', ['code' => 'fa']);
    }

    public function test_deleting_a_language_removes_its_generated_files(): void
    {
        $arabic = $this->createArabic();
        $store = app(TranslationStore::class);
        $namespace = $store->namespaces()[0];

        $this->assertFileExists($store->livePath('ar', $namespace));

        $this->actingAs($this->admin(), 'admin')
            ->delete("/admin/languages/{$arabic->id}")
            ->assertSessionHas('status');

        $this->assertDatabaseMissing('languages', ['code' => 'ar']);
        $this->assertFileDoesNotExist($store->livePath('ar', $namespace));
    }

    public function test_marking_a_language_default_demotes_the_previous_one(): void
    {
        $arabic = $this->createArabic();

        $this->actingAs($this->admin(), 'admin')
            ->put("/admin/languages/{$arabic->id}", [
                'code' => 'ar',
                'name' => 'العربية',
                'english_name' => 'Arabic',
                'direction' => 'rtl',
                'is_active' => '1',
                'is_default' => '1',
            ])
            ->assertRedirect(route('admin.languages.index'));

        $this->assertDatabaseHas('languages', ['code' => 'ar', 'is_default' => true]);
        $this->assertDatabaseHas('languages', ['code' => 'fa', 'is_default' => false]);
        $this->assertSame('ar', app(LanguageRegistry::class)->defaultCode());
    }

    public function test_a_language_code_must_look_like_a_locale(): void
    {
        $this->actingAs($this->admin(), 'admin')
            ->post('/admin/languages', [
                'code' => 'not a locale!',
                'name' => 'Nope',
                'english_name' => 'Nope',
                'direction' => 'ltr',
            ])
            ->assertSessionHasErrors('code');
    }

    public function test_the_translations_editor_saves_and_drops_blank_values(): void
    {
        $arabic = $this->createArabic();
        $store = app(TranslationStore::class);
        $namespace = $store->namespaces()[0];
        $keys = array_keys($store->flatten($store->rawNamespace('fa', $namespace)));

        $this->actingAs($this->admin(), 'admin')
            ->put("/admin/languages/{$arabic->id}/translations?namespace={$namespace}", [
                'rows' => [
                    ['key' => $keys[0], 'value' => 'مترجم'],
                    ['key' => $keys[1] ?? 'unused', 'value' => ''],
                ],
            ])
            ->assertRedirect();

        $stored = $store->rawNamespace('ar', $namespace);

        $this->assertSame('مترجم', data_get($stored, $keys[0]));
        // A blank value is dropped so the key inherits the default language
        // rather than rendering empty.
        if (isset($keys[1])) {
            $this->assertNull(data_get($stored, $keys[1]));
        }
    }

    private function createArabic(): Language
    {
        $language = Language::create([
            'code' => 'ar',
            'name' => 'العربية',
            'english_name' => 'Arabic',
            'direction' => 'rtl',
            'is_active' => true,
            'is_default' => false,
            'sort_order' => 20,
        ]);

        app(LanguageRegistry::class)->flush();
        app(LanguageProvisioner::class)->provision($language, 'fa');

        return $language;
    }
}
