<?php

declare(strict_types=1);

use App\Domain\Seo\Audit\Actions\QueueSeoAudit;
use App\Domain\Seo\Audit\Actions\RunSeoAudit;
use App\Domain\Seo\Audit\AuditedPage;
use App\Domain\Seo\Audit\AuditOptions;
use App\Domain\Seo\Audit\AuditResult;
use App\Domain\Seo\Audit\Enums\RunStatus;
use App\Domain\Seo\Audit\Enums\RunTrigger;
use App\Domain\Seo\Audit\Jobs\RunSeoAuditJob;
use App\Domain\Seo\Audit\Models\AuditRun;
use App\Domain\Seo\Audit\Models\AuditRunIssue;
use App\Domain\Seo\Audit\Models\AuditRunPage;
use App\Domain\Seo\Audit\Queries\AuditHistory;
use App\Domain\Seo\Audit\Severity;
use App\Domain\Seo\Redirects\Enums\AgentClass;
use App\Domain\Seo\Redirects\Models\NotFoundLog;
use App\Domain\Seo\Sitemap\SitemapEntryData;
use App\Domain\Seo\Sitemap\SitemapProvider;
use App\Domain\Seo\Sitemap\SitemapRegistry;
use App\Filament\Auth\AdminRole;
use App\Filament\Pages\Seo\AuditReport;
use App\Filament\Widgets\Seo\SeoContentNeedingWork;
use App\Filament\Widgets\Seo\SeoHealthOverview;
use App\Filament\Widgets\Seo\SeoScoreTrend;
use App\Filament\Widgets\Seo\SeoTopIssues;
use App\Filament\Widgets\Seo\SeoTopNotFound;
use App\Filament\Widgets\Seo\SeoZeroResultSearches;
use App\Models\User;
use Carbon\CarbonImmutable;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\DatabaseSeeder;
use Filament\Facades\Filament;
use Illuminate\Console\Scheduling\Schedule;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Queue;
use Illuminate\Support\Facades\Route;
use Livewire\Livewire;
use Tests\Feature\Admin\AdminMfa;
use Tests\Feature\Seo\SeoFixtures;

/*
 * L7-05 — SEO audit engine: in-process crawl with injected faults, stored runs, queue + schedule, admin report and
 * dashboard widgets.
 */

/**
 * A complete, valid page; `$o` overrides head / body parts.
 *
 * @param  array<string, string>  $o
 */
function auditFixtureHtml(string $path, array $o = []): string
{
    $o += [
        'title' => 'صفحه آزمایشی ممیزی سئو برای '.$path.' در ریتمی',
        'description' => 'این صفحه برای آزمودن موتور ممیزی سئوی ریتمی ساخته شده است و توضیحی به اندازه کافی بلند دارد: '.$path,
        'canonical' => 'https://ritme.test'.$path,
        'robots' => 'index,follow',
        'og_image' => '',
        'jsonld' => '{"@context":"https://schema.org","@graph":[{"@type":"WebPage","name":"x"}]}',
        'head' => '',
        'body' => '',
    ];
    $og = '<meta property="og:locale" content="fa_IR"><meta property="og:site_name" content="ریتمی"><meta property="og:type" content="website">'
        .'<meta property="og:url" content="'.$o['canonical'].'"><meta property="og:title" content="t"><meta property="og:description" content="d">'
        .($o['og_image'] === '' ? '' : '<meta property="og:image" content="'.$o['og_image'].'"><meta property="og:image:width" content="1200"><meta property="og:image:height" content="630"><meta property="og:image:alt" content="x">');

    return '<!doctype html><html lang="fa" dir="rtl"><head><meta charset="utf-8">'
        ."<title>{$o['title']}</title><meta name=\"description\" content=\"{$o['description']}\">"
        ."<link rel=\"canonical\" href=\"{$o['canonical']}\"><meta name=\"robots\" content=\"{$o['robots']}\">{$og}"
        ."<script type=\"application/ld+json\">{$o['jsonld']}</script>{$o['head']}</head>"
        ."<body><main id=\"main\"><h1>تیتر صفحه</h1>{$o['body']}</main></body></html>";
}

/**
 * @param  array<string, string>  $o
 */
function auditFixtureRoute(string $path, array $o = []): void
{
    Route::get($path, static fn () => response(auditFixtureHtml($path, $o), 200, ['Content-Type' => 'text/html; charset=UTF-8']));
}

/**
 * @param  list<string>  $paths
 */
