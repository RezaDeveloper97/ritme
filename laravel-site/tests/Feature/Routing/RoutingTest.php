<?php

declare(strict_types=1);

use App\Domain\Directory\Booking\Models\BookingRequest;
use App\Http\Controllers\PlaceholderPageController;
use Database\Seeders\BlogSeeder;
use Database\Seeders\DirectorySeeder;
use Database\Seeders\SettingsSeeder;
use Database\Seeders\ShopSeeder;
use Illuminate\Contracts\Http\Kernel;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Testing\TestResponse;

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
});

/**
 * The machine-readable ```json urlmap``` block of docs/AUDIT.md §7.
 *
 * @return list<array{design: string, route: string, name: string, owner: string}>
 */
function auditUrlMap(): array
{
    $audit = (string) file_get_contents(dirname(__DIR__, 3).'/docs/AUDIT.md');
    preg_match('/```json urlmap\s*(.+?)```/s', $audit, $match);

    /** @var list<array{design: string, route: string, name: string, owner: string}> $map */
    $map = json_decode($match[1] ?? '[]', true, flags: JSON_THROW_ON_ERROR);

    return $map;
}

/**
 * Rows of the AUDIT §7 table: old file (without .html) => [new path, WordPress slugs].
 *
 * @return array<string, array{0: string, 1: list<string>}>
 */
function auditLegacyRows(): array
{
    $audit = (string) file_get_contents(dirname(__DIR__, 3).'/docs/AUDIT.md');
    preg_match_all('/^\| `\/([a-z-]+)\.html` \| (.+?) \| `([^`]+)` \| 301 \|$/m', $audit, $rows, PREG_SET_ORDER);

    $map = [];
    foreach ($rows as [, $file, $slugs, $target]) {
        preg_match_all('/`([^`]+)`/', $slugs, $slugMatches);
        $map[$file] = [$target, $slugMatches[1]];
    }

    return $map;
}

/** Origin the test requests run on (APP_URL); redirects keep it while the host switch is off. */
function appOrigin(): string
{
    return rtrim((string) config('app.url'), '/');
}

/**
 * Sends the path exactly as given: the test client's get() trims trailing slashes, which these tests are about.
 */
function rawRequest(string $uri, string $method = 'GET'): TestResponse
{
    $url = str_starts_with($uri, 'http') ? $uri : appOrigin().$uri;

    return TestResponse::fromBaseResponse(app(Kernel::class)->handle(Request::create($url, $method)));
}

function concreteUrl(string $route): string
{
    return str_replace('{code}', 'RT-2026-0042', $route);
}

it('reads the full URL map from the audit', function (): void {
    expect(auditUrlMap())->toHaveCount(29)
        ->and(auditLegacyRows())->toHaveCount(29);
});

