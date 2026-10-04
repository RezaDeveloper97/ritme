<?php

declare(strict_types=1);

use App\Domain\Seo\Analysis\AnalysisInput;
use App\Domain\Seo\Analysis\ContentType;
use App\Domain\Seo\Analysis\SeoAnalysis;
use App\Domain\Seo\Analysis\SeoAnalyzer;
use App\Domain\Seo\Analysis\Severity;

/**
 * A healthy Persian post body around «درد پریود»: ~650 words, h2/h3, images with alt, 2 internal + 1 external link.
 */
function analyserBody(int $paragraphs = 12): string
{
    $sentence = 'بسیاری از زنان در روزهای اول چرخه گرفتگی عضلات شکم را تجربه می‌کنند و این حس معمولاً با استراحت کمتر می‌شود.';
    $html = '<p>درد پریود برای بسیاری آشناست و دانستن علت آن کمک می‌کند. '.$sentence.'</p>';
    $html .= '<h2>چرا درد پریود رخ می‌دهد</h2>';
    for ($i = 0; $i < $paragraphs; $i++) {
        $html .= '<p>'.str_repeat($sentence.' ', 2).'گرمای ملایم، خواب کافی و تحرک سبک به بدن کمک می‌کند.</p>';
        if ($i === 4) {
            $html .= '<h3>نشانه‌ها</h3><img src="/media/1.webp" alt="نمودار درد پریود در چرخه">';
        }
        if ($i === 8) {
            $html .= '<h2>چه زمانی به پزشک مراجعه کنیم</h2><p>اگر درد شدید است، با پزشک مشورت کنید. <a href="/cycle">چرخه</a> و <a href="https://ritmeapp.ir/blog/pms">سندرم پیش از قاعدگی</a> را ببینید. <a href="https://www.who.int/" rel="noopener" target="_blank">منبع</a></p>';
        }
    }

    return $html;
}

/**
 * @param  array<string, mixed>  $overrides
 */
function analyse(array $overrides = []): SeoAnalysis
{
    $input = array_merge([
        'type' => ContentType::Post,
        'title' => 'درد پریود چه زمانی طبیعی است؟ راهنمای ساده — ریتمی',
        'heading' => 'درد پریود چه زمانی طبیعی است؟',
        'description' => 'درد پریود چه زمانی طبیعی است و چه زمانی باید با پزشک مشورت کرد؟ نشانه‌ها، علت‌ها و راه‌های ساده برای آرام‌تر گذراندن این روزها.',
        'slug' => 'درد-پریود',
        'focusKeyword' => 'درد پریود',
        'contentHtml' => analyserBody(),
        'ownHosts' => ['ritmeapp.ir'],
    ], $overrides);

    return (new SeoAnalyzer)->analyze(new AnalysisInput(...$input));
}

function severityOf(SeoAnalysis $analysis, string $key): ?Severity
{
    return $analysis->check($key)?->severity;
}

it('passes a well-written Persian post with a high score and no errors', function (): void {
    $analysis = analyse();

    $failing = array_map(static fn ($c): string => $c->key.': '.$c->message, array_filter($analysis->checks, static fn ($c): bool => in_array($c->severity, [Severity::Error, Severity::Warning], true)));
    expect($failing)->toBe([])
        ->and($analysis->score)->toBeGreaterThanOrEqual(SeoAnalysis::GOOD_SCORE)
        ->and($analysis->needsWork())->toBeFalse()
        ->and($analysis->color())->toBe('success');
});

it('is deterministic', function (): void {
    expect(analyse())->toEqual(analyse());
});

it('lists errors first, then warnings, info and passes', function (): void {
    $ranks = array_map(static fn ($c): int => $c->severity->rank(), analyse(['title' => 'کوتاه', 'contentHtml' => '<h1>x</h1><p>حتماً</p>'])->checks);
    $sorted = $ranks;
    sort($sorted);

    expect($ranks)->toBe($sorted);
});

