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
        ->and($html)->toContain('ریتم بدنت را بشناس')
        ->toContain('id="stages"')
        ->toContain('id="features"')
        ->toContain('id="download"')
        ->toContain('مرحله‌ات را انتخاب کن')
        ->toContain('چرخه‌ات را بهتر بشناس، مراقبت از خودت را ساده‌تر کن')
        ->toContain('وقتی مسیرت تغییر می‌کند')
        ->toContain('کمتر از یک دقیقه تا اولین ثبت')
        ->toContain('لازم نیست هر ماه توضیح بدهی')
        ->toContain('مال خودت است')
        ->toContain('ریتمی را رایگان نصب کن')
        ->toContain(route('stage.teen'))
        ->not->toContain('style="');
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

it('hides the readings block while the magazine is empty and shows the six newest posts', function (): void {
    expect($this->get('/')->getContent())->not->toContain('تازه‌ترین مقاله‌ها');

    Post::factory()->published(now()->subDays(3))->create(['title' => 'مقاله قدیمی‌تر']);
    $newest = Post::factory()->published(now()->subHour())->count(6)->create();

    $html = $this->get('/')->getContent();
    expect($html)->toContain('تازه‌ترین مقاله‌ها')
        ->toContain('همه مقاله‌ها')
        ->toContain(e($newest[0]->title))
        ->toContain(e($newest[5]->title))
        ->not->toContain('مقاله قدیمی‌تر');
});

it('renders hero store buttons only for links set in settings', function (): void {
    expect($this->get('/')->getContent())->not->toContain('https://cafebazaar.ir/app/ritme');

    app(SettingsRepository::class)->put(SettingGroup::AppLinks, ['bazaar' => 'https://cafebazaar.ir/app/ritme']);

    expect($this->get('/')->getContent())->toContain('https://cafebazaar.ir/app/ritme');
});

it('keeps the copy inside the content red lines', function (): void {
    $copy = json_encode(trans('home'), JSON_UNESCAPED_UNICODE);

    // Whole words only: «احتمالی» (probabilistic language) is welcome, «حتما» is not.
    foreach (['حتماً', 'حتما', 'قطعاً', 'قطعا', 'دقیق‌ترین', 'تضمینی'] as $word) {
        expect(preg_match('/(?<![\\p{L}\\x{200C}])'.$word.'/u', (string) $copy))->toBe(0, $word);
    }
});