it('serves every audit page under its route name as a noindex placeholder with one h1', function (string $design, string $route, string $name): void {
    // Real content pages need their demo rows (pages are swapped from the placeholder task by task).
    if ($name === 'blog.show') {
        $this->seed(BlogSeeder::class);
    }
    if ($name === 'directory.place') {
        $this->seed(DirectorySeeder::class);
    }
    if ($name === 'shop.category') {
        $this->seed(ShopSeeder::class);
    }
    if ($name === 'directory.booked') {
        BookingRequest::query()->create([
            'code' => 'RT-2026-0042', 'place_name' => 'مجموعه', 'preferred_date' => '2026-10-10', 'time_window' => 'any',
            'parent_name' => 'سارا', 'mobile' => '09121234567',
        ]);
    }

    $url = concreteUrl($route);
    $html = $this->get($url)->assertOk()->getContent();

    expect(app('router')->getRoutes()->match(Request::create($url))->getName())->toBe($name)
        ->and(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('<meta name="robots" content="noindex')
        ->and($html)->toContain('<link rel="canonical" href="https://')
        ->and($html)->not->toMatch('/\sstyle\s*=/i');
})->with(fn (): array => array_map(static fn (array $row): array => [$row['design'], $row['route'], $row['name']], auditUrlMap()));

it('serves /terms (added beyond the design)', function (): void {
    $this->get('/terms')->assertOk()->assertSee('شرایط استفاده');
});

it('keeps the placeholder pages noindex even in production', function (): void {
    config(['app.env' => 'production']);

    // Any route still served by the placeholder (pages are swapped out task by task).
    $route = collect(app('router')->getRoutes()->getRoutes())
        ->first(fn ($r): bool => $r->getActionName() === PlaceholderPageController::class
            && in_array('GET', $r->methods(), true) && $r->parameterNames() === []);

    if ($route === null) {
        $this->markTestSkipped('No placeholder pages left.');
    }

    $this->get('/'.ltrim($route->uri(), '/'))->assertOk()->assertSee('<meta name="robots" content="noindex', false);
});

it('redirects every old .html URL in one 301 hop to a page that answers 200', function (string $file, string $target): void {
    foreach (["/{$file}.html", "/ritme-static/{$file}.html", '/'.strtoupper($file).'.HTML'] as $old) {
        $response = $this->get($old)->assertStatus(301)->assertHeader('Location', appOrigin().($target === '/' ? '/' : $target));
        $this->get((string) parse_url((string) $response->headers->get('Location'), PHP_URL_PATH))->assertOk();
    }
})->with(fn (): array => array_map(static fn (array $row, string $file): array => [$file, $row[0]], auditLegacyRows(), array_keys(auditLegacyRows())));

it('redirects every WordPress slug in one 301 hop', function (string $file, string $target, array $slugs): void {
    foreach ($slugs as $slug) {
        rawRequest($slug)->assertStatus(301)->assertHeader('Location', appOrigin().$target);
    }
})->with(fn (): array => array_map(static fn (array $row, string $file): array => [$file, $row[0], $row[1]], auditLegacyRows(), array_keys(auditLegacyRows())));

it('keeps the query string on redirects', function (): void {
    $this->get('/cycle.html?utm_source=telegram&x=1')->assertStatus(301)
        ->assertHeader('Location', appOrigin().'/cycle?utm_source=telegram&x=1');
    rawRequest('/blog/?page=2')->assertStatus(301)->assertHeader('Location', appOrigin().'/blog?page=2');
});

it('removes trailing slashes with a 301', function (string $from, string $to): void {
    rawRequest($from)->assertStatus(301)->assertHeader('Location', appOrigin().$to);
})->with([
    ['/cycle/', '/cycle'],
    ['/blog/', '/blog'],
    ['/shop/', '/shop'],
    ['/directory/', '/directory'],
    ['/directory/join/done/', '/directory/join/done'],
    ['/blog/period-pain/', '/blog/period-pain'],
    ['/nope/', '/nope'],
]);

it('serves the root without a redirect', function (): void {
    $this->get('/')->assertOk();
});

it('collapses duplicate slashes with a 301', function (string $from, string $to): void {
    rawRequest($from)->assertStatus(301)->assertHeader('Location', appOrigin().$to);
})->with([
    ['/shop//cart', '/shop/cart'],
    ['/blog///period-pain/', '/blog/period-pain'],
    ['/directory//join//done', '/directory/join/done'],
]);

it('lowercases static page paths with a 301', function (string $from, string $to): void {
    rawRequest($from)->assertStatus(301)->assertHeader('Location', appOrigin().$to);
})->with([
    ['/Cycle', '/cycle'],
    ['/SHOP/Cart', '/shop/cart'],
    ['/About/', '/about'],
    ['/Social-Responsibility', '/social-responsibility'],
]);

it('leaves the case of content slugs alone', function (string $url): void {
    // No case-folding redirect; the content lookup itself decides between 200 and 404.
    expect($this->get($url)->status())->not->toBe(301);
})->with([
    '/blog/Period-Pain',
    '/blog/'.rawurlencode('درد-پریود'),
    '/shop/product/Long-Sleeve',
]);

it('answers 410 for the old asset folders, for any method', function (string $url): void {
    $html = $this->get($url)->assertStatus(410)->getContent();
    $this->post($url)->assertStatus(410);

    expect($html)->toContain('<meta name="robots" content="noindex')
        ->and(substr_count($html, '<h1'))->toBe(1);
})->with([
    '/ritme-static/assets/css/ritme.css',
    '/ritme-static/assets',
    '/wp-content/uploads/ritme/hero.png',
    '/WP-Content/Uploads/Ritme/hero.png',
]);

it('answers 404 for unknown URLs, including unknown legacy files', function (string $url): void {
    $this->get($url)->assertNotFound();
})->with(['/nope', '/nope.html', '/ritme-static/nope.html', '/ritme-static/cycle', '/cycle/extra.html', '/blog/a/b', '/directory/booked/<x>']);

it('never redirects unsafe methods', function (): void {
    rawRequest('/cycle/', 'POST')->assertStatus(405);
    $this->post('/cycle.html')->assertNotFound();
});

it('forces https and the APP_URL host in one hop when enabled', function (): void {
    config(['app.canonical_redirect' => true, 'app.url' => 'https://ritme.app']);

    rawRequest('http://www.ritme.app/Cycle/?a=1')->assertStatus(301)->assertHeader('Location', 'https://ritme.app/cycle?a=1');
    $this->get('http://ritme.app/cycle.html')->assertStatus(301)->assertHeader('Location', 'https://ritme.app/cycle');
    $this->get('https://ritme.app/cycle')->assertOk();
});

it('leaves host and scheme alone unless switched on, also in production', function (): void {
    config(['app.url' => 'https://ritme.app']);
    rawRequest('http://localhost/faq')->assertOk();

    config(['app.env' => 'production']);
    rawRequest('http://localhost/faq')->assertOk();

    config(['app.canonical_redirect' => false]);
    rawRequest('http://localhost/faq')->assertOk();
});

it('caches the routes (no closures)', function (): void {
    $path = storage_path('framework/testing/routes-l105-'.getmypid().'.php');
    $_SERVER['APP_ROUTES_CACHE'] = $_ENV['APP_ROUTES_CACHE'] = $path;

    try {
        expect(Artisan::call('route:cache'))->toBe(0)
            ->and(is_file($path))->toBeTrue()
            ->and((string) file_get_contents($path))->toContain('stage.cycle')->toContain('shop.order');
    } finally {
        @unlink($path);
        unset($_SERVER['APP_ROUTES_CACHE'], $_ENV['APP_ROUTES_CACHE']);
    }
});
