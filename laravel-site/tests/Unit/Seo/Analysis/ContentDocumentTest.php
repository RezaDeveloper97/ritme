<?php

declare(strict_types=1);

use App\Domain\Seo\Analysis\ContentDocument;

it('reads paragraphs, headings, images and links from rich-editor HTML', function (): void {
    $doc = ContentDocument::parse(
        '<p>پاراگراف اول درباره چرخه.</p><h2>بخش یک</h2><ul><li><p>مورد لیست</p></li></ul>'
        .'<h3>زیربخش</h3><p>متن با <a href="/cycle">پیوند</a> و <a href="https://who.int" target="_blank">منبع</a>.</p>'
        .'<img src="/media/a.webp" alt="نمودار چرخه"><img src="/media/b.webp" alt=" ">',
    );

    expect($doc->paragraphs)->toBe(['پاراگراف اول درباره چرخه.', 'مورد لیست', 'متن با پیوند و منبع.'])
        ->and($doc->headings)->toBe([['level' => 2, 'text' => 'بخش یک'], ['level' => 3, 'text' => 'زیربخش']])
        ->and($doc->imageAlts)->toBe(['نمودار چرخه', null])
        ->and(array_column($doc->links, 'href'))->toBe(['/cycle', 'https://who.int'])
        ->and($doc->links[1]['blank'])->toBeTrue()
        ->and($doc->wordCount)->toBeGreaterThan(10);
});

it('treats plain text lines as paragraphs', function (): void {
    $doc = ContentDocument::parse("خط اول توضیح.\n\nخط دوم توضیح.");

    expect($doc->paragraphs)->toBe(['خط اول توضیح.', 'خط دوم توضیح.'])
        ->and($doc->headings)->toBe([])
        ->and($doc->wordCount)->toBe(6);
});

it('is empty for empty content', function (): void {
    expect(ContentDocument::parse('  ')->wordCount)->toBe(0);
});