describe('title and description', function (): void {
    it('errors on an empty or cut title and warns on a short one', function (): void {
        expect(severityOf(analyse(['title' => '']), 'title_length'))->toBe(Severity::Error)
            ->and(severityOf(analyse(['title' => str_repeat('عنوان بسیار طولانی ', 8)]), 'title_length'))->toBe(Severity::Error)
            ->and(analyse(['title' => str_repeat('عنوان بسیار طولانی ', 8)])->check('title_length')?->message)->toContain('پیکسل')
            ->and(severityOf(analyse(['title' => 'درد پریود — ریتمی']), 'title_length'))->toBe(Severity::Warning);
    });

    it('warns on an empty, short or long description', function (): void {
        expect(severityOf(analyse(['description' => '']), 'description_length'))->toBe(Severity::Warning)
            ->and(severityOf(analyse(['description' => 'درد پریود کوتاه.']), 'description_length'))->toBe(Severity::Warning)
            ->and(severityOf(analyse(['description' => str_repeat('درد پریود و راه‌های آرام کردن آن. ', 8)]), 'description_length'))->toBe(Severity::Warning);
    });

    it('reports duplicates only when the caller checked them', function (): void {
        expect(analyse()->check('duplicate_title'))->toBeNull()
            ->and(severityOf(analyse(['duplicateTitle' => true]), 'duplicate_title'))->toBe(Severity::Error)
            ->and(severityOf(analyse(['duplicateTitle' => false]), 'duplicate_title'))->toBe(Severity::Pass)
            ->and(severityOf(analyse(['duplicateDescription' => true]), 'duplicate_description'))->toBe(Severity::Error);
    });
});

describe('focus keyword', function (): void {
    it('asks for a keyword and skips keyword checks without one', function (): void {
        $analysis = analyse(['focusKeyword' => '  ']);

        expect(severityOf($analysis, 'focus_keyword'))->toBe(Severity::Warning)
            ->and($analysis->check('keyword_title'))->toBeNull()
            ->and($analysis->check('keyword_density'))->toBeNull();
    });

    it('finds the keyword in title, description, h1, slug and first paragraph with Persian normalisation', function (): void {
        $analysis = analyse(['focusKeyword' => 'دردِ پريود', 'slug' => rawurlencode('درد-پریود-طبیعی')]);

        foreach (['keyword_title', 'keyword_description', 'keyword_heading', 'keyword_slug', 'keyword_first_paragraph', 'keyword_subheadings', 'keyword_image_alt'] as $key) {
            expect(severityOf($analysis, $key))->toBe(Severity::Pass, $key);
        }
    });

    it('flags each place the keyword is missing', function (): void {
        $analysis = analyse(['focusKeyword' => 'کیست تخمدان']);

        expect(severityOf($analysis, 'keyword_title'))->toBe(Severity::Error)
            ->and(severityOf($analysis, 'keyword_description'))->toBe(Severity::Warning)
            ->and(severityOf($analysis, 'keyword_heading'))->toBe(Severity::Warning)
            ->and(severityOf($analysis, 'keyword_slug'))->toBe(Severity::Warning)
            ->and(severityOf($analysis, 'keyword_first_paragraph'))->toBe(Severity::Warning)
            ->and(severityOf($analysis, 'keyword_subheadings'))->toBe(Severity::Warning)
            ->and(severityOf($analysis, 'keyword_image_alt'))->toBe(Severity::Warning)
            ->and(severityOf($analysis, 'keyword_density'))->toBe(Severity::Warning);
    });

    it('treats a Latin slug with a Persian keyword as advice, not a failure', function (): void {
        expect(severityOf(analyse(['slug' => 'period-pain']), 'keyword_slug'))->toBe(Severity::Info);
    });

    it('keeps keyword density between 0.5% and 2.5%', function (): void {
        $filler = '<p>'.str_repeat('متن عمومی درباره سلامت و زندگی روزمره ', 30).'</p>';
        $stuffed = '<p>'.str_repeat('درد پریود ', 40).'</p>'.$filler;
        $sparse = '<p>درد پریود</p>'.str_repeat($filler, 4);

        expect(severityOf(analyse(['contentHtml' => $stuffed]), 'keyword_density'))->toBe(Severity::Warning)
            ->and(analyse(['contentHtml' => $stuffed])->check('keyword_density')?->message)->toContain('تکرار زیاد')
            ->and(severityOf(analyse(['contentHtml' => $sparse]), 'keyword_density'))->toBe(Severity::Warning)
            ->and(analyse(['contentHtml' => $sparse])->check('keyword_density')?->message)->toContain('تراکم کم')
            ->and(severityOf(analyse(['contentHtml' => '<p>درد پریود کوتاه</p>']), 'keyword_density'))->toBe(Severity::Info);
    });
});