function auditFixtureSitemap(array $paths): void
{
    app()->instance('audit.fixture.sitemap', new class($paths) implements SitemapProvider
    {
        /** @param list<string> $paths */
        public function __construct(private readonly array $paths) {}

        public function key(): string
        {
            return 'audit-fixtures';
        }

        public function count(): int
        {
            return count($this->paths);
        }

        public function entries(int $page = 1, int $perPage = 5000): array
        {
            return array_map(static fn (string $p): SitemapEntryData => new SitemapEntryData('https://ritme.test'.$p), $this->paths);
        }
    });
    app()->tag(['audit.fixture.sitemap'], SitemapRegistry::TAG);
    app()->forgetInstance(SitemapRegistry::class);
}

/**
 * @return list<string> "path code" pairs of one severity (or all)
 */
function auditFindings(AuditResult $result, ?Severity $severity = Severity::Error, string $prefix = '/audit'): array
{
    $found = [];
    foreach ($result->pages as $page) {
        if (! str_starts_with($page->path(), $prefix)) {
            continue;
        }
        foreach ($page->audit->issues as $issue) {
            if ($severity === null || $issue->severity === $severity) {
                $found[] = $page->path().' '.$issue->code;
            }
        }
    }
    sort($found);

    return array_values(array_unique($found));
}

function auditAdmin(?AdminRole $role = AdminRole::SeoManager): User
{
    $user = User::factory()->create();
    if ($role !== null) {
        $user->assignRole($role->value);
    }

    return AdminMfa::enrol($user); // PII roles must have MFA (L9-04b)
}

beforeEach(function (): void {
    SeoFixtures::boot();
});

it('detects injected faults across pages, links, redirects, sitemap and JSON-LD', function (): void {
    auditFixtureRoute('/audit/home', ['body' => implode('', [
        '<a href="/audit/clean">ok</a>',
        '<a href="/audit/clean#part">fragment ok</a>',
        '<a href="/audit/clean#nowhere">fragment missing</a>',
        '<a href="#main">skip link</a>',
        '<a href="/audit/missing">broken</a>',
        '<a href="/audit/old">one redirect</a>',
        '<a href="/audit/chain-1">two redirects</a>',
        '<a href="/audit/twin-a">a</a><a href="/audit/twin-b">b</a><a href="/audit/lazy">l</a><a href="/audit/hidden">n</a>',
        '<a href="https://example.org/x">external links are not fetched</a>',
        '<a href="mailto:hi@ritme.test">mail</a>',
    ])]);
    auditFixtureRoute('/audit/clean', ['body' => '<h2 id="part">بخش</h2><a href="/audit/home">خانه</a>']);
    auditFixtureRoute('/audit/twin-a', ['title' => 'عنوان تکراری برای دو صفحه آزمایشی ریتمی']);
    auditFixtureRoute('/audit/twin-b', ['title' => 'عنوان تکراری برای دو صفحه آزمایشی ریتمی']);
    auditFixtureRoute('/audit/lazy', [
        'jsonld' => '{"@context":"https://schema.org","@graph":[{"@type":"Product","name":"x"},{"@type":"BreadcrumbList","itemListElement":[]}]',
        'canonical' => 'https://ritme.test/audit/clean',
        'og_image' => 'https://ritme.test/media/does-not-exist.jpg',
        'head' => '<script src="https://cdn.example.org/lib.js"></script>',
        'body' => '<img src="/a.webp" alt="" width="1" height="1" fetchpriority="high" loading="lazy"><img src="/b.webp" width="1" height="1">',
    ]);
    auditFixtureRoute('/audit/product', ['jsonld' => '{"@context":"https://schema.org","@graph":[{"@type":"Product","name":"x"},{"@type":"BreadcrumbList","itemListElement":[{"@type":"ListItem"}]}]}']);
    auditFixtureRoute('/audit/hidden', ['robots' => 'noindex,follow']);
    auditFixtureRoute('/audit/orphan');
    Route::redirect('/audit/old', '/audit/clean', 301);
    Route::redirect('/audit/chain-1', '/audit/chain-2', 301);
    Route::redirect('/audit/chain-2', '/audit/clean', 301);
    Route::redirect('/audit/moved', '/audit/clean', 301);
    app('router')->getRoutes()->refreshNameLookups();
    auditFixtureSitemap(['/audit/hidden', '/audit/orphan', '/audit/moved', '/audit/gone', '/audit/home', '/audit/clean', '/audit/product']);

    [$result] = app(RunSeoAudit::class)->handle(new AuditOptions, store: false);

    expect(auditFindings($result))->toBe([
        '/audit/gone sitemap.status',
        '/audit/hidden sitemap.noindex',
        '/audit/home link.broken',
        '/audit/home link.fragment',
        '/audit/home redirect.chain',
        '/audit/lazy canonical.self',
        '/audit/lazy external.request',
        '/audit/lazy img.attributes',
        '/audit/lazy img.lcp-lazy',
        '/audit/lazy jsonld.parse',
        '/audit/lazy og.image-missing',
        '/audit/moved sitemap.redirect',
        '/audit/product jsonld.required',
        '/audit/twin-a title.duplicate',
        '/audit/twin-b title.duplicate',
    ])
        ->and(auditFindings($result, Severity::Warning))->toContain('/audit/home link.redirect', '/audit/orphan page.orphan')
        ->and(auditFindings($result, Severity::Warning))->not->toContain('/audit/clean page.orphan', '/audit/home page.orphan')
        ->and(auditFindings($result, Severity::Notice))->toContain('/audit/twin-a sitemap.missing')
        ->and($result->truncated)->toBeFalse();

    $clean = collect($result->pages)->first(static fn (AuditedPage $p): bool => $p->path() === '/audit/clean');
    expect($clean->inbound)->toBeGreaterThanOrEqual(1)
        ->and($clean->audit->errorCount())->toBe(0)
        ->and($clean->audit->score())->toBeGreaterThan(90);
});

