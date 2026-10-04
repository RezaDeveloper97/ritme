<?php

declare(strict_types=1);

use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Support\ViewCounter;
use App\Http\Controllers\Blog\PostViewController;
use Illuminate\Cookie\Middleware\EncryptCookies;
use Illuminate\Foundation\Http\Middleware\ValidateCsrfToken;
use Illuminate\Session\Middleware\StartSession;
use Illuminate\Support\Facades\RateLimiter;
use Illuminate\Support\Facades\Route;
use Illuminate\Testing\TestResponse;

const BEACON_UA = 'Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0 Mobile Safari/537.36';

/**
 * @param  array<string, string>  $headers
 */
function beacon(Post $post, array $headers = [], ?string $id = null): TestResponse
{
    return test()->withHeaders($headers + ['User-Agent' => BEACON_UA, 'Sec-Fetch-Site' => 'same-origin'])
        ->post(route('blog.view', $post->slug), ['id' => $id ?? (string) $post->id]);
}

beforeEach(function (): void {
    RateLimiter::clear(sha1('127.0.0.1')); // fresh throttle bucket per test
});

it('registers a POST route without session, cookies or CSRF but throttled', function (): void {
    $route = Route::getRoutes()->getByName('blog.view');

    expect($route?->getActionName())->toBe(PostViewController::class)
        ->and($route->methods())->toContain('POST')
        ->and($route->excludedMiddleware())->toContain(StartSession::class, EncryptCookies::class, ValidateCsrfToken::class)
        ->and(implode(' ', $route->gatherMiddleware()))->toContain('throttle:'.PostViewController::PER_MINUTE.',1');
});

it('counts a cached (HIT) article once per visit through the beacon', function (): void {
    $post = Post::factory()->published()->create(['slug' => 'beacon-post']);

    $this->get('/blog/beacon-post')->assertOk()->assertHeader('X-Page-Cache', 'MISS');
    $html = (string) $this->get('/blog/beacon-post')->assertOk()->assertHeader('X-Page-Cache', 'HIT')->getContent();

    expect($html)->toContain('data-module="view-beacon"')
        ->toContain('data-view-url="'.route('blog.view', 'beacon-post').'"')
        ->toContain('data-view-id="'.$post->id.'"')
        ->and(app(ViewCounter::class)->pending($post->id))->toBe(0);

    $response = beacon($post)->assertNoContent();
    expect($response->headers->getCookies())->toBe([])
        ->and(app(ViewCounter::class)->pending($post->id))->toBe(1);

    beacon($post)->assertNoContent(); // second visit (another tab session)
    expect(app(ViewCounter::class)->pending($post->id))->toBe(2);
});

it('ignores bots, prefetches, Save-Data and cross-site requests but still answers 204', function (array $headers): void {
    $post = Post::factory()->published()->create(['slug' => 'ignored']);

    beacon($post, $headers)->assertNoContent();

    expect(app(ViewCounter::class)->pending($post->id))->toBe(0);
})->with([
    'googlebot' => [['User-Agent' => 'Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)']],
    'bingbot' => [['User-Agent' => 'Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)']],
    'link preview' => [['User-Agent' => 'TelegramBot (like TwitterBot)']],
    'lighthouse' => [['User-Agent' => 'Mozilla/5.0 Chrome-Lighthouse']],
    'curl' => [['User-Agent' => 'curl/8.5.0']],
    'empty agent' => [['User-Agent' => '']],
    'save-data' => [['Save-Data' => 'on']],
    'prefetch' => [['Sec-Purpose' => 'prefetch']],
    'cross-site' => [['Sec-Fetch-Site' => 'cross-site']],
    'foreign origin' => [['Origin' => 'https://evil.example']],
]);

it('accepts a same-origin Origin header', function (): void {
    $post = Post::factory()->published()->create(['slug' => 'origin-ok']);

    beacon($post, ['Origin' => rtrim(url('/'), '/')])->assertNoContent();

    expect(app(ViewCounter::class)->pending($post->id))->toBe(1);
});

it('404s unless the id is the published post at the slug', function (): void {
    $post = Post::factory()->published()->create(['slug' => 'real']);
    $other = Post::factory()->published()->create(['slug' => 'other']);
    $draft = Post::factory()->create(['slug' => 'draft-post']);

    beacon($post, id: (string) $other->id)->assertNotFound();
    beacon($post, id: 'abc')->assertNotFound();
    beacon($draft)->assertNotFound();
    test()->withHeaders(['User-Agent' => BEACON_UA])->post(route('blog.view', 'real'))->assertNotFound(); // no id
    test()->withHeaders(['User-Agent' => BEACON_UA])->post('/blog/missing/view', ['id' => '1'])->assertNotFound();

    expect(app(ViewCounter::class)->pending($post->id))->toBe(0)
        ->and(app(ViewCounter::class)->pending($other->id))->toBe(0);
});

it('throttles beacons per IP', function (): void {
    $post = Post::factory()->published()->create(['slug' => 'throttled']);

    for ($i = 0; $i < PostViewController::PER_MINUTE; $i++) {
        beacon($post)->assertNoContent();
    }
    beacon($post)->assertStatus(429);

    expect(app(ViewCounter::class)->pending($post->id))->toBe(PostViewController::PER_MINUTE);
});

it('renders the share copy module hooks and moves blog UI copy to lang', function (): void {
    Post::factory()->published()->create(['slug' => 'share-post']);

    $html = (string) $this->get('/blog/share-post')->assertOk()->getContent();

    expect($html)->toContain('data-module="share"')
        ->toContain('data-share-copied="'.__('blog.article.share.copied').'"')
        ->toContain('data-share-copy')
        ->toMatch('/<button[^>]*hidden[^>]*data-share-copy|<button[^>]*data-share-copy[^>]*hidden/')
        ->toContain('role="status"')
        ->toContain(__('blog.article.share.title'))
        ->toContain(__('blog.article.disclaimer'))
        ->not->toMatch('/<script(?![^>]*\bsrc=)[^>]*>(?!\s*\{)/') // no inline JS (JSON-LD data blocks only)
        ->not->toContain('blog.article.');
});
