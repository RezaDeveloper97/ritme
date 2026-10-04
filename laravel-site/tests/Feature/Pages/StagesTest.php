<?php

declare(strict_types=1);

use App\Domain\Blog\Enums\LifeStage;
use App\Domain\Blog\Models\Post;
use App\Domain\Content\Stages\Cycle;
use App\Domain\Content\Stages\StagePageBuilder;
use App\Domain\Content\Stages\StageRegistry;
use App\Domain\Seo\Models\SeoMeta;
use Database\Seeders\SettingsSeeder;

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
    config(['app.url' => 'https://ritme.test']);
});

/**
 * @return array<string, mixed>
 */
function stageGraph(string $html): array
{
    preg_match('~<script type="application/ld\+json">(.*?)</script>~s', $html, $m);
    $graph = [];
    foreach (json_decode($m[1] ?? '{}', true, 512, JSON_THROW_ON_ERROR)['@graph'] ?? [] as $node) {
        $graph[$node['@type']] = $node;
    }

    return $graph;
}

it('renders /cycle on the stage template with the design sections', function (): void {
    $html = $this->get('/cycle')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('الگوی بدن خودت')
        ->and($html)->toContain('<title>پیگیری چرخه و پریود با پیش‌بینی و سطح اطمینان — ریتمی</title>')
        ->and($html)->toContain('<link rel="canonical" href="https://ritme.test/cycle">')
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        // hero «چطور کار می‌کند؟» → first feature split
        ->and($html)->toContain('href="#how"')
        ->and(substr_count($html, 'id="how"'))->toBe(1)
        ->and(substr_count($html, 'id="download"'))->toBe(1)
        ->and(substr_count($html, '<h2'))->toBeGreaterThanOrEqual(7)
        // blocks
        ->and($html)->toContain('کارهای کوچک، آمادگی بیشتر')
        ->and($html)->toContain('وقتی کمک بیشتری لازم داری')
        ->and($html)->toContain('href="tel:115"')
        ->and($html)->toContain('سؤال‌های رایج درباره پیگیری چرخه')
        ->and(substr_count($html, '<details'))->toBe(3)
        ->and($html)->toContain('data-faq-group="stage-cycle"')
        ->and($html)->toContain('href="/tools#fertility"');
});

it('renders the six stage pills with the current stage marked', function (): void {
    $html = $this->get('/cycle')->assertOk()->getContent();

    expect(preg_match('~<nav aria-label="مرحله‌های زندگی".*?</nav>~s', $html, $nav))->toBe(1);
    expect(substr_count($nav[0], '<a '))->toBe(6)
        ->and(substr_count($nav[0], 'aria-current="page"'))->toBe(1)
        ->and($nav[0])->toMatch('~href="/cycle" aria-current="page"[^>]*border-stage-cycle~');
});

it('keeps phone mock-ups and float cards out of the accessibility tree', function (): void {
    $html = $this->get('/cycle')->assertOk()->getContent();

    // hero phone column + 3 feature mock-up columns, each wrapped in aria-hidden
    expect(substr_count($html, 'shadow-phone'))->toBe(4)
        ->and(preg_match_all('~<div aria-hidden="true"[^>]*(?:h-165|bg-\[radial-gradient)~', $html))->toBe(4);
});

it('describes the page in JSON-LD: breadcrumbs, FAQ and the app', function (): void {
    $graph = stageGraph($this->get('/cycle')->assertOk()->getContent());

    expect(array_column($graph['BreadcrumbList']['itemListElement'], 'name'))->toBe(['خانه', 'مرحله‌ها', 'پیگیری چرخه'])
        ->and($graph['BreadcrumbList']['itemListElement'][1]['item'])->toBe('https://ritme.test/#stages')
        ->and($graph['FAQPage']['@id'])->toBe('https://ritme.test/cycle#faq')
        ->and($graph['FAQPage']['mainEntity'])->toHaveCount(3)
        ->and($graph['FAQPage']['mainEntity'][0]['name'])->toBe('اگر چرخه‌ام نامنظم باشد، ریتمی به دردم می‌خورد؟')
        ->and($graph)->toHaveKey('MobileApplication')
        ->and($graph)->toHaveKey('WebPage');
});