describe('content and structure', function (): void {
    it('sizes the body per content type and cornerstone flag', function (): void {
        $body = analyserBody(4); // ~250 words

        expect(severityOf(analyse(['contentHtml' => $body]), 'content_length'))->toBe(Severity::Error)
            ->and(severityOf(analyse(['contentHtml' => $body, 'type' => ContentType::Product]), 'content_length'))->toBe(Severity::Pass)
            ->and(severityOf(analyse(['contentHtml' => analyserBody(), 'cornerstone' => true]), 'content_length'))->toBe(Severity::Warning)
            ->and(severityOf(analyse(['cornerstone' => true]), 'cornerstone'))->toBe(Severity::Info)
            ->and(severityOf(analyse(['contentHtml' => '']), 'content_length'))->toBe(Severity::Error);
    });

    it('skips body checks when the page has no body (static pages)', function (): void {
        $analysis = analyse(['type' => ContentType::Page, 'contentHtml' => null, 'heading' => '']);

        expect($analysis->check('content_length'))->toBeNull()
            ->and($analysis->check('image_alt'))->toBeNull()
            ->and($analysis->check('keyword_first_paragraph'))->toBeNull()
            ->and($analysis->check('keyword_heading'))->toBeNull()
            ->and(severityOf($analysis, 'keyword_title'))->toBe(Severity::Pass);
    });

    it('allows only the page title as h1 and no skipped heading levels', function (): void {
        expect(severityOf(analyse(['contentHtml' => '<h1>تیتر دوم</h1>'.analyserBody()]), 'heading_single_h1'))->toBe(Severity::Error)
            ->and(severityOf(analyse(), 'heading_single_h1'))->toBe(Severity::Pass)
            ->and(severityOf(analyse(['contentHtml' => '<h3>پرش</h3>'.analyserBody()]), 'heading_hierarchy'))->toBe(Severity::Warning)
            ->and(analyse(['contentHtml' => '<h2>الف</h2><h4>ب</h4>'])->check('heading_hierarchy')?->message)->toContain('h2→h4')
            ->and(severityOf(analyse(['contentHtml' => strip_tags(analyserBody(), '<p>')]), 'heading_hierarchy'))->toBe(Severity::Warning);
    });

    it('errors on images without alt', function (): void {
        expect(severityOf(analyse(['contentHtml' => analyserBody().'<img src="/media/2.webp">']), 'image_alt'))->toBe(Severity::Error)
            ->and(analyse(['contentHtml' => analyserBody().'<img src="/media/2.webp" alt="">'])->check('image_alt')?->message)->toContain('۱ تصویر از ۲')
            ->and(severityOf(analyse(['contentHtml' => '<p>بی‌تصویر</p>']), 'image_alt'))->toBe(Severity::Info);
    });

    it('counts internal links (relative or own host) against the type minimum', function (): void {
        $noLinks = (string) preg_replace('/<a [^>]*>(.*?)<\/a>/u', '$1', analyserBody());

        expect(severityOf(analyse(), 'internal_links'))->toBe(Severity::Pass)
            ->and(analyse()->check('internal_links')?->message)->toContain('۲ پیوند داخلی')
            ->and(severityOf(analyse(['contentHtml' => $noLinks]), 'internal_links'))->toBe(Severity::Warning)
            ->and(severityOf(analyse(['contentHtml' => '<p><a href="/shop">x</a></p>', 'type' => ContentType::Product]), 'internal_links'))->toBe(Severity::Pass)
            ->and(severityOf(analyse(['contentHtml' => '<p><a href="mailto:a@b.c">x</a> <a href="#top">y</a></p>']), 'internal_links'))->toBe(Severity::Warning);
    });

    it('checks outbound links open safely and suggests sources when there are none', function (): void {
        expect(severityOf(analyse(), 'external_links'))->toBe(Severity::Pass)
            ->and(severityOf(analyse(['contentHtml' => '<p><a href="https://example.org" target="_blank">x</a></p>']), 'external_links'))->toBe(Severity::Warning)
            ->and(analyse(['contentHtml' => '<p><a href="https://example.org" rel="nofollow">x</a></p>'])->check('external_links')?->message)->toContain('nofollow')
            ->and(severityOf(analyse(['contentHtml' => '<p>بدون پیوند</p>']), 'external_links'))->toBe(Severity::Info);
    });

    it('warns on long slugs and stopwords', function (): void {
        expect(severityOf(analyse(), 'slug'))->toBe(Severity::Pass)
            ->and(severityOf(analyse(['slug' => 'درد-و-پریود']), 'slug'))->toBe(Severity::Warning)
            ->and(severityOf(analyse(['slug' => 'how-to-ease-period-pain']), 'slug'))->toBe(Severity::Warning)
            ->and(severityOf(analyse(['slug' => str_repeat('a', 80)]), 'slug'))->toBe(Severity::Warning)
            ->and(severityOf(analyse(['slug' => '']), 'slug'))->toBe(Severity::Info);
    });
});

