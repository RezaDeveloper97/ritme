<?php

declare(strict_types=1);

use App\Domain\Seo\Analysis\RedLines;

it('finds the brief red-line words in any spelling', function (string $text, string $word): void {
    expect(RedLines::words($text))->toContain($word);
})->with([
    ['این روش حتماً جواب می‌دهد', 'حتماً'],
    ['حتما امتحان کنید', 'حتماً'],
    ['قطعا بهتر می‌شوید', 'قطعاً'],
    ['دقیق‌ترین محاسبه‌گر', 'دقیق‌ترین'],
    ['دقیق ترین روش', 'دقیق‌ترین'],
    ['دقيقترين', 'دقیق‌ترین'],
    ['نتیجه تضمینی', 'تضمینی'],
    ['صد در صد مطمئن', 'صددرصد'],
]);

it('does not flag ordinary words that share letters with a red-line word', function (): void {
    expect(RedLines::words('قطعات دستگاه و محاسبه دقیق دوره با حتمیت کم'))->toBe([]);
});

it('flags diagnosis and cure claims but not advice to see a doctor', function (): void {
    expect(RedLines::claims('ریتمی بیماری شما را تشخیص می‌دهد'))->not->toBe([])
        ->and(RedLines::claims('این تست کیست تخمدان را تشخیص میدهد'))->toContain('ادعای تشخیص توسط اپ یا ابزار')
        ->and(RedLines::claims('ما تشخیص می‌دهیم که چه مشکلی دارید'))->toContain('ادعای تشخیص')
        ->and(RedLines::claims('درمان قطعی درد پریود'))->toContain('وعده درمان قطعی')
        ->and(RedLines::claims('شما احتمالاً مبتلا هستید'))->toContain('برچسب بیماری به خواننده')
        ->and(RedLines::claims('بدون نیاز به پزشک درمان کنید'))->toContain('جایگزین کردن پزشک')
        ->and(RedLines::claims('برای تشخیص دقیق به پزشک مراجعه کنید؛ پزشک تشخیص می‌دهد که درمان لازم است یا نه.'))->toBe([]);
});
