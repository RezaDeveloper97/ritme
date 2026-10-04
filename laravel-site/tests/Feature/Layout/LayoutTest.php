<?php

declare(strict_types=1);

use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Enums\SettingGroup;
use App\Domain\Settings\Models\Setting;
use App\Support\Cache\NamespaceVersions;
use Database\Seeders\SettingsSeeder;
use Illuminate\Support\Facades\Blade;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Route;
use Illuminate\Support\Facades\View;

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
    config(['app.url' => 'https://ritme.test']);
    View::addNamespace('layout-fixtures', __DIR__.'/fixtures');
});

/** Writes a setting straight to the table (no observer, so no namespace bump). */
function rawSetting(SettingGroup $group, string $key, mixed $value): void
{
    DB::table((new Setting)->getTable())
        ->where('group', $group->value)->where('key', $key)
        ->update(['value' => json_encode($value)]);
}

it('renders the preview layouts as valid shells', function (string $variant): void {
    $html = $this->get("/_preview/layout/{$variant}")->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('<html lang="fa" dir="rtl">')
        ->and($html)->toMatch('/<main id="main"[^>]*>/')
        ->and($html)->toContain('href="#main"')
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and(substr_count($html, '<header'))->toBe(1)
        ->and(substr_count($html, '<footer'))->toBe(1)
        ->and($html)->toContain('<nav aria-label="منوی اصلی"');

    // No external hosts in any src/href (links to this site only; JSON-LD @context is not a request).
    preg_match_all('/\s(?:src|href|action|srcset)="((?:https?:)?\/\/[^"\/]+)/i', $html, $matches);
    $hosts = array_unique(array_map(static fn (string $url): string => (string) parse_url(str_starts_with($url, '//') ? 'http:'.$url : $url, PHP_URL_HOST), $matches[1]));
    expect(array_values(array_diff($hosts, ['ritme.test', 'localhost', '127.0.0.1'])))->toBe([]);
})->with(['dark', 'light']);

it('renders the burger as an accessible disclosure for the hidden mobile menu', function (): void {
    $html = $this->get('/_preview/layout/light')->assertOk()->getContent();

    expect(preg_match('/<button[^>]*data-module="menu"[^>]*>/', $html, $button))->toBe(1);
    expect($button[0])
        ->toContain('type="button"')
        ->toContain('aria-controls="site-menu"')
        ->toContain('aria-expanded="false"')
        ->toContain('aria-label="باز کردن منو"')
        ->toContain('data-label-close="بستن منو"');
    expect($html)->toMatch('/<div id="site-menu" hidden[\s>]/')
        ->toContain('<nav aria-label="منوی موبایل">');
});

it('marks the active nav item and uses the dark block on dark pages', function (): void {
    $dark = $this->get('/_preview/layout/dark')->getContent();
    $light = $this->get('/_preview/layout/light')->getContent();

    expect($dark)->toMatch('/<a href="\/"\s+aria-current="page"[^>]*font-extrabold/')
        ->toContain('bg-night bg-hero-glow')
        ->and($light)->not->toContain('aria-current="page"  class="flex items-center gap-1')
        ->and($light)->not->toContain('bg-hero-glow')
        ->and($light)->toContain('border-b border-line bg-surface');
});

it('renders footer columns as labelled navs with the corrected links', function (): void {
    $html = $this->get('/_preview/layout/light')->getContent();

    foreach (['مرحله‌ها', 'خدمات', 'ابزارها', 'ریتمی', 'اعتماد'] as $title) {
        expect($html)->toContain('<nav aria-label="'.$title.'"');
    }
    expect($html)->toContain('href="/tools#due-date"')
        ->toContain('href="/tools#fertility"')
        ->toContain('href="/tools#hospital-bag"')
        ->toContain('href="/tools#sisemoni"')
        ->toContain('href="/terms"')
        ->toContain('href="tel:115"');
});