describe('readability', function (): void {
    it('warns on long sentences', function (): void {
        $long = '<p>'.str_repeat('این جمله بسیار طولانی است و بدون نقطه ادامه پیدا می‌کند و خواننده را خسته می‌کند و ', 12).'.</p>';

        expect(severityOf(analyse(['contentHtml' => $long]), 'readability_sentences'))->toBe(Severity::Warning)
            ->and(severityOf(analyse(), 'readability_sentences'))->toBe(Severity::Pass);
    });

    it('warns on paragraphs over 150 words', function (): void {
        $wall = '<p>'.str_repeat('جمله کوتاه است. ', 60).'</p>';

        expect(severityOf(analyse(['contentHtml' => $wall]), 'readability_paragraphs'))->toBe(Severity::Warning)
            ->and(severityOf(analyse(), 'readability_paragraphs'))->toBe(Severity::Pass)
            ->and(severityOf(analyse(['contentHtml' => '<p>کوتاه.</p>']), 'readability_paragraphs'))->toBe(Severity::Info);
    });
});

describe('red lines', function (): void {
    it('errors on brief red-line words anywhere and caps the score', function (string $field, string $value): void {
        $analysis = analyse([$field => $value]);

        expect(severityOf($analysis, 'red_lines'))->toBe(Severity::Error)
            ->and($analysis->score)->toBeLessThanOrEqual(SeoAnalyzer::RED_LINE_SCORE_CAP)
            ->and($analysis->needsWork())->toBeTrue();
    })->with([
        'title' => ['title', 'درد پریود را حتماً جدی بگیرید — ریتمی'],
        'description' => ['description', 'دقیق‌ترین راهنمای درد پریود'],
        'body' => ['contentHtml', analyserBody().'<p>نتیجه قطعاً تضمینی است.</p>'],
    ]);

    it('names every red-line word found', function (): void {
        expect(analyse(['contentHtml' => analyserBody().'<p>نتیجه قطعاً تضمینی است.</p>'])->check('red_lines')?->message)
            ->toContain('قطعاً')->toContain('تضمینی');
    });

    it('errors on diagnosis claims', function (): void {
        $analysis = analyse(['contentHtml' => analyserBody().'<p>ابزار ریتمی کیست تخمدان را تشخیص می‌دهد.</p>']);

        expect(severityOf($analysis, 'diagnosis_claims'))->toBe(Severity::Error)
            ->and($analysis->score)->toBeLessThanOrEqual(SeoAnalyzer::RED_LINE_SCORE_CAP)
            ->and(severityOf(analyse(), 'diagnosis_claims'))->toBe(Severity::Pass);
    });
});

it('runs fast enough for every keystroke', function (): void {
    $body = analyserBody(60); // ~3000 words
    $start = hrtime(true);
    for ($i = 0; $i < 10; $i++) {
        analyse(['contentHtml' => $body]);
    }

    expect((hrtime(true) - $start) / 1e6 / 10)->toBeLessThan(150.0);
});