it('reports a clean seed with zero errors', function (): void {
    $this->seed(DatabaseSeeder::class);

    [$result] = app(RunSeoAudit::class)->handle(new AuditOptions, store: false);

    expect(count($result->pages))->toBeGreaterThan(15)
        ->and(auditFindings($result, Severity::Error, '/'))->toBe([]);

    $this->artisan('seo:audit', ['--no-store' => true])->assertExitCode(0);
});

it('stays within its page bound and skips the orphan check then', function (): void {
    auditFixtureRoute('/audit/one', ['body' => '<a href="/audit/two">2</a>']);
    auditFixtureRoute('/audit/two');
    app('router')->getRoutes()->refreshNameLookups();

    [$result] = app(RunSeoAudit::class)->handle(new AuditOptions(maxPages: 3), store: false);

    expect($result->pages)->toHaveCount(3)
        ->and($result->truncated)->toBeTrue()
        ->and($result->truncatedBy)->toBe('max-pages')
        ->and(auditFindings($result, Severity::Warning, '/'))->not->toContain('/audit/two page.orphan');
});

it('stores runs with pages, findings and fix links, prioritises failing pages next time and prunes old runs', function (): void {
    auditFixtureRoute('/audit/broken', ['title' => 'کوتاه']);
    app('router')->getRoutes()->refreshNameLookups();

    [$result, $run] = app(RunSeoAudit::class)->handle(new AuditOptions(paths: ['/audit/broken', '/cycle']), RunTrigger::Admin);

    expect($run)->toBeInstanceOf(AuditRun::class)
        ->and($run->status)->toBe(RunStatus::Completed)
        ->and($run->pages_count)->toBe(2)
        ->and($run->errors_count)->toBe($result->count(Severity::Error))
        ->and($run->score)->toBe($result->score())
        ->and(AuditRunPage::query()->where('run_id', $run->id)->where('path', '/audit/broken')->value('errors_count'))->toBe(1);

    // /cycle is a static page: its findings link to the static-page SEO editor.
    $cycle = AuditRunIssue::query()->where('run_id', $run->id)->where('path', '/cycle')->where('code', 'og.image')->first();
    expect($cycle?->fix_url)->toBe('/admin/seo/static-pages/stage-cycle');

    expect(app(AuditHistory::class)->priorityPaths())->toBe(['/audit/broken', '/cycle']);

    for ($i = 0; $i < RunSeoAudit::KEEP_RUNS + 2; $i++) {
        app(RunSeoAudit::class)->handle(new AuditOptions(paths: ['/cycle']));
    }
    expect(AuditRun::query()->count())->toBe(RunSeoAudit::KEEP_RUNS)
        ->and(AuditRun::query()->whereKey($run->id)->exists())->toBeFalse()
        ->and(AuditRunIssue::query()->where('run_id', $run->id)->exists())->toBeFalse();
});

