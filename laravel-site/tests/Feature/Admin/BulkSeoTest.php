<?php

declare(strict_types=1);

use App\Domain\Blog\Models\Post;
use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Actions\BulkSaveSeoMeta;
use App\Domain\Seo\Actions\ExportBulkSeoCsv;
use App\Domain\Seo\Actions\ImportBulkSeoCsv;
use App\Domain\Seo\Actions\ListBulkSeoRows;
use App\Domain\Seo\Actions\ListStaticPageSeo;
use App\Domain\Seo\Actions\ResolveSeoTarget;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Enums\SettingGroup;
use App\Domain\Shop\Catalog\Models\Product;
use App\Filament\Auth\AdminRole;
use App\Filament\Pages\Seo\BulkEditor;
use App\Filament\Pages\Settings\OrganizationSettings;
use App\Filament\Pages\Settings\SeoSettings;
use App\Models\User;
use App\Support\Cache\NamespaceVersions;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Facades\Filament;
use Livewire\Livewire;
use Spatie\Activitylog\Models\Activity;
use Tests\Feature\Admin\AdminMfa;

function bulkSeoUser(?AdminRole $role = AdminRole::SeoManager): User
{
    $user = User::factory()->create();
    if ($role !== null) {
        $user->assignRole($role->value);
    }

    return AdminMfa::enrol($user); // PII roles must have MFA (L9-04b)
}

/**
 * @return array<string, mixed>
 */
function bulkSeoRow(string $key): array
{
    foreach (app(ListBulkSeoRows::class)->handle() as $row) {
        if ($row['key'] === $key) {
            return $row;
        }
    }

    throw new RuntimeException("row {$key} not listed");
}

/**
 * @return array{seo: int, sitemap: int, pages: int}
 */
function bulkSeoVersions(): array
{
    $versions = app(NamespaceVersions::class);

    return ['seo' => $versions->version('seo'), 'sitemap' => $versions->version('sitemap'), 'pages' => $versions->version('pages')];
}

function bulkSeoTitle(string $html): string
{
    preg_match('#<title>(.*?)</title>#s', $html, $m);

    return html_entity_decode($m[1] ?? '', ENT_QUOTES | ENT_HTML5, 'UTF-8');
}

beforeEach(function (): void {
    $this->seed([SettingsSeeder::class, AdminRolesSeeder::class]);
    Filament::setCurrentPanel('admin');
});

it('lets SEO managers in and denies every other role', function (): void {
    $urls = [BulkEditor::getUrl(), SeoSettings::getUrl(), OrganizationSettings::getUrl()];

    foreach ($urls as $url) {
        $this->get($url)->assertRedirect('/admin/login');
    }

    $manager = bulkSeoUser();
    foreach ($urls as $url) {
        $this->actingAs($manager)->get($url)->assertOk();
    }

    foreach ([AdminRole::Editor, AdminRole::Support, AdminRole::ShopManager, AdminRole::DirectoryManager, null] as $role) {
        $user = bulkSeoUser($role);
        foreach ($urls as $url) {
            $this->actingAs($user)->get($url)->assertForbidden();
        }
    }

    $this->actingAs(bulkSeoUser(AdminRole::Editor));
    Livewire::test(BulkEditor::class)->assertForbidden();
});

