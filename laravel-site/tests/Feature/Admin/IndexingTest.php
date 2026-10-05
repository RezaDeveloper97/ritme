<?php

declare(strict_types=1);

use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;
use App\Domain\Seo\Indexing\Actions\RegenerateSitemaps;
use App\Domain\Seo\Indexing\Actions\SaveIndexingSettings;
use App\Domain\Seo\Indexing\HeadCode;
use App\Domain\Seo\Indexing\IndexNow\IndexNow;
use App\Domain\Seo\Indexing\InvalidIndexingSettings;
use App\Domain\Seo\Indexing\Jobs\PingSitemaps;
use App\Domain\Seo\Indexing\Jobs\SubmitToIndexNow;
use App\Domain\Seo\Indexing\RobotsTxtValidator;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Enums\SettingGroup;
use App\Filament\Auth\AdminRole;
use App\Filament\Pages\Seo\IndexingSettingsPage;
use App\Models\User;
use App\Support\Cache\NamespaceVersions;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Actions\Testing\TestAction;
use Filament\Facades\Filament;
use Illuminate\Http\Client\Request as HttpRequest;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\Bus;
use Illuminate\Support\Facades\Http;
use Livewire\Livewire;
use Spatie\Activitylog\Models\Activity;

function indexingAdmin(?AdminRole $role = AdminRole::SeoManager): User
{
    $user = User::factory()->create();
    if ($role !== null) {
        $user->assignRole($role->value);
    }

    return $user;
}

/** Stores indexing settings straight through the action (as a super-admin would). */
function indexingSettings(array $values): void
{
    app(SaveIndexingSettings::class)->handle($values, true);
}

beforeEach(function (): void {
    $this->seed([SettingsSeeder::class, AdminRolesSeeder::class]);
    Filament::setCurrentPanel('admin');
    config(['app.url' => 'https://ritme.ir']);
    Http::preventStrayRequests(); // IndexNow / ping never leave the test process
});

afterEach(fn () => Carbon::setTestNow());

it('lets SEO managers in and denies other roles', function (): void {
    $url = IndexingSettingsPage::getUrl();

    $this->get($url)->assertRedirect('/admin/login');
    $this->actingAs(indexingAdmin())->get($url)->assertOk()->assertSee('robots.txt');

    foreach ([AdminRole::Editor, AdminRole::Support, AdminRole::ShopManager, AdminRole::DirectoryManager, null] as $role) {
        $this->actingAs(indexingAdmin($role))->get($url)->assertForbidden();
    }

    $inactive = indexingAdmin();
    $inactive->forceFill(['is_active' => false])->save();
    $this->actingAs($inactive);
    expect(IndexingSettingsPage::canAccess())->toBeFalse();
});

it('reflects a robots.txt change at /robots.txt in production and logs it', function (): void {
    config(['app.env' => 'production']);
    $before = (string) $this->get('/robots.txt')->getContent(); // warm the cached document
    expect($before)->toContain('Disallow: /admin');

    $this->actingAs($manager = indexingAdmin());
    Livewire::test(IndexingSettingsPage::class)
        ->fillForm(['robots_txt' => "User-agent: *\nDisallow: /private/\nAllow: /private/open\n\nUser-agent: BadBot\nDisallow: /"])
        ->call('save')
        ->assertHasNoFormErrors()
        ->assertNotified();

    $body = (string) $this->get('/robots.txt')->assertOk()->getContent();
    expect($body)->toBe("User-agent: *\nDisallow: /private/\nAllow: /private/open\n\nUser-agent: BadBot\nDisallow: /\n\nSitemap: https://ritme.ir/sitemap.xml\n");

    $activity = Activity::query()->where('log_name', 'seo')->where('description', 'seo.indexing.updated')->latest('id')->first();
    expect($activity?->causer_id)->toBe($manager->id)
        ->and($activity?->properties->get('attributes'))->toHaveKey('robots_txt');
});

it('keeps non-production robots.txt at Disallow: / whatever the rules', function (): void {
    indexingSettings(['robots_txt' => "User-agent: *\nDisallow: /private/"]);

    expect((string) $this->get('/robots.txt')->getContent())->toBe("User-agent: *\nDisallow: /\n");
});

