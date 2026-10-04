<?php

declare(strict_types=1);

use App\Domain\Blog\Models\Post;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Enums\SettingGroup;
use App\Http\Controllers\HomeController;
use Illuminate\Support\Facades\Route;

/** @return array<string, mixed> */
function homeJsonLd(string $html): array
{
    preg_match('~<script type="application/ld\+json"[^>]*>(.*?)</script>~s', $html, $m);

    return json_decode($m[1] ?? '{}', true, 512, JSON_THROW_ON_ERROR);
}

it('serves the home route from HomeController', function (): void {
    expect(Route::getRoutes()->getByName('home')?->getActionName())->toBe(HomeController::class);
});

it('renders the home page with one h1 and the design sections', function (): void {
    $html = $this->get('/')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('کنار تو')
        ->toContain('id="stages"')
        ->toContain('id="how"')
        ->toContain('id="download"')
        ->toContain('مرحله‌ات را انتخاب کن')
        ->toContain('یک اپ، پنج بخش ساده')
        ->toContain('وقتی داده کم است، می‌گوییم «هنوز مطمئن نیستیم»')
        ->toContain('خانواده‌ات هم می‌تواند کنارت باشد')
        ->toContain('وقتی به کمک بیشتری نیاز داری')
        ->toContain('داده‌ات مال خودت است')
        ->toContain('بدون نصب هم می‌توانی شروع کنی')
        ->toContain('آگاهی، حق همه زنان ایران است')
        ->toContain(route('stage.teen'))
        ->toContain(route('tools').'#due-date')
        ->not->toContain('style="')
        ->not->toContain('loading="lazy"');
});

it('has the design title, description, canonical and the app in the JSON-LD graph', function (): void {
    $html = $this->get('/')->getContent();

    expect($html)->toContain('<title>ریتمی — همراه سلامت زنان، از اولین پریود تا یائسگی</title>')
        ->toContain('ریتمی اپ پیگیری چرخه، بارداری، مادری و یائسگی است')
        ->toContain('rel="canonical"');

    $types = array_column(homeJsonLd($html)['@graph'] ?? [], '@type');
    expect($types)->toContain('Organization', 'WebSite', 'WebPage', 'MobileApplication')
        ->not->toContain('BreadcrumbList', 'FAQPage');
});

it('hides the readings block while the magazine is empty and shows the newest posts', function (): void {
    expect($this->get('/')->getContent())->not->toContain('خواندنی‌های این هفته');

    Post::factory()->published(now()->subDays(3))->create(['title' => 'مقاله قدیمی‌تر']);
    $newest = Post::factory()->published(now()->subHour())->count(3)->create();

    $html = $this->get('/')->getContent();
    expect($html)->toContain('خواندنی‌های این هفته')
        ->toContain('همه مقاله‌ها')
        ->toContain(e($newest[0]->title))
        ->not->toContain('مقاله قدیمی‌تر');
});

it('leaves the FAQ to L3-09 and renders no hard-coded questions', function (): void {
    expect($this->get('/')->getContent())->not->toContain('ریتمی رایگان است؟')->not->toContain('قبل از نصب');
});

it('renders store badges and the QR only from settings', function (): void {
    expect($this->get('/')->getContent())->not->toContain('کافه‌بازار');

    app(SettingsRepository::class)->put(SettingGroup::AppLinks, ['bazaar' => 'https://cafebazaar.ir/app/ritme']);

    expect($this->get('/')->getContent())->toContain('https://cafebazaar.ir/app/ritme')->toContain('کد QR');
});

it('keeps the copy inside the content red lines', function (): void {
    $copy = json_encode(trans('home'), JSON_UNESCAPED_UNICODE);

    // Whole words only: «احتمالی» (probabilistic language) is welcome, «حتما» is not.
    foreach (['حتماً', 'حتما', 'قطعاً', 'قطعا', 'دقیق‌ترین', 'تضمینی'] as $word) {
        expect(preg_match('/(?<![\\p{L}\\x{200C}])'.$word.'/u', (string) $copy))->toBe(0, $word);
    }
});