it('lists posts, products, places, categories and static pages with their effective head and checks', function (): void {
    $post = Post::factory()->create(['title' => 'راهنمای خواب کودک در ماه‌های اول']);
    $product = Product::factory()->create(['title' => 'بالش شیردهی']);
    $twin = Product::factory()->create(['title' => 'بالش بارداری']);
    SeoMeta::query()->create(['seoable_type' => 'shop_product', 'seoable_id' => $product->id, 'title' => 'عنوان مشترک', 'robots' => 'noindex,follow']);
    SeoMeta::query()->create(['seoable_type' => 'shop_product', 'seoable_id' => $twin->id, 'title' => 'عنوان تکراری محصول']);
    $third = Product::factory()->create();
    SeoMeta::query()->create(['seoable_type' => 'shop_product', 'seoable_id' => $third->id, 'title' => 'عنوان تکراری محصول']);

    $rows = collect(app(ListBulkSeoRows::class)->handle())->keyBy('key');

    expect($rows->pluck('type')->unique()->values()->all())->toContain('blog_post', 'shop_product', ResolveSeoTarget::PAGE)
        ->and($rows->where('type', ResolveSeoTarget::PAGE)->count())->toBe(count(app(ListStaticPageSeo::class)->handle()));

    $postRow = $rows->get('blog_post-'.$post->id);
    expect($postRow['effective_title'])->toBe('راهنمای خواب کودک در ماه‌های اول — ریتمی')
        ->and($postRow['effective_description'])->toBe('خلاصه کوتاه برای فهرست مقاله‌ها.')
        ->and($postRow['missing'])->toBeTrue()
        ->and($postRow['needs_work'])->toBeTrue() // not scored yet
        ->and($postRow['published'])->toBeFalse();

    // Duplicates among indexable rows only; the noindex product is excluded from the comparison.
    expect($rows->get('shop_product-'.$twin->id)['duplicate_title'])->toBeTrue()
        ->and($rows->get('shop_product-'.$third->id)['duplicate_title'])->toBeTrue()
        ->and($rows->get('shop_product-'.$product->id)['indexable'])->toBeFalse()
        ->and($rows->get('shop_product-'.$product->id)['duplicate_title'])->toBeFalse();

    // A static page is matched by its own route name: never a duplicate of itself.
    expect($rows->get(ResolveSeoTarget::pageKey(StaticPage::About))['duplicate_title'])->toBeFalse();

    $this->actingAs(bulkSeoUser());
    Livewire::test(BulkEditor::class)
        ->assertSuccessful()
        ->assertSee('بالش بارداری')
        ->filterTable('issue', 'duplicate')
        ->assertSee('بالش بارداری')
        ->assertDontSee('راهنمای خواب کودک در ماه‌های اول')
        ->resetTableFilters()
        ->filterTable('type', 'blog_post')
        ->assertSee('راهنمای خواب کودک در ماه‌های اول')
        ->assertDontSee('بالش بارداری');
});

it('saves 50 inline edits in one action, bumps the caches once, scores them and logs it', function (): void {
    $products = Product::factory()->count(50)->create();
    $manager = bulkSeoUser();
    $this->actingAs($manager);

    $component = Livewire::test(BulkEditor::class);
    foreach ($products as $i => $product) {
        $component->set("drafts.shop_product-{$product->id}.title", "عنوان سئوی تازه محصول شماره {$i}")
            ->set("drafts.shop_product-{$product->id}.description", "توضیح متای تازه برای محصول شماره {$i} که برای آزمون ویرایش گروهی نوشته شده است.");
    }

    $before = bulkSeoVersions();
    $component->call('saveDrafts')->assertNotified()->assertSet('drafts', []);
    $after = bulkSeoVersions();

    expect($after['seo'] - $before['seo'])->toBe(1)
        ->and($after['sitemap'] - $before['sitemap'])->toBe(1)
        ->and($after['pages'] - $before['pages'])->toBe(1)
        ->and(SeoMeta::query()->where('seoable_type', 'shop_product')->whereNotNull('title')->count())->toBe(50)
        ->and(SeoMeta::query()->where('seoable_type', 'shop_product')->whereNull('score')->count())->toBe(0);

    $activity = Activity::query()->where('log_name', 'seo')->where('description', 'seo.bulk.updated')->latest('id')->first();
    expect($activity?->causer_id)->toBe($manager->id)
        ->and($activity?->properties->get('changes'))->toHaveCount(50)
        ->and(bulkSeoRow('shop_product-'.$products[0]->id)['effective_title'])->toBe('عنوان سئوی تازه محصول شماره 0 — ریتمی');

    // Saving the same values again writes nothing and bumps nothing.
    $before = bulkSeoVersions();
    app(BulkSaveSeoMeta::class)->handle(['shop_product-'.$products[0]->id => ['title' => 'عنوان سئوی تازه محصول شماره 0']]);
    expect(bulkSeoVersions())->toBe($before);
});

