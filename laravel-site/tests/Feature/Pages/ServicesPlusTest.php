<?php

declare(strict_types=1);

use App\Http\Controllers\PlusController;
use App\Http\Controllers\ServicesController;
use Database\Seeders\FaqSeeder;
use Illuminate\Support\Facades\Route;

/** @return list<array<string, mixed>> */
function servicesPlusGraph(string $html): array
{
    preg_match('~<script type="application/ld\+json"[^>]*>(.*?)</script>~s', $html, $m);

    return json_decode($m[1] ?? '{}', true, 512, JSON_THROW_ON_ERROR)['@graph'] ?? [];
}

/** @return array<string, mixed> */
function servicesPlusNode(string $html, string $idSuffix): array
{
    foreach (servicesPlusGraph($html) as $node) {
        if (str_ends_with((string) ($node['@id'] ?? ''), $idSuffix)) {
            return $node;
        }
    }

    return [];
}

dataset('services plus pages', [
    'services' => ['services', '/services', ServicesController::class],
    'plus' => ['plus', '/plus', PlusController::class],
]);

it('serves each page from its controller with one h1, SEO tags and no dead links or inline styles', function (string $name, string $path, string $controller): void {
    expect(Route::getRoutes()->getByName($name)?->getActionName())->toBe($controller);

    $html = $this->get($path)->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('rel="canonical"')
        ->toContain('<meta name="description"')
        ->toContain('id="download"')
        ->not->toContain('style="')
        ->not->toContain('href="#"')
        ->not->toContain('در حال آماده شدن');

    preg_match('~<title>(.*?)</title>~u', $html, $title);
    preg_match('~<meta name="description" content="([^"]*)"~u', $html, $description);
    expect(mb_strlen(html_entity_decode($title[1] ?? '')))->toBeGreaterThanOrEqual(30)->toBeLessThanOrEqual(60)
        ->and(mb_strlen(html_entity_decode($description[1] ?? '')))->toBeGreaterThanOrEqual(70)->toBeLessThanOrEqual(160);

    $types = array_column(servicesPlusGraph($html), '@type');
    expect($types)->toContain('Organization', 'WebSite', 'BreadcrumbList')
        ->not->toContain('Product', 'Offer', 'AggregateRating');
})->with('services plus pages');

it('renders services as a CollectionPage linking into the directory and the shop', function (): void {
    $html = $this->get('/services')->getContent();

    expect(servicesPlusNode($html, '#webpage')['@type'] ?? null)->toBe('CollectionPage')
        ->and($html)->toContain('مراقبت، برنامه‌ها و خدمات شهری، در یک جا')
        ->toContain('از سؤال ساده تا ویزیت')
        ->toContain('پزشک و ماما')
        ->toContain('برنامه‌های مراقبتی')
        ->toContain('href="'.route('directory.index').'"')
        ->toContain('href="'.route('shop.index').'"')
        ->not->toContain('href="tel:115"')
        ->toContain('همه خدمات در اپ ریتمی');
});

it('shows the four design plans with their prices and links «انتخاب» to the download band', function (): void {
    $html = $this->get('/plus')->getContent();

    expect(servicesPlusNode($html, '#webpage')['@type'] ?? null)->toBe('WebPage')
        ->and($html)->toContain('ریتمی رایگان است.')
        ->toContain('پلاس، عمق بیشتر.')
        ->toContain('۱۲۹٬۰۰۰')
        ->toContain('۱۱۶٬۰۰۰')
        ->toContain('۹۹٬۰۰۰')
        ->toContain('۸۲٬۵۰۰')
        ->toContain('۳۴۸٬۰۰۰ تومان، ۱۰٪ کمتر')
        ->toContain('۵۹۴٬۰۰۰ تومان، ۲۳٪ کمتر')
        ->toContain('۹۹۰٬۰۰۰ تومان، ۳۶٪ کمتر')
        ->toContain('پیشنهاد ما')
        ->toContain('<table')
        ->toContain('دارد')
        ->not->toContain('قیمت در اپ')
        ->and(substr_count($html, 'href="#download"'))->toBeGreaterThanOrEqual(5);
});

it('replaces the amounts with «قیمت در اپ» when the pricing switch is off', function (): void {
    __('plus.pricing'); // load the group first, so addLines only overrides this key
    app('translator')->addLines(['plus.pricing.show' => false], 'fa');

    $plans = PlusController::plans();

    expect($plans)->toHaveCount(4)
        ->and(array_column($plans, 'price'))->each->toBe('قیمت در اپ')
        ->and(array_column($plans, 'unit'))->each->toBe('')
        ->and($plans[1]['note'])->toBe('');

    $html = $this->get('/plus')->getContent();
    expect(substr_count($html, 'قیمت در اپ'))->toBe(4)
        ->and($html)->not->toContain('۱۲۹٬۰۰۰');
});

it('formats the plans from the lang amounts when the switch is on', function (): void {
    $plans = PlusController::plans();

    expect($plans[0]['price'])->toBe('۱۲۹٬۰۰۰')
        ->and($plans[0]['note'])->toBe('۱۲۹٬۰۰۰ تومان، هر وقت خواستی لغو کن')
        ->and($plans[2]['featured'])->toBeTrue()
        ->and($plans[3]['note'])->toBe('۹۹۰٬۰۰۰ تومان، ۳۶٪ کمتر');
});

it('renders the plus FAQ group as a visible grid with matching FAQPage JSON-LD', function (): void {
    $this->seed(FaqSeeder::class);

    $html = $this->get('/plus')->getContent();

    expect($html)->toContain('سؤالات متداول')
        ->toContain('اگر پلاس را لغو کنم، داده‌ام چه می‌شود؟')
        ->toContain('هشدارهای سلامت پولی است؟');

    $faq = servicesPlusNode($html, '#faq');
    expect($faq['@type'] ?? null)->toBe('FAQPage')
        ->and(array_column($faq['mainEntity'] ?? [], 'name'))->toContain('اگر پلاس را لغو کنم، داده‌ام چه می‌شود؟', 'پرداخت و لغو چطور است؟');
    foreach ($faq['mainEntity'] as $question) {
        expect($html)->toContain(e($question['name']));
    }
});

it('renders no FAQ block and no FAQPage node when the plus group is empty', function (): void {
    $html = $this->get('/plus')->getContent();

    expect($html)->not->toContain('plus-faq-title')
        ->and(servicesPlusNode($html, '#faq'))->toBe([]);
});