it('rejects invalid robots rules with Persian errors and keeps the stored rules', function (string $rules): void {
    $this->actingAs(indexingAdmin());

    Livewire::test(IndexingSettingsPage::class)
        ->fillForm(['robots_txt' => $rules])
        ->call('save')
        ->assertHasFormErrors(['robots_txt']);

    expect(app(SettingsRepository::class)->all()->seo->indexing->robotsTxt)->toBeNull();
})->with([
    'whole site blocked' => ["User-agent: *\nDisallow: /"],
    'unknown directive' => ["User-agent: *\nNoindex: /x"],
    'rule before agent' => ['Disallow: /x'],
    'sitemap line' => ["User-agent: *\nSitemap: https://evil.test/s.xml"],
    'bad path' => ["User-agent: *\nDisallow: private"],
]);

it('validates robots syntax line by line', function (): void {
    $validator = new RobotsTxtValidator;

    expect($validator->errors("# comment\nUser-agent: Googlebot\nUser-agent: Bingbot\nDisallow: /tmp/ # inline\nCrawl-delay: 2\n\nUser-agent: *\nAllow: /\nDisallow: /*?q="))->toBe([])
        ->and($validator->errors("User-agent: *\nCrawl-delay: abc"))->toHaveCount(1)
        ->and($validator->errors("User-agent: *\nDisallow /x")[0])->toContain('خط ۲');
});

it('resets robots.txt through the reset action', function (): void {
    config(['app.env' => 'production']);
    indexingSettings(['robots_txt' => "User-agent: *\nDisallow: /private/"]);
    expect((string) $this->get('/robots.txt')->getContent())->toContain('/private/');

    $this->actingAs(indexingAdmin());
    Livewire::test(IndexingSettingsPage::class)
        ->callAction(TestAction::make('resetRobots')->schemaComponent('robotsActions', 'form'))
        ->assertNotified();

    $body = (string) $this->get('/robots.txt')->getContent();
    expect($body)->toContain('Disallow: /admin')->not->toContain('/private/')
        ->and(Activity::query()->where('description', 'seo.robots.reset')->exists())->toBeTrue();
});

it('applies per-type robots defaults to pages and drops noindex types from the sitemap', function (): void {
    config(['app.env' => 'production']);
    $category = Category::factory()->create(['slug' => 'cycle']);
    Post::factory()->published(Carbon::parse('2026-09-20 09:00:00'))->create(['slug' => 'period-pain', 'category_id' => $category->id]);

    expect((string) $this->get('/sitemap.xml')->getContent())->toContain('/sitemaps/blog-categories.xml')
        ->and((string) $this->get('/blog/category/cycle')->assertOk()->getContent())->toContain('<meta name="robots" content="index,follow');

    indexingSettings(['robots_types' => ['blog-categories' => 'noindex,follow']]);

    expect((string) $this->get('/blog/category/cycle')->assertOk()->getContent())->toContain('<meta name="robots" content="noindex,follow">')
        ->and((string) $this->get('/blog/period-pain')->getContent())->toContain('<meta name="robots" content="index,follow')
        ->and((string) $this->get('/sitemap.xml')->getContent())->not->toContain('blog-categories');
    $this->get('/sitemaps/blog-categories.xml')->assertNotFound();
});

it('excludes sitemap types and overrides priority and changefreq', function (): void {
    $category = Category::factory()->create(['slug' => 'cycle']);
    Post::factory()->published(Carbon::parse('2026-09-20 09:00:00'))->create(['slug' => 'period-pain', 'category_id' => $category->id]);

    indexingSettings([
        'sitemap_exclude' => ['blog-categories'],
        'sitemap_priorities' => ['posts' => '0.9'],
        'sitemap_changefreq' => ['posts' => 'daily'],
    ]);

    $index = (string) $this->get('/sitemap.xml')->getContent();
    $posts = (string) $this->get('/sitemaps/posts.xml')->assertOk()->getContent();

    expect($index)->not->toContain('blog-categories')->toContain('/sitemaps/posts.xml')
        ->and($posts)->toContain('<priority>0.9</priority>', '<changefreq>daily</changefreq>');
    $this->get('/sitemaps/blog-categories.xml')->assertNotFound();
});

it('regenerates the sitemap: bumps the cache, logs, pings only in production', function (): void {
    Bus::fake([PingSitemaps::class]);
    indexingSettings(['sitemap_ping_urls' => ['https://ping.example/ping?sitemap={sitemap}']]);
    $versions = app(NamespaceVersions::class);
    $before = $versions->version('sitemap');

    $this->actingAs(indexingAdmin());
    Livewire::test(IndexingSettingsPage::class)
        ->callAction(TestAction::make('regenerateSitemap')->schemaComponent('sitemapActions', 'form'))
        ->assertNotified();

    expect($versions->version('sitemap'))->toBeGreaterThan($before)
        ->and(Activity::query()->where('description', 'seo.sitemap.regenerated')->exists())->toBeTrue();
    Bus::assertNotDispatched(PingSitemaps::class); // testing env

    config(['app.env' => 'production']);
    app(RegenerateSitemaps::class)->handle();
    Bus::assertDispatched(PingSitemaps::class);
});

