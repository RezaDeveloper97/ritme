<?php

declare(strict_types=1);

use App\Domain\Seo\Support\DescriptionText;

it('returns short text untouched apart from tags, entities and whitespace', function (): void {
    expect(DescriptionText::fromExcerpt("<p>ریتمی&nbsp;همراه   سلامت</p>\n<b>زنان</b>"))->toBe('ریتمی همراه سلامت زنان');
});

it('trims long Persian text at a word boundary to at most the limit', function (): void {
    $text = str_repeat('چرخه‌ی قاعدگی هر ماه کمی فرق می‌کند، ', 12);

    $description = DescriptionText::fromExcerpt($text);

    expect(mb_strlen($description))->toBeLessThanOrEqual(155)
        ->and(mb_strlen($description))->toBeGreaterThan(120)
        ->and($description)->toEndWith('…')
        ->and($description)->not->toContain('،…')
        ->and(mb_substr($description, -2, 1))->not->toBe("\u{200C}");

    // Never cut inside a word: the text before the ellipsis is a prefix that ends where the source has a space.
    $body = mb_substr($description, 0, -1);
    expect(str_starts_with($text, $body))->toBeTrue()
        ->and(in_array(mb_substr($text, mb_strlen($body), 1), [' ', '،'], true))->toBeTrue();
});

it('never splits a ZWNJ-joined word', function (): void {
    $text = str_repeat('الف ', 37).'می‌خواهم'.str_repeat(' ب', 20);

    expect(DescriptionText::fromExcerpt($text, 155))->not->toContain('می…')->not->toContain("می\u{200C}…");
});