it('registers breadcrumbs before the head renders, so the JSON-LD has the BreadcrumbList', function (): void {
    $html = $this->get('/_preview/layout/light')->getContent();

    $script = strpos($html, '"@type":"BreadcrumbList"');
    expect($script)->not->toBeFalse()
        ->and($script)->toBeLessThan(strpos($html, '</head>'))
        ->and($html)->toContain('"name":"پیش‌نمایش قالب"')
        ->and($html)->toContain('<nav aria-label="مسیر"')
        ->and($html)->toMatch('/<span\s+aria-current="page"\s+class="text-ink">پیش‌نمایش قالب<\/span>/u');
});

it('builds breadcrumbs from the StaticPage registry for the current route', function (): void {
    Route::view('/faq', 'layout-fixtures::registry-page')->name('faq');
    app('router')->getRoutes()->refreshNameLookups();

    $html = $this->get('/faq')->assertOk()->getContent();
    $head = substr($html, 0, (int) strpos($html, '</head>'));

    expect($head)->toContain('"@type":"BreadcrumbList"')
        ->toContain('"name":"خانه"')
        ->toContain('"name":"سؤالات متداول"')
        ->and($html)->toMatch('/<span\s+aria-current="page"\s+class="text-ink">سؤالات متداول<\/span>/u')
        // faq has an app CTA: «دانلود اپ» stays on the page; no active nav item.
        ->and($html)->toContain('href="#download"')
        ->and($html)->not->toContain('aria-current="true"');
});

it('caches the header fragment until the settings namespace version changes', function (): void {
    $render = static fn (): string => Blade::render('<x-layout.header variant="light" route="faq"/>');

    $before = $render();
    expect($before)->not->toContain('https://web.ritme.example');

    rawSetting(SettingGroup::AppLinks, 'web_app', 'https://web.ritme.example');
    expect($render())->toBe($before); // same key → cached fragment

    app(NamespaceVersions::class)->bump('settings');
    expect($render())->toContain('href="https://web.ritme.example"');
});

it('caches the footer fragment until the settings namespace version changes', function (): void {
    $render = static fn (): string => Blade::render('<x-layout.footer/>');

    $before = $render();
    expect($before)->not->toContain('اینستاگرام');

    rawSetting(SettingGroup::Social, 'instagram', 'https://instagram.com/ritme');
    expect($render())->toBe($before);

    app(NamespaceVersions::class)->bump('settings');
    expect($render())->toContain('href="https://instagram.com/ritme"')->toContain('اینستاگرام');
});

it('keys the header fragment by variant and active route', function (): void {
    $faq = Blade::render('<x-layout.header variant="light" route="faq"/>');
    $home = Blade::render('<x-layout.header variant="dark" route="home"/>');

    expect($faq)->toContain('border-b border-line bg-surface')->not->toContain('aria-current="page"')
        ->and($home)->not->toContain('bg-surface')->toContain('aria-current="page"');
});

it('updates the cached shell when settings change through the model (observer bump)', function (): void {
    Blade::render('<x-layout.footer/>');

    app(SettingsRepository::class)->put(SettingGroup::AppLinks, ['bazaar' => 'https://cafebazaar.ir/app/ir.ritmeapp.ritme']);

    expect(Blade::render('<x-layout.footer/>'))->toContain('href="https://cafebazaar.ir/app/ir.ritmeapp.ritme"')->toContain('کافه‌بازار');
});

it('reduces admin enamad HTML to allowed tags in the footer', function (): void {
    rawSetting(SettingGroup::Legal, 'enamad_html', '<a href="https://trustseal.enamad.ir/?id=1" onclick="x()"><img src="https://trustseal.enamad.ir/logo.png"></a><script>alert(1)</script>');
    app(NamespaceVersions::class)->bump('settings');

    $html = Blade::render('<x-layout.footer/>');

    expect($html)->toContain('<a href="https://trustseal.enamad.ir/?id=1" rel="noopener nofollow">نماد اعتماد الکترونیکی</a>')
        ->not->toContain('trustseal.enamad.ir/logo.png')
        ->not->toContain('<script')
        ->not->toContain('onclick');
});
