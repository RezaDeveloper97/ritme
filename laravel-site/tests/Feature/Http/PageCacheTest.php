<?php

declare(strict_types=1);

use App\Domain\Settings\Models\Setting;
use App\Http\Middleware\PageCache;
use App\Models\User;
use Database\Seeders\SettingsSeeder;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Route;
use Illuminate\Testing\TestResponse;
use Livewire\Features\SupportAutoInjectedAssets\SupportAutoInjectedAssets;

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);

    // Livewire keeps "a component was rendered" in a static; an earlier admin test would make it inject its
    // assets into these public pages (never happens in a real, fresh PHP request).
    SupportAutoInjectedAssets::$hasRenderedAComponentThisRequest = false;

    // A page with a CSRF-protected form, inside the web group like every page route.
    Route::middleware('web')->group(function (): void {
        Route::get('/_test/form', static fn () => response(
            '<!doctype html><html><body><h1>فرم</h1><form method="post" action="/_test/form">'.csrf_field().'</form></body></html>',
        ));
        Route::get('/_test/flash', static function () {
            session()->flash('status', 'saved');

            return response('<html><body>flashed</body></html>');
        });
        Route::get('/_test/cookie', static fn () => response('<html><body>cookie</body></html>')->cookie('ritme_seen', '1'));
        Route::get('/_test/no-store', static fn () => response('<html><body>private</body></html>')->header('Cache-Control', 'no-store'));
        Route::get('/_test/json', static fn () => response()->json(['ok' => true]));
        Route::get('/_test/missing', static fn () => abort(404));
    });
});

/** Per-request values (CSP nonce, CSRF token) normalised away. */
function withoutPerRequestValues(string $html): string
{
    return (string) preg_replace('/(nonce|data-csrf|value)="[A-Za-z0-9+\/=_-]{20,}"/', '$1=""', $html);
}

function pageCacheState(TestResponse $response): ?string
{
    return $response->headers->get(PageCache::HEADER);
}

it('serves the second anonymous request to / from the cache with zero database queries', function (): void {
    $first = $this->get('/')->assertOk();
    expect(pageCacheState($first))->toBe('MISS');

    $queries = 0;
    DB::listen(function () use (&$queries): void {
        $queries++;
    });

    $second = $this->get('/')->assertOk();

    expect(pageCacheState($second))->toBe('HIT')
        ->and($queries)->toBe(0)
        ->and(withoutPerRequestValues((string) $second->getContent()))->toBe(withoutPerRequestValues((string) $first->getContent()))
        ->and($second->headers->get('Content-Type'))->toContain('text/html');
});

it('serves HEAD from the same entry', function (): void {
    $this->get('/cycle');

    expect(pageCacheState($this->call('HEAD', '/cycle')->assertOk()))->toBe('HIT');
});

it('keys the cache by path and whitelisted query, ignoring tracking parameters', function (): void {
    $this->get('/faq');

    expect(pageCacheState($this->get('/faq?utm_source=x&fbclid=y')))->toBe('HIT')
        ->and(pageCacheState($this->get('/faq?page=2')))->toBe('MISS')
        ->and(pageCacheState($this->get('/faq?page=2')))->toBe('HIT')
        ->and(pageCacheState($this->get('/about')))->toBe('MISS');
});

it('bypasses unknown query parameters (search, filters)', function (): void {
    expect(pageCacheState($this->get('/blog?q=درد')))->toBe('BYPASS')
        ->and(pageCacheState($this->get('/blog?q=درد')))->toBe('BYPASS');
});

it('is invalidated when an observer bumps the pages namespace (settings change)', function (): void {
    $this->get('/');
    expect(pageCacheState($this->get('/')))->toBe('HIT');

    $setting = Setting::query()->firstOrFail();
    $setting->touch();

    expect(pageCacheState($this->get('/')))->toBe('MISS');
});

it('never caches authenticated users', function (): void {
    $this->get('/');

    $response = $this->actingAs(User::factory()->create())->get('/');

    expect(pageCacheState($response))->toBe('BYPASS');
});

it('bypasses remember-me and cart cookies', function (): void {
    $this->get('/');

    expect(pageCacheState($this->withUnencryptedCookie('remember_web_abc', 'x')->get('/')))->toBe('BYPASS');
    $this->flushHeaders();
    expect(pageCacheState($this->withUnencryptedCookie('ritme_cart', '1')->get('/')))->toBe('BYPASS');
});

it('bypasses a request Cache-Control: no-cache', function (): void {
    $this->get('/');

    expect(pageCacheState($this->withHeader('Cache-Control', 'no-cache')->get('/')))->toBe('BYPASS');
});

it('bypasses sessions with flash data or validation errors', function (): void {
    $this->get('/');

    expect(pageCacheState($this->withSession(['_flash' => ['old' => ['status'], 'new' => []], 'status' => 'ok'])->get('/')))->toBe('BYPASS');
});

it('does not store a render that flashed, set a cookie, opted out or is not 200 HTML', function (string $uri): void {
    $this->get($uri);

    expect(pageCacheState($this->get($uri)))->toBe('BYPASS');
})->with(['/_test/flash', '/_test/cookie', '/_test/no-store', '/_test/json', '/_test/missing']);

it('never caches transactional noindex routes', function (string $uri): void {
    $this->get($uri);

    expect(pageCacheState($this->get($uri)))->toBe('BYPASS');
})->with(['/shop/cart', '/shop/checkout', '/shop/order/AB-12', '/directory/booked/RT-1']);

it('never caches the admin, Livewire or POST requests', function (): void {
    // Filament's panel routes do not use the web group; PageCache also excludes the admin path explicitly.
    $this->get('/admin/login');
    expect(pageCacheState($this->get('/admin/login')))->toBeIn([null, 'BYPASS'])
        ->and(pageCacheState($this->get('/livewire/livewire.js')))->toBeIn([null, 'BYPASS'])
        ->and($this->post('/_test/form')->headers->get(PageCache::HEADER))->not->toBe('HIT');
});

it('swaps the CSRF token of cached forms for the current visitor\'s token', function (): void {
    $first = $this->get('/_test/form');
    expect(pageCacheState($first))->toBe('MISS');
    $firstToken = session()->token();
    expect($first->getContent())->toContain('value="'.$firstToken.'"');

    session()->flush();
    session()->regenerateToken();
    $secondToken = session()->token();

    $second = $this->get('/_test/form');

    expect(pageCacheState($second))->toBe('HIT')
        ->and($secondToken)->not->toBe($firstToken)
        ->and($second->getContent())->toContain('value="'.$secondToken.'"')
        ->and($second->getContent())->not->toContain($firstToken)
        ->and($second->getContent())->not->toContain('__RT_PAGE_CACHE');
});

it('can be switched off', function (): void {
    config(['pagecache.enabled' => false]);

    $this->get('/');

    expect(pageCacheState($this->get('/')))->toBe('BYPASS');
});
