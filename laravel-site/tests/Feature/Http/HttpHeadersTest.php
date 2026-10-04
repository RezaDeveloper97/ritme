<?php

declare(strict_types=1);

use App\Filament\Auth\AdminRole;
use App\Http\Middleware\MinifyHtml;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Illuminate\Foundation\Vite;

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
});

// --- HTTP caching ----------------------------------------------------------------------------------------------

it('sends public shared-cache headers and no cookie to a new visitor on a cacheable page', function (): void {
    $response = $this->get('/cycle')->assertOk();

    $headers = $response->headers;

    expect($headers->hasCacheControlDirective('public'))->toBeTrue()
        ->and($headers->hasCacheControlDirective('private'))->toBeFalse()
        ->and($headers->getCacheControlDirective('max-age'))->toBe('0')
        ->and($headers->getCacheControlDirective('s-maxage'))->toBe('300')
        ->and($headers->getCacheControlDirective('stale-while-revalidate'))->toBe('60')
        ->and($headers->getCookies())->toBe([])
        ->and($response->headers->get('ETag'))->toStartWith('W/"');
});

it('keeps the session private for a visitor who already has a session', function (): void {
    $response = $this->withCookie((string) config('session.cookie'), 'existing-session-id')->get('/cycle')->assertOk();

    expect($response->headers->hasCacheControlDirective('private'))->toBeTrue()
        ->and($response->headers->hasCacheControlDirective('no-cache'))->toBeTrue()
        ->and($response->headers->hasCacheControlDirective('s-maxage'))->toBeFalse()
        ->and($response->headers->get('ETag'))->toStartWith('W/"');
});

it('answers 304 Not Modified for a matching If-None-Match', function (): void {
    $etag = (string) $this->get('/about')->headers->get('ETag');

    $response = $this->withHeader('If-None-Match', $etag)->get('/about');

    $response->assertStatus(304);
    expect($response->getContent())->toBe('');
});

it('leaves uncached responses alone', function (): void {
    $admin = $this->get('/admin/login');
    $user = $this->actingAs(User::factory()->create())->get('/');

    expect($admin->headers->get('Cache-Control'))->not->toContain('public')
        ->and($user->headers->get('Cache-Control'))->not->toContain('public')
        ->and($user->headers->has('ETag'))->toBeFalse();
});

// --- Security headers ------------------------------------------------------------------------------------------

it('sends a strict CSP without any external origin on public pages', function (string $uri): void {
    $response = $this->get($uri);
    $csp = (string) $response->headers->get('Content-Security-Policy');

    expect($csp)->toContain("default-src 'self'")
        ->and($csp)->toContain("script-src 'self';")
        ->and($csp)->toContain("style-src 'self';")
        ->and($csp)->toContain("img-src 'self' data:;")
        ->and($csp)->toContain("object-src 'none'")
        ->and($csp)->toContain("frame-ancestors 'self'")
        ->and($csp)->not->toContain('unsafe-inline')
        ->and($csp)->not->toContain('unsafe-eval')
        ->and($csp)->not->toMatch('#(https?:|wss?:)?//|\*#');
})->with(['/', '/cycle', '/does-not-exist']);

it('adds a fresh per-request nonce when enabled and keeps the ETag stable across nonces', function (): void {
    config(['pagecache.security.nonce' => true]);

    $first = $this->get('/cycle');
    $firstCsp = (string) $first->headers->get('Content-Security-Policy');
    $second = $this->get('/cycle');
    $secondCsp = (string) $second->headers->get('Content-Security-Policy');
    preg_match("/'nonce-([^']+)'/", $secondCsp, $m);

    expect($m[1] ?? '')->toBe(app(Vite::class)->cspNonce())
        ->and($secondCsp)->not->toBe($firstCsp)
        ->and($second->headers->get('X-Page-Cache'))->toBe('HIT')
        ->and($second->headers->get('ETag'))->toBe($first->headers->get('ETag'));
});

it('sends the baseline security headers on every response, including redirects', function (string $uri): void {
    $headers = $this->get($uri)->headers;

    expect($headers->get('X-Content-Type-Options'))->toBe('nosniff')
        ->and($headers->get('X-Frame-Options'))->toBe('SAMEORIGIN')
        ->and($headers->get('Referrer-Policy'))->toBe('strict-origin-when-cross-origin')
        ->and($headers->get('Permissions-Policy'))->toContain('camera=()')
        ->and($headers->has('Content-Security-Policy'))->toBeTrue();
})->with(['/', '/cycle.html', '/admin/login']);

it('gives the Filament admin its own policy that still allows no external origin', function (): void {
    $response = $this->get('/admin/login')->assertOk();
    $csp = (string) $response->headers->get('Content-Security-Policy');

    expect($csp)->toContain("script-src 'self' 'unsafe-inline' 'unsafe-eval'")
        ->and($csp)->toContain("style-src 'self' 'unsafe-inline'")
        ->and($csp)->not->toContain('nonce-')
        ->and($csp)->not->toMatch('#(https?:|wss?:)?//|\*#');
});

it('renders the admin dashboard for an admin under the admin policy', function (): void {
    $this->seed(AdminRolesSeeder::class);
    $user = User::factory()->create();
    $user->assignRole(AdminRole::Editor->value);

    $response = $this->actingAs($user)->get('/admin');

    expect($response->getStatusCode())->toBeIn([200, 302])
        ->and((string) $response->headers->get('Content-Security-Policy'))->toContain("'unsafe-eval'");
});

it('sends HSTS only over https in production', function (): void {
    expect($this->get('https://localhost/')->headers->has('Strict-Transport-Security'))->toBeFalse();

    $this->app['env'] = 'production';

    expect($this->get('http://localhost/')->headers->has('Strict-Transport-Security'))->toBeFalse()
        ->and($this->get('https://localhost/')->headers->get('Strict-Transport-Security'))->toBe('max-age=31536000');
});

it('adds upgrade-insecure-requests only over https', function (): void {
    expect((string) $this->get('https://localhost/')->headers->get('Content-Security-Policy'))->toContain('upgrade-insecure-requests')
        ->and((string) $this->get('http://localhost/')->headers->get('Content-Security-Policy'))->not->toContain('upgrade-insecure-requests');
});

// --- Minifier --------------------------------------------------------------------------------------------------

it('minifies the public HTML (cached pages are stored minified)', function (): void {
    $html = (string) $this->get('/')->getContent();

    expect($html)->not->toMatch("/>\n\s+</")
        ->and($html)->toStartWith('<!doctype html><html');
});

it('keeps pre, textarea, script, style and conditional comments intact', function (): void {
    $html = "<div>\n  <p>a  <b>b</b>\n c</p>\n  <!-- note -->\n<pre>  x\n   y</pre>\n<textarea>\n line </textarea>"
        ."<script type=\"application/ld+json\">{\n  \"a\": 1\n}</script><style>\n a { b: c }\n</style><!--[if IE]><p>ie</p><![endif]--></div>";

    expect(MinifyHtml::minify($html))->toBe(
        '<div><p>a <b>b</b> c</p><pre>  x'."\n".'   y</pre> <textarea>'."\n".' line </textarea>'
        ."<script type=\"application/ld+json\">{\n  \"a\": 1\n}</script><style>\n a { b: c }\n</style><!--[if IE]><p>ie</p><![endif]--></div>",
    );
});