it('lets the admin SEO title and description win over the stage copy', function (): void {
    SeoMeta::query()->create(['route_name' => 'stage.cycle', 'title' => 'عنوان مدیر برای صفحه پیگیری چرخه', 'description' => str_repeat('توضیح مدیر ', 10)]);

    $html = $this->get('/cycle')->assertOk()->getContent();

    expect($html)->toContain('<title>عنوان مدیر برای صفحه پیگیری چرخه — ریتمی</title>')
        ->and($html)->toContain('توضیح مدیر');
});

it('lists stage posts first and tops the readings up with the newest posts', function (): void {
    $cycle = Post::factory()->published(now()->subDays(3))->create(['life_stage' => LifeStage::Cycle, 'title' => 'نوشته چرخه']);
    Post::factory()->published(now()->subDay())->create(['life_stage' => LifeStage::Pregnancy, 'title' => 'نوشته بارداری']);
    Post::factory()->published(now()->subDays(2))->create(['life_stage' => LifeStage::Menopause, 'title' => 'نوشته یائسگی']);
    Post::factory()->published(now()->subDays(9))->create(['life_stage' => LifeStage::Teen, 'title' => 'نوشته قدیمی']);

    $readings = app(StagePageBuilder::class)->build(new Cycle)->readings;

    expect(array_map(static fn ($post): string => $post->title, $readings))->toBe(['نوشته چرخه', 'نوشته بارداری', 'نوشته یائسگی'])
        ->and($readings[0]->id)->toBe($cycle->id);

    $html = $this->get('/cycle')->assertOk()->getContent();
    expect($html)->toContain('برای همین مرحله')->and($html)->toContain('نوشته چرخه');
});

it('hides the readings block while the magazine is empty', function (): void {
    expect($this->get('/cycle')->assertOk()->getContent())->not->toContain('برای همین مرحله');
});

it('builds every defined stage without missing copy', function (): void {
    $stages = app(StageRegistry::class)->all();
    expect($stages)->not->toBeEmpty();

    foreach ($stages as $definition) {
        $page = app(StagePageBuilder::class)->build($definition);

        expect($page->features)->not->toBeEmpty()
            ->and($page->features[0]->anchor)->toBe('how')
            ->and($page->tools)->not->toBeEmpty()
            ->and($page->faq)->toHaveCount(3)
            ->and($page->faqTitle)->not->toBe($page->name)
            ->and(mb_strlen($page->seoTitle.' — ریتمی'))->toBeBetween(30, 60)
            ->and(mb_strlen($page->seoDescription))->toBeBetween(70, 160);
    }
});

it('serves stages without a definition as the noindex placeholder', function (LifeStage $stage): void {
    if (class_exists(StageRegistry::classFor($stage))) {
        $this->get('/'.$stage->value)->assertOk();

        return;
    }

    $html = $this->get('/'.$stage->value)->assertOk()->getContent();
    expect($html)->toContain('noindex')->and(substr_count($html, '<h1'))->toBe(1);
})->with(LifeStage::cases());

it('keeps the stage copy inside the content red lines', function (): void {
    foreach (glob(lang_path('fa/stages/*.php')) ?: [] as $file) {
        // strings only — the file header names the forbidden words
        $copy = implode(' ', array_map(static fn (array $t): string => $t[1], array_filter(token_get_all((string) file_get_contents($file)), static fn ($t): bool => is_array($t) && $t[0] === T_CONSTANT_ENCAPSED_STRING)));
        foreach (['حتماً', 'حتما ', 'قطعاً', 'دقیق‌ترین', 'تضمینی', 'تشخیص می‌دهیم'] as $word) {
            expect(str_contains($copy, $word))->toBeFalse("«{$word}» in ".basename($file));
        }
    }
});
