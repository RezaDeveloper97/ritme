<?php

declare(strict_types=1);

use App\Support\Html\ExternalLinks;
use App\Support\Html\HeadingAnchors;
use App\Support\Html\RichHtmlSanitizer;

function blogSanitizer(): RichHtmlSanitizer
{
    return new RichHtmlSanitizer(['ritme.ir'], ['/media/']);
}

it('removes scripts, event handlers, styles, iframes and forms', function (): void {
    $html = blogSanitizer()->sanitize('<p onclick="x()" style="color:red">سلام<script>alert(1)</script></p><iframe src="https://x.test"></iframe><form><input></form><style>p{}</style>');

    expect($html)->toBe('<p>سلام</p>');
});

it('keeps the allowed structure: headings, lists, tables, figure, blockquote, links', function (): void {
    $input = '<h2>عنوان</h2><h3>زیر</h3><ul><li>یک</li></ul><ol start="2"><li>دو</li></ol>'
        .'<table><thead><tr><th scope="col">الف</th></tr></thead><tbody><tr><td colspan="2">ب</td></tr></tbody></table>'
        .'<figure><img data-media-id="5" alt="تصویر"><figcaption>زیرنویس</figcaption></figure>'
        .'<blockquote><p>نقل</p></blockquote><p><a href="/blog/x" title="t">داخلی</a> <strong>پررنگ</strong></p>';

    expect(blogSanitizer()->sanitize($input))->toBe($input);
});

it('demotes h1 to h2 and unwraps div/span wrappers keeping their text', function (): void {
    expect(blogSanitizer()->sanitize('<h1>تیتر</h1><div><span>متن</span></div>'))->toBe('<h2>تیتر</h2>متن');
});

it('keeps images only from own media', function (string $img, bool $kept): void {
    $html = blogSanitizer()->sanitize('<p>x</p>'.$img);

    expect(str_contains($html, '<img'))->toBe($kept);
})->with([
    'media id' => ['<img data-media-id="12" alt="a">', true],
    'relative media path' => ['<img src="/media/2026/10/a.jpg" alt="a">', true],
    'absolute own host' => ['<img src="https://ritme.ir/media/a.jpg" alt="a">', true],
    'external host' => ['<img src="https://evil.test/media/a.jpg" alt="a">', false],
    'own host outside media' => ['<img src="https://ritme.ir/uploads/a.jpg" alt="a">', false],
    'relative non-media' => ['<img src="/images/a.jpg" alt="a">', false],
    'path traversal' => ['<img src="/media/../secret.png" alt="a">', false],
    'protocol relative' => ['<img src="//evil.test/media/a.jpg" alt="a">', false],
    'data uri' => ['<img src="data:image/png;base64,AAAA" alt="a">', false],
    'bad media id' => ['<img data-media-id="1 OR 1" alt="a">', false],
]);

it('drops javascript links but keeps their text, and filters rel tokens', function (): void {
    $html = blogSanitizer()->sanitize('<p><a href="javascript:alert(1)">بد</a> <a href="https://x.test" rel="nofollow evil" target="_blank">خوب</a></p>');

    expect($html)->toBe('<p>بد <a href="https://x.test" rel="nofollow">خوب</a></p>');
});

it('adds rel=noopener to external links only', function (): void {
    $html = ExternalLinks::apply('<a href="https://x.test/a" rel="nofollow">x</a><a href="https://ritme.ir/a">own</a><a href="/blog">rel</a><a href="mailto:a@b.c">m</a>', ['ritme.ir']);

    expect($html)->toBe('<a href="https://x.test/a" rel="nofollow noopener">x</a><a href="https://ritme.ir/a">own</a><a href="/blog">rel</a><a href="mailto:a@b.c">m</a>');
});

it('gives h2/h3 unique Persian ids and reads the outline back', function (): void {
    $html = HeadingAnchors::apply('<h2>درد پریود چرا ایجاد می‌شود</h2><p>x</p><h3 id="Custom Id">بخش</h3><h2>درد پریود چرا ایجاد می‌شود</h2><h4>نه</h4>');

    expect($html)->toBe('<h2 id="درد-پریود-چرا-ایجاد-می-شود">درد پریود چرا ایجاد می‌شود</h2><p>x</p><h3 id="custom-id">بخش</h3><h2 id="درد-پریود-چرا-ایجاد-می-شود-2">درد پریود چرا ایجاد می‌شود</h2><h4>نه</h4>')
        ->and(HeadingAnchors::outline($html))->toBe([
            ['level' => 2, 'id' => 'درد-پریود-چرا-ایجاد-می-شود', 'text' => 'درد پریود چرا ایجاد می‌شود'],
            ['level' => 3, 'id' => 'custom-id', 'text' => 'بخش'],
            ['level' => 2, 'id' => 'درد-پریود-چرا-ایجاد-می-شود-2', 'text' => 'درد پریود چرا ایجاد می‌شود'],
        ]);
});

it('prefixes ids that would start with a digit', function (): void {
    expect(HeadingAnchors::apply('<h2>۱۰ نکته</h2><h2>!!!</h2>'))->toBe('<h2 id="section-10-نکته">۱۰ نکته</h2><h2 id="section">!!!</h2>');
});