it('pings listed endpoints with the encoded sitemap URL from the job (production only)', function (): void {
    Http::fake(['ping.example/*' => Http::response('ok')]);
    indexingSettings(['sitemap_ping_urls' => ['https://ping.example/ping?sitemap={sitemap}']]);

    app()->call([new PingSitemaps, 'handle']);
    Http::assertNothingSent();

    config(['app.env' => 'production']);
    app()->call([new PingSitemaps, 'handle']);
    Http::assertSent(fn (HttpRequest $request): bool => $request->url() === 'https://ping.example/ping?sitemap='.rawurlencode('https://ritme.ir/sitemap.xml'));
});

it('rejects non-https ping URLs', function (): void {
    expect(fn () => indexingSettings(['sitemap_ping_urls' => ['http://ping.example/?s={sitemap}']]))
        ->toThrow(InvalidIndexingSettings::class);
});

it('renders verification codes as meta tags and keeps other stored codes', function (): void {
    indexingSettings(['verification' => ['pinterest' => 'pin123']]);

    $this->actingAs(indexingAdmin());
    Livewire::test(IndexingSettingsPage::class)
        ->fillForm(['verification' => ['google' => 'g-Code_1', 'bing' => 'B1NG', 'yandex' => 'ya42']])
        ->call('save')
        ->assertHasNoFormErrors();

    expect(app(SettingsRepository::class)->get(SettingGroup::Seo, 'verification'))
        ->toBe(['pinterest' => 'pin123', 'google' => 'g-Code_1', 'bing' => 'B1NG', 'yandex' => 'ya42']);

    auth()->logout();
    $html = (string) $this->get('/')->getContent();
    expect($html)->toContain('<meta name="google-site-verification" content="g-Code_1">', '<meta name="msvalidate.01" content="B1NG">', '<meta name="yandex-verification" content="ya42">');
});

it('rejects HTML in verification codes', function (): void {
    $this->actingAs(indexingAdmin());

    Livewire::test(IndexingSettingsPage::class)
        ->fillForm(['verification' => ['google' => '<meta name="x">']])
        ->call('save')
        ->assertHasFormErrors(['verification.google']);
});

it('keeps IndexNow off by default: no job, no key file', function (): void {
    Bus::fake([SubmitToIndexNow::class]);
    config(['app.env' => 'production']);

    Post::factory()->published(Carbon::parse('2026-09-20 09:00:00'))->create(['slug' => 'period-pain', 'category_id' => Category::factory()->create()->id]);

    Bus::assertNotDispatched(SubmitToIndexNow::class);
    expect(app(SettingsRepository::class)->all()->seo->indexing->indexNowEnabled)->toBeFalse();
    $this->get('/0123456789abcdef0123456789abcdef.txt')->assertNotFound();
});

it('creates a key on enable, serves the key file and queues a submission on publish in production', function (): void {
    Bus::fake([SubmitToIndexNow::class]);

    $this->actingAs(indexingAdmin());
    Livewire::test(IndexingSettingsPage::class)
        ->fillForm(['indexnow_enabled' => true])
        ->call('save')
        ->assertHasNoFormErrors();

    $key = app(SettingsRepository::class)->all()->seo->indexing->indexNowKey;
    expect($key)->toMatch('/^[a-f0-9]{32}$/');

    $this->get("/{$key}.txt")->assertOk()->assertHeader('Content-Type', 'text/plain; charset=UTF-8')->assertContent($key);
    $this->get('/ffffffffffffffffffffffffffffffff.txt')->assertNotFound();

    // Testing env: switched on, but nothing goes out outside production.
    $category = Category::factory()->create();
    Post::factory()->published(Carbon::parse('2026-09-20 09:00:00'))->create(['slug' => 'draft-env', 'category_id' => $category->id]);
    Bus::assertNotDispatched(SubmitToIndexNow::class);

    config(['app.env' => 'production']);
    $post = Post::factory()->published(Carbon::parse('2026-09-20 09:00:00'))->create(['slug' => 'period-pain', 'category_id' => $category->id]);
    Bus::assertDispatched(SubmitToIndexNow::class, fn (SubmitToIndexNow $job): bool => $job->urls === ['https://ritme.ir/blog/period-pain']);

    // Slug change: both the new and the old URL; a draft never.
    $post->update(['slug' => 'cramps']);
    Bus::assertDispatched(SubmitToIndexNow::class, fn (SubmitToIndexNow $job): bool => $job->urls === ['https://ritme.ir/blog/cramps', 'https://ritme.ir/blog/period-pain']);

    Bus::fake([SubmitToIndexNow::class]);
    Post::factory()->create(['slug' => 'just-a-draft', 'category_id' => $category->id]);
    Bus::assertNotDispatched(SubmitToIndexNow::class);
});

