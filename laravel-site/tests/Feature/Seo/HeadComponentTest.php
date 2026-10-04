<?php

declare(strict_types=1);

use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Enums\SettingGroup;
use Illuminate\Support\Facades\Route;
use Tests\Feature\Seo\SeoFixtures;

beforeEach(function (): void {
    SeoFixtures::boot();
});

it('renders a complete head from <x-seo.head/> with zero per-view boilerplate', function (): void {
    app(SettingsRepository::class)->put(SettingGroup::Seo, ['verification' => ['google' => 'g-token']]);
    SeoFixtures::routeMeta('fixture.cycle', [
        'title' => 'پیگیری چرخه قاعدگی', 'description' => 'توضیح "ویژه" صفحه چرخه & دوره',
    ]);
    Route::view('/cycle', 'seo-fixtures::page')->name('fixture.cycle');
    app('router')->getRoutes()->refreshNameLookups();

    $html = $this->get('/cycle?utm_source=ig')->assertOk()->getContent();

    expect($html)
        ->toContain('<title>پیگیری چرخه قاعدگی — ریتمی</title>')
        ->toContain('<meta name="description" content="توضیح &quot;ویژه&quot; صفحه چرخه &amp; دوره">')
        ->toContain('<link rel="canonical" href="https://ritme.test/cycle">')
        ->toContain('<meta name="robots" content="noindex,follow">')
        ->toContain('<meta property="og:locale" content="fa_IR">')
        ->toContain('<meta property="og:site_name" content="ریتمی">')
        ->toContain('<meta property="og:type" content="website">')
        ->toContain('<meta property="og:url" content="https://ritme.test/cycle">')
        ->toContain('<meta property="og:title" content="پیگیری چرخه قاعدگی — ریتمی">')
        ->toContain('<meta property="og:description"')
        ->toContain('<meta name="twitter:card" content="summary">')
        ->toContain('<meta name="twitter:title"')
        ->toContain('<meta name="google-site-verification" content="g-token">')
        ->not->toContain('rel="prev"')
        ->not->toContain('rel="next"')
        ->toContain('<link rel="alternate" type="application/rss+xml"') // blog.feed (L4-04)
        ->and(substr_count($html, '<title>'))->toBe(1);
});

it('renders pushed JSON-LD inside the head', function (): void {
    Route::get('/jsonld', fn () => view('seo-fixtures::page'));

    view()->startPush('seo.jsonld', '<script type="application/ld+json">{"@graph":[]}</script>');

    expect($this->get('/jsonld')->getContent())->toContain('<script type="application/ld+json">{"@graph":[]}</script>');
});
