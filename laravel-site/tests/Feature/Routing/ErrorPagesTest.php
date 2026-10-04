<?php

declare(strict_types=1);

use Database\Seeders\SettingsSeeder;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Route;

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
    config(['app.debug' => false]);

    Route::get('/_test/abort/{code}', static fn (string $code) => abort((int) $code))->middleware('web');
    Route::get('/_test/boom', static fn () => throw new RuntimeException('boom'));
});

it('renders the layout-based error pages: status, one h1, noindex, helpful links', function (int $code, string $h1): void {
    $html = $this->get("/_test/abort/{$code}")->assertStatus($code)->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain($h1)
        ->and($html)->toContain('<meta name="robots" content="noindex')
        ->and($html)->toContain('<header')
        ->and($html)->toContain('<footer')
        ->and($html)->toContain('href="/"')
        ->and($html)->toContain('href="/cycle"')
        ->and($html)->toContain('href="/faq"')
        ->and($html)->not->toMatch('/\sstyle\s*=/i');
})->with([
    [404, 'این صفحه پیدا نشد'],
    [410, 'این صفحه دیگر وجود ندارد'],
    [419, 'زمان این صفحه تمام شده است'],
    [429, 'درخواست‌ها کمی زیاد شد'],
]);

it('renders unknown URLs with the 404 page', function (): void {
    $html = $this->get('/does-not-exist')->assertNotFound()->getContent();

    expect($html)->toContain('این صفحه پیدا نشد')
        ->and($html)->toContain('<meta name="robots" content="noindex')
        ->and($html)->toContain('۴۰۴');
});

it('renders 500 and 503 standalone, without touching the database', function (string $url, int $code, string $h1): void {
    DB::enableQueryLog();
    $html = $this->get($url)->assertStatus($code)->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain($h1)
        ->and($html)->toContain('<meta name="robots" content="noindex, nofollow">')
        ->and($html)->toContain('<html lang="fa" dir="rtl">')
        ->and(DB::getQueryLog())->toBe([]);
})->with([
    ['/_test/boom', 500, 'مشکلی پیش آمد'],
    ['/_test/abort/503', 503, 'ریتمی در حال به‌روزرسانی است'],
]);

it('does not query the database for a 404 once settings are cached', function (): void {
    $this->get('/warm-up-404')->assertNotFound();

    DB::enableQueryLog();
    $this->get('/another-missing-page')->assertNotFound();

    expect(DB::getQueryLog())->toBe([]);
});

it('shows the search link only once the search route exists', function (): void {
    expect($this->get('/nope')->getContent())->not->toContain('جست‌وجو در ریتمی');

    Route::get('/search', static fn () => 'search')->name('search');
    app('router')->getRoutes()->refreshNameLookups();

    expect($this->get('/nope')->getContent())->toContain('جست‌وجو در ریتمی')->toContain('href="/search"');
});