it('queues one run at a time, runs it from the queue and schedules a weekly audit', function (): void {
    config(['queue.default' => 'database']);
    Queue::fake();

    $run = app(QueueSeoAudit::class)->handle(RunTrigger::Admin, 7);
    expect($run?->status)->toBe(RunStatus::Queued)
        ->and($run?->triggered_by)->toBe(7)
        ->and(app(QueueSeoAudit::class)->handle(RunTrigger::Admin))->toBeNull();
    Queue::assertPushed(RunSeoAuditJob::class, static fn (RunSeoAuditJob $job): bool => $job->runId === $run?->id);

    config(['seo.audit.max_pages' => 5]);
    (new RunSeoAuditJob((int) $run?->id))->handle(app(RunSeoAudit::class), config());
    $run?->refresh();
    expect($run?->status)->toBe(RunStatus::Completed)
        ->and($run?->pages_count)->toBe(5)
        ->and($run?->truncated)->toBeTrue();

    $this->artisan('seo:audit', ['--queue' => true])->expectsOutputToContain('queued')->assertExitCode(0);

    $events = collect(app(Schedule::class)->events())->map(static fn ($e): string => (string) $e->description);
    expect($events)->toContain('seo-audit');
});

it('shows the report and SEO widgets to SEO managers and super-admins only, with filters and "run again"', function (): void {
    $this->seed(AdminRolesSeeder::class);
    Filament::setCurrentPanel('admin');
    auditFixtureRoute('/audit/broken', ['title' => 'کوتاه']);
    app('router')->getRoutes()->refreshNameLookups();
    app(RunSeoAudit::class)->handle(new AuditOptions(paths: ['/audit/broken', '/cycle']));
    NotFoundLog::query()->create(['path' => '/old-page', 'path_hash' => sha1('/old-page'), 'agent' => AgentClass::Human, 'hits' => 9, 'first_seen_at' => now(), 'last_seen_at' => now()]);
    Cache::put('search-terms:'.CarbonImmutable::now()->format('Y-m-d'), ['کیسه آب گرم' => ['count' => 4, 'results' => 0], 'پریود' => ['count' => 9, 'results' => 3]], 3600);

    $this->get(AuditReport::getUrl())->assertRedirect('/admin/login');
    foreach ([AdminRole::Editor, AdminRole::ShopManager, AdminRole::Support, null] as $role) {
        $user = auditAdmin($role);
        $this->actingAs($user)->get(AuditReport::getUrl())->assertForbidden();
        $this->actingAs($user)->get('/admin')->assertDontSeeLivewire(SeoHealthOverview::class);
    }

    foreach ([AdminRole::SeoManager, AdminRole::SuperAdmin] as $role) {
        $this->actingAs(auditAdmin($role))->get(AuditReport::getUrl())->assertOk()->assertSee('گزارش ممیزی سئو');
    }

    $this->actingAs($manager = auditAdmin());
    $this->get('/admin')->assertOk()
        ->assertSeeLivewire(SeoHealthOverview::class)->assertSeeLivewire(SeoScoreTrend::class)
        ->assertSeeLivewire(SeoTopIssues::class)->assertSeeLivewire(SeoTopNotFound::class)
        ->assertSeeLivewire(SeoZeroResultSearches::class)->assertSeeLivewire(SeoContentNeedingWork::class);

    $broken = AuditRunIssue::query()->where('path', '/audit/broken')->where('code', 'title.length')->firstOrFail();
    $og = AuditRunIssue::query()->where('path', '/cycle')->where('code', 'og.image')->firstOrFail();
    Livewire::test(AuditReport::class)
        ->assertCanSeeTableRecords([$broken, $og])
        ->filterTable('severity', 'error')
        ->assertCanSeeTableRecords([$broken])
        ->assertCanNotSeeTableRecords([$og])
        ->resetTableFilters()
        ->filterTable('code', 'og.image')
        ->assertCanSeeTableRecords([$og])
        ->assertCanNotSeeTableRecords([$broken]);
    $this->get(AuditReport::getUrl(['code' => 'title.length']))->assertOk()->assertSee('/audit/broken');

    Livewire::test(SeoHealthOverview::class)->assertSee('امتیاز سلامت سئو')->assertSee('صفحه‌های بدون تصویر اشتراک');
    Livewire::test(SeoTopIssues::class)->assertSee('طول عنوان')->assertSee('بدون تصویر اشتراک');
    Livewire::test(SeoTopNotFound::class)->assertSee('/old-page')->assertSee('۹');
    Livewire::test(SeoZeroResultSearches::class)->assertSee('کیسه آب گرم')->assertDontSee('پریود');
    Livewire::test(SeoScoreTrend::class)->assertOk();
    Livewire::test(SeoContentNeedingWork::class)->assertOk();

    config(['queue.default' => 'database']);
    Queue::fake();
    Livewire::test(AuditReport::class)->callAction('rerun')->assertNotified();
    Queue::assertPushed(RunSeoAuditJob::class);
    expect(AuditRun::query()->latest('id')->first()?->triggered_by)->toBe($manager->id)
        ->and(AuditRun::query()->latest('id')->first()?->trigger)->toBe(RunTrigger::Admin);
});
