<?php

declare(strict_types=1);

use App\Support\Text\Toman;

it('formats toman amounts like the design', function (int $amount, string $expected): void {
    expect(Toman::format($amount))->toBe($expected);
})->with([
    'whole thousands' => [485000, '۴۸۵ هزار'],
    'exactly one thousand' => [1000, '۱ هزار'],
    'large whole thousands' => [1500000, '۱٬۵۰۰ هزار'],
    'not whole thousands' => [12500, '۱۲٬۵۰۰'],
    'below a thousand' => [950, '۹۵۰'],
    'zero' => [0, '۰'],
]);

it('can skip the thousands shorthand', function (): void {
    expect(Toman::format(485000, false))->toBe('۴۸۵٬۰۰۰');
});

it('appends the unit', function (): void {
    expect(Toman::withUnit(320000))->toBe('۳۲۰ هزار تومان')
        ->and(Toman::withUnit(12500))->toBe('۱۲٬۵۰۰ تومان');
});
