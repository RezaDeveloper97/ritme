<?php

declare(strict_types=1);

use App\Domain\Seo\Audit\AuditIssue;
use App\Domain\Seo\Audit\SeoAuditor;
use App\Domain\Seo\Audit\Severity;

function seoAuditHtml(array $overrides = []): string
{
    $o = $overrides + [
        'title' => 'راهنمای پیگیری چرخه قاعدگی — ریتمی',
        'description' => 'ریتمی کمک می‌کند چرخه‌ات را بشناسی؛ با زبان احتمالی، بدون ترساندن و با داده‌ای که پیش خودت می‌ماند.',
        'canonical' => 'https://ritme.test/cycle',
        'robots' => 'index,follow',
        'og' => '<meta property="og:locale" content="fa_IR"><meta property="og:site_name" content="ریتمی"><meta property="og:type" content="website"><meta property="og:url" content="https://ritme.test/cycle"><meta property="og:title" content="t"><meta property="og:description" content="d">',
        'body' => '<h1>چرخه</h1><img src="/a.webp" alt="" width="10" height="10"><a href="/faq">سؤال‌ها</a>',
    ];

    return '<!doctype html><html lang="fa" dir="rtl"><head><meta charset="utf-8">'
        ."<title>{$o['title']}</title><meta name=\"description\" content=\"{$o['description']}\">"
        .($o['canonical'] === null ? '' : "<link rel=\"canonical\" href=\"{$o['canonical']}\">")
        ."<meta name=\"robots\" content=\"{$o['robots']}\">{$o['og']}</head><body>{$o['body']}"
        .'<svg><title>آیکون</title></svg></body></html>';
}

/**
 * @return list<string>
 */
function seoAuditCodes(array $issues, ?Severity $severity = Severity::Error): array
{
    return array_values(array_map(
        static fn (AuditIssue $i): string => $i->code,
        array_filter($issues, static fn (AuditIssue $i): bool => $severity === null || $i->severity === $severity),
    ));
}

it('passes a complete page with only the og:image warning', function (): void {
    $page = (new SeoAuditor)->auditPage('/cycle', seoAuditHtml());

    expect(seoAuditCodes($page->issues))->toBe([])
        ->and(seoAuditCodes($page->issues, Severity::Warning))->toBe(['og.image'])
        ->and($page->indexable)->toBeTrue()
        ->and($page->title)->toBe('راهنمای پیگیری چرخه قاعدگی — ریتمی');
});

it('flags every v1 rule', function (): void {
    $page = (new SeoAuditor)->auditPage('/bad', seoAuditHtml([
        'title' => 'کوتاه',
        'description' => 'خیلی کوتاه',
        'canonical' => '/bad',
        'og' => '<meta property="og:image" content="https://ritme.test/og.jpg">',
        'body' => '<h1>یک</h1><h1>دو</h1><img src="/a.webp" alt="x"><img src="/b.webp" width="1" height="1"><a href="#">x</a><a href=" # ">y</a>',
    ]));

    expect(seoAuditCodes($page->issues))->toBe([
        'title.length', 'description.length', 'h1.count', 'canonical.relative',
        'og.missing', 'og.missing', 'og.missing', 'og.missing', 'og.missing', 'og.missing',
        'og.missing', 'og.missing', 'og.missing',
        'img.attributes', 'img.attributes', 'link.hash',
    ]);
});

it('reports missing title, description, h1 and canonical', function (): void {
    $html = '<!doctype html><html><head><meta charset="utf-8"></head><body><p>x</p></body></html>';

    expect(seoAuditCodes((new SeoAuditor)->auditPage('/empty', $html)->issues))
        ->toContain('title.missing', 'description.missing', 'h1.count', 'canonical.missing');
});

it('flags duplicate titles and descriptions among indexable pages only', function (): void {
    $auditor = new SeoAuditor;
    $pages = $auditor->crossCheck([
        $auditor->auditPage('/a', seoAuditHtml()),
        $auditor->auditPage('/b', seoAuditHtml()),
        $auditor->auditPage('/c', seoAuditHtml(['robots' => 'noindex,follow'])),
    ]);

    expect(seoAuditCodes($pages[0]->issues))->toBe(['title.duplicate', 'description.duplicate'])
        ->and($pages[0]->issues[1]->message)->toContain('/b')->not->toContain('/c')
        ->and(seoAuditCodes($pages[2]->issues))->toBe([])
        ->and($pages[2]->indexable)->toBeFalse();
});
