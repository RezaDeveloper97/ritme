<?php

declare(strict_types=1);

use App\Domain\Content\Stages\Pregnancy;
use App\Domain\Content\Stages\StagePageBuilder;
use App\Domain\Content\Stages\StageRegistry;
use App\Domain\Content\Stages\Ttc;
use Database\Seeders\SettingsSeeder;

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
    config(['app.url' => 'https://ritme.test']);
});

it('discovers the ttc and pregnancy stage definitions', function (): void {
    $classes = array_map(static fn (object $definition): string => $definition::class, app(StageRegistry::class)->all());

    expect($classes)->toContain(Ttc::class)->toContain(Pregnancy::class);
});

it('renders /ttc with the fertility, companion and IVF/IUI splits', function (): void {
    $html = $this->get('/ttc')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('<title>اقدام به بارداری با تقویم باروری، تست LH و IVF — ریتمی</title>')
        ->and($html)->toContain('<link rel="canonical" href="https://ritme.test/ttc">')
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and(substr_count($html, 'id="how"'))->toBe(1)
        ->and($html)->toContain('پنجره باروری، از داده خودت')
        ->and($html)->toContain('این مسیر را تنها نرو')
        ->and($html)->toContain('اگر IVF یا IUI داری')
        ->and($html)->toContain('همدم سارا')
        ->and($html)->toContain('href="/tools#fertility"')
        ->and($html)->toContain('تحلیل آزمایش')
        ->and($html)->toContain('href="tel:115"')
        ->and($html)->toContain('سؤال‌های رایج درباره اقدام به بارداری')
        ->and($html)->toContain('data-faq-group="stage-ttc"')
        ->and(substr_count($html, '<details'))->toBe(3)
        ->and(substr_count($html, 'shadow-phone'))->toBe(4)
        ->and($html)->toMatch('~href="/ttc" aria-current="page"[^>]*border-stage-ttc~');
});

it('renders /pregnancy with week-by-week, appointments and birth preparation', function (): void {
    $html = $this->get('/pregnancy')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('<title>بارداری هفته‌به‌هفته، ویزیت‌ها و ساک بیمارستان — ریتمی</title>')
        ->and($html)->toContain('<link rel="canonical" href="https://ritme.test/pregnancy">')
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and(substr_count($html, 'id="how"'))->toBe(1)
        ->and($html)->toContain('هر هفته، همان چیزی که لازم است')
        ->and($html)->toContain('هیچ نوبتی از قلم نیفتد')
        ->and($html)->toContain('ساک بیمارستان، برنامه زایمان و لیست سیسمونی')
        ->and($html)->toContain('هفته ۲۴ بارداری')
        // tool cards → stable anchors of the /tools page (L3-07)
        ->and($html)->toContain('href="/tools#due-date"')
        ->and($html)->toContain('href="/tools#hospital-bag"')
        ->and($html)->toContain('href="/tools#sisemoni"')
        ->and($html)->toContain('href="/directory"')
        ->and($html)->toContain('کاهش محسوس حرکت جنین')
        ->and($html)->toContain('سؤال‌های رایج درباره بارداری')
        ->and($html)->toContain('data-faq-group="stage-pregnancy"')
        ->and(substr_count($html, '<details'))->toBe(3)
        ->and(substr_count($html, 'shadow-phone'))->toBe(4)
        ->and($html)->toMatch('~href="/pregnancy" aria-current="page"[^>]*border-stage-pregnancy~');
});

it('builds both stages with six pregnancy tools and three help cards each', function (): void {
    $builder = app(StagePageBuilder::class);
    $ttc = $builder->build(new Ttc);
    $pregnancy = $builder->build(new Pregnancy);

    expect($ttc->features)->toHaveCount(3)
        ->and($ttc->tools)->toHaveCount(3)
        ->and($ttc->help)->toHaveCount(3)
        ->and($pregnancy->features)->toHaveCount(3)
        ->and($pregnancy->tools)->toHaveCount(6)
        ->and($pregnancy->help)->toHaveCount(3)
        ->and($pregnancy->hero->floatCards)->toHaveCount(2)
        ->and($ttc->seoTitle)->not->toBe($pregnancy->seoTitle)
        ->and($ttc->seoDescription)->not->toBe($pregnancy->seoDescription);
});

it('describes both pages in JSON-LD with breadcrumbs and the FAQ', function (string $path, string $name): void {
    $html = $this->get($path)->assertOk()->getContent();
    preg_match('~<script type="application/ld\+json">(.*?)</script>~s', $html, $m);
    $graph = [];
    foreach (json_decode($m[1] ?? '{}', true, 512, JSON_THROW_ON_ERROR)['@graph'] ?? [] as $node) {
        $graph[$node['@type']] = $node;
    }

    expect(array_column($graph['BreadcrumbList']['itemListElement'], 'name'))->toBe(['خانه', 'مرحله‌ها', $name])
        ->and($graph['FAQPage']['@id'])->toBe('https://ritme.test'.$path.'#faq')
        ->and($graph['FAQPage']['mainEntity'])->toHaveCount(3)
        ->and($graph)->toHaveKey('MobileApplication');
})->with([
    ['/ttc', 'اقدام به بارداری'],
    ['/pregnancy', 'بارداری'],
]);