it('changes the live <title> of a static page from the bulk editor', function (): void {
    $this->get('/about')->assertOk();
    $this->actingAs(bulkSeoUser());

    Livewire::test(BulkEditor::class)
        ->set('drafts.'.ResolveSeoTarget::pageKey(StaticPage::About).'.title', 'درباره تیم ریتمی و داستان ساخت اپ')
        ->call('saveDrafts');

    auth()->logout();
    expect(bulkSeoTitle((string) $this->get('/about')->assertOk()->getContent()))->toBe('درباره تیم ریتمی و داستان ساخت اپ — ریتمی');
});

it('noindexes, re-indexes and resets selected rows with bulk actions', function (): void {
    $a = Product::factory()->create();
    $b = Product::factory()->create();
    SeoMeta::query()->create(['seoable_type' => 'shop_product', 'seoable_id' => $b->id, 'title' => 'عنوان اختصاصی محصول دوم']);
    $keys = ['shop_product-'.$a->id, 'shop_product-'.$b->id];
    $this->actingAs(bulkSeoUser());

    $before = bulkSeoVersions();
    Livewire::test(BulkEditor::class)->callTableBulkAction('noindex', $keys)->assertNotified();
    expect(bulkSeoVersions()['seo'] - $before['seo'])->toBe(1)
        ->and(SeoMeta::query()->where('seoable_type', 'shop_product')->pluck('robots')->unique()->all())->toBe(['noindex,follow'])
        ->and(bulkSeoRow($keys[0])['indexable'])->toBeFalse();

    Livewire::test(BulkEditor::class)->callTableBulkAction('index', $keys);
    expect(SeoMeta::query()->where('seoable_type', 'shop_product')->whereNotNull('robots')->count())->toBe(0);

    Livewire::test(BulkEditor::class)->callTableBulkAction('reset', [$keys[1]]);
    expect(SeoMeta::query()->where('seoable_type', 'shop_product')->where('seoable_id', $b->id)->value('title'))->toBeNull()
        ->and(Activity::query()->where('description', 'seo.bulk.reset')->exists())->toBeTrue()
        ->and(Activity::query()->where('description', 'seo.bulk.noindex')->exists())->toBeTrue();
});

it('exports and re-imports the bulk CSV in one batch', function (): void {
    $product = Product::factory()->create();
    $key = 'shop_product-'.$product->id;
    SeoMeta::query()->create(['seoable_type' => 'shop_product', 'seoable_id' => $product->id, 'title' => '=HYPERLINK("x")']);

    $out = fopen('php://memory', 'w+b');
    app(ExportBulkSeoCsv::class)->handle($out);
    rewind($out);
    $csv = (string) stream_get_contents($out);
    expect($csv)->toContain($key)->toContain("'=HYPERLINK");

    $path = tempnam(sys_get_temp_dir(), 'seo');
    file_put_contents($path, "key,title,robots\n{$key},عنوان درون‌ریزی شده محصول,noindex\nblog_post-999999,x,\nnope,x,\n");
    $before = bulkSeoVersions();
    $result = app(ImportBulkSeoCsv::class)->handle($path);
    unlink($path);

    expect($result['changed'])->toHaveKey($key)
        ->and($result['skipped'])->toBe(2)
        ->and($result['errors'])->toHaveCount(2)
        ->and(bulkSeoVersions()['seo'] - $before['seo'])->toBe(1);
    $meta = SeoMeta::query()->where('seoable_type', 'shop_product')->where('seoable_id', $product->id)->firstOrFail();
    expect($meta->title)->toBe('عنوان درون‌ریزی شده محصول')->and($meta->robots)->toBe('noindex,follow');
});