it('posts the IndexNow payload from the job with a faked HTTP client', function (): void {
    Http::fake(['api.indexnow.org/*' => Http::response('', 202)]);
    indexingSettings(['indexnow_enabled' => true]);
    $key = app(SettingsRepository::class)->all()->seo->indexing->indexNowKey;

    (new SubmitToIndexNow(['https://ritme.ir/blog/a']))->handle(app(IndexNow::class));
    Http::assertNothingSent(); // not production

    config(['app.env' => 'production']);
    (new SubmitToIndexNow(['http://www.ritme.ir/blog/a?utm_source=x', 'https://other.example/x']))->handle(app(IndexNow::class));

    Http::assertSent(function (HttpRequest $request) use ($key): bool {
        return $request->method() === 'POST'
            && $request->url() === IndexNow::ENDPOINT
            && $request->data() === [
                'host' => 'ritme.ir',
                'key' => $key,
                'keyLocation' => "https://ritme.ir/{$key}.txt",
                'urlList' => ['https://ritme.ir/blog/a'],
            ];
    });
    Http::assertSentCount(1);
});

it('retries IndexNow on 429/5xx and drops 4xx', function (): void {
    config(['app.env' => 'production']);
    indexingSettings(['indexnow_enabled' => true]);

    Http::fake(['api.indexnow.org/*' => Http::sequence()->push('', 422)->push('', 503)]);
    expect(app(IndexNow::class)->submit(['https://ritme.ir/blog/a']))->toBe(0);
    expect(fn () => app(IndexNow::class)->submit(['https://ritme.ir/blog/a']))->toThrow(RuntimeException::class);
});

it('hides head code from SEO managers and ignores it server side', function (): void {
    $this->actingAs(indexingAdmin());

    Livewire::test(IndexingSettingsPage::class)
        ->assertDontSee('تگ‌های اضافه head')
        ->set('data.head_code', '<meta name="x" content="y">')
        ->call('save');

    expect(app(SettingsRepository::class)->all()->seo->indexing->headCode)->toBeNull()
        ->and(fn () => app(SaveIndexingSettings::class)->handle(['head_code' => '<meta name="x" content="y">'], false))->not->toThrow(Throwable::class)
        ->and(app(SettingsRepository::class)->all()->seo->indexing->headCode)->toBeNull();
});

it('lets super-admins add same-origin meta/link tags but never scripts or external URLs', function (): void {
    $super = indexingAdmin(AdminRole::SuperAdmin);
    $this->actingAs($super);
    expect(IndexingSettingsPage::canEditHeadCode())->toBeTrue();

    foreach ([
        '<script src="https://cdn.example/a.js"></script>',
        '<script>alert(1)</script>',
        '<style>body{}</style>',
        '<link rel="stylesheet" href="/x.css">',
        '<link rel="preconnect" href="https://fonts.example">',
        '<meta http-equiv="refresh" content="0;url=https://evil.test">',
        '<meta name="x" content="y" onload="alert(1)">',
        '<iframe src="/x"></iframe>',
        'plain text',
    ] as $code) {
        expect(app(HeadCode::class)->errors($code))->not->toBe([], $code);
        expect(fn () => indexingSettings(['head_code' => $code]))->toThrow(InvalidIndexingSettings::class);
    }

    indexingSettings(['head_code' => "<meta name=\"p:domain_verify\" content=\"abc\">\n<link rel=\"author\" href=\"/humans.txt\">"]);

    $html = (string) $this->get('/')->getContent();
    expect($html)->toContain('<meta name="p:domain_verify" content="abc">', '<link rel="author" href="/humans.txt">')
        ->and($html)->not->toContain('<script src');

    // Even a value written around the validator is rebuilt from the allow-list on render.
    app(SettingsRepository::class)->put(SettingGroup::Seo, ['head_code' => '<meta name="ok" content="1"><script>alert(1)</script>']);
    $html = (string) $this->get('/')->getContent();
    expect($html)->toContain('<meta name="ok" content="1">')->not->toContain('alert(1)');
});
