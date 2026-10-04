<?php

declare(strict_types=1);

use App\Support\Html\HtmlText;
use App\Support\Html\TextSlug;

it('makes Persian-friendly slugs', function (string $input, string $slug): void {
    expect(TextSlug::make($input))->toBe($slug);
})->with([
    'persian title' => ['درد پریود؛ کی عادی است و کی ارزش پیگیری دارد؟', 'درد-پریود-کی-عادی-است-و-کی-ارزش-پیگیری-دارد'],
    'zwnj and arabic letters' => ['خواندنی‌های كوتاه', 'خواندنی-های-کوتاه'],
    'persian digits' => ['هفته ۱۲ بارداری', 'هفته-12-بارداری'],
    'latin' => ['Hello  World!', 'hello-world'],
    'diacritics' => ['سَلام', 'سلام'],
    'empty' => ['؟!', ''],
]);

it('trims long slugs on a word boundary', function (): void {
    $slug = TextSlug::make(str_repeat('کلمه ', 40), 30);

    expect(mb_strlen($slug))->toBeLessThanOrEqual(30)
        ->and($slug)->not->toEndWith('-')
        ->and($slug)->toStartWith('کلمه-کلمه');
});

it('counts Persian words (ZWNJ joins one word, punctuation is no word)', function (): void {
    expect(HtmlText::wordCount('<p>این مقاله‌ها می‌شود</p><p>دو.</p> - ؟'))->toBe(4)
        ->and(HtmlText::wordCount('<h2>یک</h2><p>دو&nbsp;سه</p>'))->toBe(3)
        ->and(HtmlText::wordCount(''))->toBe(0);
});

it('rounds reading time up with a floor of one minute', function (): void {
    expect(HtmlText::readingMinutes(0))->toBe(1)
        ->and(HtmlText::readingMinutes(200))->toBe(1)
        ->and(HtmlText::readingMinutes(201))->toBe(2)
        ->and(HtmlText::readingMinutes(1150))->toBe(6);
});