it('saves SEO defaults that flow into the head', function (): void {
    $this->actingAs(bulkSeoUser());

    Livewire::test(SeoSettings::class)
        ->fillForm(['twitter_handle' => 'not valid!'])
        ->call('save')
        ->assertHasFormErrors(['twitter_handle']);

    // `%s` alone is valid; "%sep%" alone is not.
    Livewire::test(SeoSettings::class)
        ->fillForm(['title_template' => '%sep% ریتمی'])
        ->call('save')
        ->assertHasFormErrors(['title_template']);

    Livewire::test(SeoSettings::class)
        ->fillForm([
            'title_template' => '%s %sep% ریتمی',
            'separator' => '|',
            'default_description' => 'توضیح پیش‌فرض تازه سایت ریتمی برای صفحه‌هایی که توضیح خودشان را ندارند و باید کوتاه باشد.',
            'twitter_handle' => '@ritme_app',
        ])
        ->call('save')
        ->assertHasNoFormErrors()
        ->assertNotified();

    $seo = app(SettingsRepository::class)->all()->seo;
    expect($seo->twitterHandle)->toBe('ritme_app')
        ->and($seo->title('چرخه'))->toBe('چرخه | ریتمی')
        ->and($seo->indexing->indexNowEnabled)->toBeFalse(); // indexing keys of the same group untouched

    SeoMeta::query()->create(['route_name' => 'about', 'title' => 'درباره تیم ریتمی']);
    auth()->logout();
    $html = (string) $this->get('/about')->assertOk()->getContent();
    expect(bulkSeoTitle($html))->toBe('درباره تیم ریتمی | ریتمی')
        ->and($html)->toContain('<meta name="twitter:site" content="@ritme_app"');

    expect(Activity::query()->where('log_name', 'settings')->latest('id')->first()?->properties->get('group'))->toBe('seo');
});

it('saves organization settings into the JSON-LD graph and validates sameAs URLs', function (): void {
    $this->actingAs(bulkSeoUser());

    foreach (['ftp://example.com/x', 'javascript:alert(1)', 'not a url'] as $bad) {
        Livewire::test(OrganizationSettings::class)
            ->fillForm(['same_as' => ['https://example.com/ok', $bad]])
            ->call('save')
            ->assertHasFormErrors();
    }

    Livewire::test(OrganizationSettings::class)
        ->fillForm(['founding_date' => '2024-13'])
        ->call('save')
        ->assertHasFormErrors(['founding_date']);

    Livewire::test(OrganizationSettings::class)
        ->fillForm([
            'legal_name' => 'شرکت ریتم سلامت پارسیان',
            'founding_date' => '2024-03-21',
            'same_as' => ['https://www.aparat.com/ritme', 'http://example.org/ritme'],
            'contact_point' => ['contact_type' => 'customer support', 'telephone' => '+982112345678', 'email' => 'support@ritmeapp.ir'],
        ])
        ->call('save')
        ->assertHasNoFormErrors();

    $organization = app(SettingsRepository::class)->group(SettingGroup::Organization);
    expect($organization->toArray()['same_as'])->toBe(['https://www.aparat.com/ritme', 'http://example.org/ritme']);

    auth()->logout();
    preg_match('#<script type="application/ld\+json">(.*?)</script>#s', (string) $this->get('/about')->assertOk()->getContent(), $m);
    $graph = json_decode($m[1] ?? '{}', true, flags: JSON_THROW_ON_ERROR);
    $node = collect($graph['@graph'] ?? [])->firstWhere('@type', 'Organization');

    expect($node['legalName'] ?? null)->toBe('شرکت ریتم سلامت پارسیان')
        ->and($node['foundingDate'] ?? null)->toBe('2024-03-21')
        ->and($node['sameAs'] ?? [])->toContain('https://www.aparat.com/ritme', 'http://example.org/ritme')
        ->and($node['contactPoint']['telephone'] ?? null)->toBe('+982112345678')
        ->and($node['contactPoint']['email'] ?? null)->toBe('support@ritmeapp.ir');
});
