<?php

declare(strict_types=1);

use Illuminate\Support\Facades\Route;
use Tests\Feature\Seo\SeoFixtures;

beforeEach(function (): void {
    SeoFixtures::boot();

    SeoFixtures::routeMeta('fixture.good', [
        'title' => 'راهنمای پیگیری چرخه قاعدگی',
        'description' => 'ریتمی کمک می‌کند چرخه‌ات را بشناسی؛ با زبان احتمالی، بدون ترساندن و با داده‌ای که پیش خودت می‌ماند.',
    ]);
    SeoFixtures::routeMeta('fixture.other', [
        'title' => 'همراه دوران بارداری، هفته به هفته',
        'description' => 'بارداری را هفته به هفته با ریتمی دنبال کن؛ تغییرات بدن، یادآور مراقبت‌ها و پاسخ سؤال‌های رایج در یک جا.',
    ]);
    SeoFixtures::routeMeta('fixture.bad', ['title' => 'کوتاه', 'description' => 'کوتاه']);

    Route::view('/fixture/good', 'seo-fixtures::page')->name('fixture.good');
    Route::view('/fixture/other', 'seo-fixtures::page')->name('fixture.other');
    Route::view('/fixture/bad', 'seo-fixtures::page', ['broken' => true])->name('fixture.bad');
    Route::view('/fixture/dupe', 'seo-fixtures::page')->name('fixture.dupe');
    Route::get('/fixture/item/{slug}', fn () => 'x')->name('fixture.item');
    Route::get('/fixture/json', fn () => response()->json([]))->name('fixture.json');
    app('router')->getRoutes()->refreshNameLookups();
});

it('passes clean pages (og:image is a warning) and exits zero', function (): void {
    $this->artisan('seo:audit', ['--path' => ['/fixture/good', '/fixture/other']])
        ->expectsOutputToContain('✔ /fixture/good')
        ->expectsOutputToContain('warning [og.image]')
        ->expectsOutputToContain('2 page(s), 0 error(s), 2 warning(s).')
        ->assertExitCode(0);
});

it('treats warnings as errors with --strict', function (): void {
    $this->artisan('seo:audit', ['--path' => ['/fixture/good'], '--strict' => true])->assertExitCode(1);
});

it('fails on broken pages and reports each rule', function (): void {
    $this->artisan('seo:audit', ['--path' => ['/fixture/bad']])
        ->expectsOutputToContain('✘ /fixture/bad')
        ->expectsOutputToContain('[title.length]')
        ->expectsOutputToContain('[description.length]')
        ->expectsOutputToContain('[h1.count]')
        ->expectsOutputToContain('[img.attributes]')
        ->expectsOutputToContain('[link.hash]')
        ->assertExitCode(1);
});

it('flags duplicate titles across indexable pages in production', function (): void {
    config(['app.env' => 'production']);
    SeoFixtures::routeMeta('fixture.dupe', [
        'title' => 'راهنمای پیگیری چرخه قاعدگی',
        'description' => 'متن دیگری برای همین صفحه که طولش کافی است تا قانون طول توضیحات را رعایت کند و تکراری نباشد.',
    ]);

    $this->artisan('seo:audit', ['--path' => ['/fixture/good', '/fixture/dupe']])
        ->expectsOutputToContain('[title.duplicate]')
        ->assertExitCode(1);
});

it('crawls every parameterless GET route and skips non-HTML responses', function (): void {
    $this->artisan('seo:audit')
        ->expectsOutputToContain('/fixture/good')
        ->expectsOutputToContain('/fixture/json skipped')
        ->doesntExpectOutputToContain('/fixture/item')
        ->doesntExpectOutputToContain(' /up')
        ->assertExitCode(1); // fixture.bad (and the welcome placeholder) fail
});
