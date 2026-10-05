<?php

declare(strict_types=1);

use App\Domain\Seo\Indexing\Actions\SaveIndexingSettings;
use App\Domain\Seo\Indexing\InvalidIndexingSettings;
use App\Domain\Seo\Indexing\Jobs\PingSitemaps;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Enums\SettingGroup;
use App\Http\Controllers\Blog\NewsletterController;
use App\Http\Controllers\Directory\BookingController;
use App\Http\Controllers\Shop\CheckoutController;
use App\Support\Http\OutboundUrl;
use Database\Seeders\SettingsSeeder;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Route;

/*
 * L9-04 regression tests: cache unserialisation, per-route rate limits, code/token lookup throttles, proxy headers,
 * cookie flags, SSRF guard for server-side pings.
 */

it('never instantiates objects from the cache (allowed_classes = false)', function (): void {
    expect(config('cache.serializable_classes'))->toBeFalse();

    $dir = sys_get_temp_dir().'/ritme-sec-cache-'.bin2hex(random_bytes(4));
    $store = Cache::build(['driver' => 'file', 'path' => $dir]);
    $store->put('object', new ArrayObject(['x' => 1]), 60);
    $store->put('array', ['a' => [1, 'b']], 60);

    expect($store->get('object'))->toBeInstanceOf(__PHP_Incomplete_Class::class)
        ->and($store->get('array'))->toBe(['a' => [1, 'b']]);

    $store->flush();
    @rmdir($dir);
});

it('counts throttle:N,M per route, not one shared counter per IP', function (): void {
    Route::middleware(['web', 'throttle:2,1'])->group(function (): void {
        Route::get('/_sec/a', fn () => 'a')->name('sec.a');
        Route::get('/_sec/b', fn () => 'b')->name('sec.b');
    });

    $this->get('/_sec/a')->assertOk();
    $this->get('/_sec/a')->assertOk();
    $this->get('/_sec/a')->assertStatus(429);
    $this->get('/_sec/b')->assertOk(); // another route keeps its own budget
});

it('ignores X-Forwarded-For unless a proxy is configured, so rate limits key on the real peer', function (): void {
    expect(config('app.trusted_proxies'))->toBe([]);
    Route::get('/_sec/ip', fn (Request $request) => (string) $request->ip());

    $this->withHeaders(['X-Forwarded-For' => '203.0.113.9'])->get('/_sec/ip')->assertSeeText('127.0.0.1');
});

it('answers 429 after the lookup budget on every code / token page', function (string $url, int $limit): void {
    $this->seed(SettingsSeeder::class);

    for ($i = 0; $i < $limit; $i++) {
        expect($this->get($url)->status())->not->toBe(429);
    }
    $this->get($url)->assertStatus(429);
})->with([
    'order page' => ['/shop/order/AAAA-BBBB-CCCC', CheckoutController::LOOKUPS_PER_MINUTE],
    'booked page' => ['/directory/booked/AAAA-BBBB-CCCC', BookingController::LOOKUPS_PER_MINUTE],
    'newsletter confirm' => ['/newsletter/confirm/abc123', NewsletterController::LOOKUPS_PER_MINUTE],
    'newsletter unsubscribe form' => ['/newsletter/unsubscribe/abc123', NewsletterController::LOOKUPS_PER_MINUTE],
]);

it('marks the session cookie secure by default when APP_URL is https, httpOnly + SameSite=Lax always', function (): void {
    $had = array_key_exists('APP_URL', $_SERVER) ? $_SERVER['APP_URL'] : null;
    expect(env('SESSION_SECURE_COOKIE'))->toBeNull(); // the default is what is under test

    try {
        $_SERVER['APP_URL'] = 'https://ritme.ir';
        $https = require config_path('session.php');
        $_SERVER['APP_URL'] = 'http://127.0.0.1:8000';
        $http = require config_path('session.php');
    } finally {
        if ($had === null) {
            unset($_SERVER['APP_URL']);
        } else {
            $_SERVER['APP_URL'] = $had;
        }
    }

    expect($https['secure'])->toBeTrue()
        ->and($http['secure'])->toBeFalse()
        ->and($https['http_only'])->toBeTrue()
        ->and($https['same_site'])->toBe('lax');
});

it('only lets the server request public https hosts (SSRF guard)', function (string $url, bool $safe): void {
    expect(OutboundUrl::isSafe($url))->toBe($safe);
})->with([
    ['https://www.bing.com/ping?sitemap=x', true],
    ['https://ping.example/ping?s=x', true],
    ['http://www.bing.com/ping', false],
    ['https://127.0.0.1/admin', false],
    ['https://169.254.169.254/latest/meta-data', false],
    ['https://[::1]/', false],
    ['https://10.0.0.5/', false],
    ['https://localhost/', false],
    ['https://intranet/', false],
    ['https://db.internal/', false],
    ['https://printer.local/', false],
    ['https://user:pass@example.com/', false],
    ['https://example.com:8080/', false],
    ['file:///etc/passwd', false],
    ['gopher://example.com/', false],
]);

it('refuses internal ping endpoints on save and skips them in the job', function (): void {
    $this->seed(SettingsSeeder::class);
    config(['app.url' => 'https://ritme.ir']);
    Http::preventStrayRequests();
    Http::fake(['ping.example/*' => Http::response('ok')]);

    expect(fn () => app(SaveIndexingSettings::class)->handle(['sitemap_ping_urls' => ['https://127.0.0.1/?s={sitemap}']], true))
        ->toThrow(InvalidIndexingSettings::class);

    // A row written before the guard existed (or straight into the DB) is skipped at send time.
    app(SettingsRepository::class)->put(SettingGroup::Seo, ['sitemap_ping_urls' => ['https://127.0.0.1/?s={sitemap}', 'https://ping.example/ping?s={sitemap}']]);
    config(['app.env' => 'production']);
    app()->call([new PingSitemaps, 'handle']);

    Http::assertSentCount(1);
    Http::assertSent(fn ($request): bool => str_starts_with($request->url(), 'https://ping.example/'));
});
