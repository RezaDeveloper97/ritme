<?php

declare(strict_types=1);

use App\Support\Text\PersianDigits;

it('converts Latin and Arabic-Indic digits to Persian', function (string|int|float|null $input, string $expected): void {
    expect(PersianDigits::toPersian($input))->toBe($expected);
})->with([
    'string' => ['1403/01/01', '۱۴۰۳/۰۱/۰۱'],
    'int' => [2026, '۲۰۲۶'],
    'float' => [4.5, '۴.۵'],
    'arabic-indic' => ['٤٥٦', '۴۵۶'],
    'mixed text' => ['گام 3 از 10', 'گام ۳ از ۱۰'],
    'already persian' => ['۱۲۳', '۱۲۳'],
    'null' => [null, ''],
]);

it('converts Persian digits and separators back to Latin', function (): void {
    expect(PersianDigits::toLatin('۰۹۱۲ ٣٤٥'))->toBe('0912 345')
        ->and(PersianDigits::toLatin('۱٬۲۵۰٫۵'))->toBe('1,250.5')
        ->and(PersianDigits::toLatin(null))->toBe('');
});

it('formats numbers with Persian separators', function (int|float $value, int $decimals, string $expected): void {
    expect(PersianDigits::number($value, $decimals))->toBe($expected);
})->with([
    'small' => [7, 0, '۷'],
    'thousands' => [1250, 0, '۱٬۲۵۰'],
    'millions' => [12500000, 0, '۱۲٬۵۰۰٬۰۰۰'],
    'one decimal' => [4.8, 1, '۴٫۸'],
    'trailing zero dropped' => [4.0, 1, '۴'],
    'rounded' => [4.86, 1, '۴٫۹'],
    'grouped decimal' => [1250.25, 2, '۱٬۲۵۰٫۲۵'],
    'negative' => [-1500, 0, '-۱٬۵۰۰'],
    'negative zero' => [-0.01, 1, '۰'],
    'float without decimals' => [3.6, 0, '۴'],
]);

it('exposes the global fa_digits() helper', function (): void {
    expect(function_exists('fa_digits'))->toBeTrue()
        ->and(fa_digits(115))->toBe('۱۱۵')
        ->and(fa_digits('021-88'))->toBe('۰۲۱-۸۸');
});
