<?php

declare(strict_types=1);

use App\Domain\Content\Stages\Menopause;
use App\Domain\Content\Stages\Postpartum;
use App\Domain\Content\Stages\StagePageBuilder;
use App\Domain\Content\Stages\Teen;
use Database\Seeders\SettingsSeeder;

/*
 * L3-05 — /postpartum, /menopause, /teen on the shared stage template (the template itself: StagesTest).
 */

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
    config(['app.url' => 'https://ritme.test']);
});

/**
 * @return array<string, mixed>
 */
function otherStageGraph(string $html): array
{
    preg_match('~<script type="application/ld\+json">(.*?)</script>~s', $html, $m);
    $graph = [];
    foreach (json_decode($m[1] ?? '{}', true, 512, JSON_THROW_ON_ERROR)['@graph'] ?? [] as $node) {
        $graph[$node['@type']] = $node;
    }

    return $graph;
}

it('renders each stage page with its own SEO, one h1 and the design sections', function (string $path, string $title, string $h1, int $splits, bool $help, string $faqTitle): void {
    $html = $this->get($path)->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain($h1)
        ->and($html)->toContain("<title>{$title} — ریتمی</title>")
        ->and($html)->toContain('<link rel="canonical" href="https://ritme.test'.$path.'">')
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and($html)->not->toContain('stages/')
        ->and(substr_count($html, 'id="how"'))->toBe(1)
        ->and(substr_count($html, 'id="download"'))->toBe(1)
        // hero phone + one phone per feature split, all decorative
        ->and(substr_count($html, 'shadow-phone'))->toBe(1 + $splits)
        ->and(preg_match_all('~<div aria-hidden="true"[^>]*(?:h-165|bg-\[radial-gradient)~', $html))->toBe(1 + $splits)
        ->and($html)->toContain('کارهای کوچک، آمادگی بیشتر')
        ->and(str_contains($html, 'وقتی کمک بیشتری لازم داری'))->toBe($help)
        ->and($html)->toContain('href="tel:115"')
        ->and($html)->toContain($faqTitle)
        ->and(substr_count($html, '<details'))->toBe(3);

    expect($html)->toMatch('~href="'.preg_quote($path, '~').'" aria-current="page"~');

    $graph = otherStageGraph($html);
    expect($graph['FAQPage']['@id'])->toBe('https://ritme.test'.$path.'#faq')
        ->and($graph['FAQPage']['mainEntity'])->toHaveCount(3)
        ->and($graph)->toHaveKey('MobileApplication')
        ->and(end($graph['BreadcrumbList']['itemListElement'])['item'] ?? null)->toBe('https://ritme.test'.$path);
})->with([
    'postpartum' => ['/postpartum', 'پس از زایمان: بهبودی مادر، شیردهی و رشد کودک', 'حال خودت', 3, true, 'سؤال‌های رایج پس از زایمان'],
    'menopause' => ['/menopause', 'یائسگی: ثبت گرگرفتگی، روند علائم و یادآور چکاپ', 'یائسگی، فصل تازه؛', 3, true, 'سؤال‌های رایج درباره یائسگی'],
    'teen' => ['/teen', 'اولین پریود؛ تقویم ساده نوجوان با همراهی مادر', 'اولین پریود، بدون ترس', 2, false, 'سؤال‌های رایج نوجوان‌ها و والدین'],
]);

it('renders the stage-specific mock screens with their copy', function (): void {
    expect($this->get('/postpartum')->getContent())
        ->toContain('آوا · ۴ ماهه')->toContain('نمودار رشد · وزن')->toContain('سلام علی')
        ->toContain('href="/directory"');

    expect($this->get('/menopause')->getContent())
        ->toContain('۱۴ ماه بدون پریود')->toContain('امتیاز علائم')->toContain('گرگرفتگی الان');
});

it('keeps the teen page age-appropriate with the parent section', function (): void {
    $html = $this->get('/teen')->assertOk()->getContent();

    expect($html)->toContain('مادر در جریان است، نه ناظر')
        ->and($html)->toContain('مادر فقط چیزهایی را می‌بیند که دخترش اجازه بدهد')
        ->and($html)->toContain('با اجازه دخترت')
        // no fertile window, partner copy or services block (the layout nav still links the shop)
        ->and($html)->not->toContain('پنجره باروری')
        ->and($html)->not->toContain('لوتئال')
        ->and($html)->not->toContain('وقتی کمک بیشتری لازم داری')
        // design placeholder replaced, never rendered raw
        ->and($html)->not->toContain('طبق قوانین]');
});

it('builds the three stages with SEO copy inside the length limits', function (string $class): void {
    $page = app(StagePageBuilder::class)->build(app($class));

    expect(mb_strlen($page->seoTitle.' — ریتمی'))->toBeBetween(30, 60)
        ->and(mb_strlen($page->seoDescription))->toBeBetween(70, 160)
        ->and($page->faqTitle)->not->toBe($page->name)
        ->and($page->tools)->toHaveCount(3);
})->with([Postpartum::class, Menopause::class, Teen::class]);

it('gives the three stages unique titles and descriptions', function (): void {
    $pages = array_map(static fn (string $class) => app(StagePageBuilder::class)->build(app($class)), [Postpartum::class, Menopause::class, Teen::class]);

    expect(array_unique(array_map(static fn ($p): string => $p->seoTitle, $pages)))->toHaveCount(3)
        ->and(array_unique(array_map(static fn ($p): string => $p->seoDescription, $pages)))->toHaveCount(3);
});
