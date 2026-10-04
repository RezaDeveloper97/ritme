<?php

declare(strict_types=1);

use App\Domain\Seo\Support\CanonicalUrl;

$base = 'https://ritme.test';

it('normalises to https on the configured host without trailing slash, fragment or tracking', function (string $url, string $expected) use ($base): void {
    expect(CanonicalUrl::normalize($url, $base))->toBe($expected);
})->with([
    'root' => ['http://ritme.test/', 'https://ritme.test'],
    'relative' => ['/cycle/', 'https://ritme.test/cycle'],
    'host case + scheme' => ['http://RITME.test/About', 'https://ritme.test/About'],
    'foreign host forced to own host' => ['http://127.0.0.1:8000/faq', 'https://ritme.test/faq'],
    'double slashes' => ['https://ritme.test//blog///period-pain/', 'https://ritme.test/blog/period-pain'],
    'fragment' => ['https://ritme.test/tools#due-date', 'https://ritme.test/tools'],
    'tracking dropped' => ['https://ritme.test/blog?utm_source=ig&utm_medium=x&gclid=1&fbclid=2', 'https://ritme.test/blog'],
    'filters dropped' => ['https://ritme.test/shop?color=red&sort=price', 'https://ritme.test/shop'],
    'page 1 dropped' => ['https://ritme.test/blog?page=1', 'https://ritme.test/blog'],
    'page 2 kept' => ['https://ritme.test/blog?page=2&utm_source=x', 'https://ritme.test/blog?page=2'],
    'page junk dropped' => ['https://ritme.test/blog?page=abc', 'https://ritme.test/blog'],
    'persian path kept' => ['https://ritme.test/blog/%D8%AF%D8%B1%D8%AF/', 'https://ritme.test/blog/%D8%AF%D8%B1%D8%AF'],
]);

it('keeps the port of the base url and allow-listed parameters', function (): void {
    expect(CanonicalUrl::normalize('/blog?tag=x&page=3&z=1', 'http://127.0.0.1:8000', ['tag']))
        ->toBe('https://127.0.0.1:8000/blog?page=3&tag=x');
});
