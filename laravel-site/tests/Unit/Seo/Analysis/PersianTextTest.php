<?php

declare(strict_types=1);

use App\Domain\Seo\Analysis\PersianText;

it('matches a keyword across Arabic letters, ZWNJ, spacing and diacritics', function (string $keyword, string $text): void {
    expect(PersianText::contains($keyword, $text))->toBeTrue();
})->with([
    'arabic yeh and kaf' => ['کیست تخمدان', 'درباره كيست تخمدان بخوانید'],
    'zwnj vs joined' => ['می‌شود', 'این درد کمتر میشود'],
    'zwnj vs space' => ['می‌شود', 'این درد کمتر می شود'],
    'harakat' => ['قاعدگی', 'قاعِدِگی منظم'],
    'tatweel' => ['پریود', 'پریــــود دردناک'],
    'suffix allowed' => ['پریود', 'پریودهای نامنظم'],
    'latin case' => ['PCOS', 'درباره pcos'],
    'persian digits' => ['هفته 12', 'بارداری هفته ۱۲'],
]);

it('requires a word start, so a keyword inside another word does not count', function (): void {
    expect(PersianText::contains('درد', 'سردرد شدید'))->toBeFalse()
        ->and(PersianText::contains('درد', 'درد شدید'))->toBeTrue()
        ->and(PersianText::contains('قطعا', 'قطعات یدکی', wholeWord: true))->toBeFalse();
});

it('counts occurrences and words Persian-style', function (): void {
    expect(PersianText::count('درد پریود', 'درد پریود را بشناسید. دردِ پريود طبیعی است؟ درد  پریود'))->toBe(3)
        ->and(PersianText::wordCount('این درد کمتر می‌شود.'))->toBe(4)
        ->and(PersianText::count('', 'متن'))->toBe(0);
});

it('splits sentences on Persian and Latin punctuation and line breaks, not inside numbers', function (): void {
    expect(PersianText::sentences("جمله اول است. جمله دوم؟ سوم! وزن ۲.۵ کیلو است\nخط بعد"))
        ->toBe(['جمله اول است.', 'جمله دوم؟', 'سوم!', 'وزن ۲.۵ کیلو است', 'خط بعد']);
});
