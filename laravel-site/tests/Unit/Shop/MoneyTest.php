<?php

declare(strict_types=1);

use App\Support\Money\Money;
use App\Support\Text\Toman;

it('stores integer rials and converts to toman', function (): void {
    $money = Money::fromToman(129_000);

    expect($money->rial)->toBe(1_290_000)
        ->and($money->rial())->toBe(1_290_000)
        ->and($money->toToman())->toBe(129_000)
        ->and(Money::fromRial(1_290_000)->equals($money))->toBeTrue()
        ->and(Money::zero()->isZero())->toBeTrue();
});

it('rounds stray rials to whole toman half away from zero', function (int $rial, int $toman): void {
    expect(Money::fromRial($rial)->toToman())->toBe($toman);
})->with([
    [14, 1], [15, 2], [19, 2], [-15, -2], [-14, -1], [0, 0],
]);

it('formats Persian prices', function (): void {
    expect(Money::fromToman(129_000)->format())->toBe('۱۲۹٬۰۰۰ تومان')
        ->and(Money::fromToman(485_000)->formatShort())->toBe('۴۸۵ هزار تومان')
        ->and(Money::fromToman(12_500)->formatShort())->toBe('۱۲٬۵۰۰ تومان')
        ->and(Money::fromToman(1_500_000)->formatAmount(true))->toBe('۱٬۵۰۰ هزار')
        ->and(Money::fromToman(98_000)->formatAmount())->toBe('۹۸٬۰۰۰')
        ->and((string) Money::fromToman(950))->toBe('۹۵۰ تومان')
        ->and(Money::zero()->format())->toBe('۰ تومان');
});

it('keeps the Toman helper consistent with Money', function (int $toman): void {
    expect(Toman::format($toman))->toBe(Money::fromToman($toman)->formatAmount(true))
        ->and(Toman::format($toman, false))->toBe(Money::fromToman($toman)->formatAmount())
        ->and(Toman::withUnit($toman))->toBe(Money::fromToman($toman)->formatShort());
})->with([0, 950, 12_500, 485_000, 1_500_000]);

it('does integer arithmetic only', function (): void {
    $price = Money::fromToman(485_000);

    expect($price->times(3)->toToman())->toBe(1_455_000)
        ->and($price->plus(Money::fromToman(15_000))->toToman())->toBe(500_000)
        ->and($price->minus(Money::fromToman(500_000))->isNegative())->toBeTrue()
        ->and(Money::sum([Money::fromToman(1), Money::fromToman(2), Money::fromToman(3)])->rial)->toBe(60)
        ->and(Money::min(Money::fromToman(5), Money::fromToman(3), Money::fromToman(9))->toToman())->toBe(3)
        ->and(Money::max(Money::fromToman(5), Money::fromToman(3), Money::fromToman(9))->toToman())->toBe(9)
        ->and($price->compareTo(Money::fromToman(1)))->toBe(1)
        ->and($price->greaterThan(Money::fromToman(1)))->toBeTrue()
        ->and($price->lessThan(Money::fromToman(1)))->toBeFalse()
        ->and(json_encode(['p' => $price]))->toBe('{"p":4850000}');
});

it('throws instead of overflowing into a float', function (): void {
    expect(fn () => Money::fromRial(PHP_INT_MAX)->plus(Money::fromRial(1)))->toThrow(OverflowException::class)
        ->and(fn () => Money::fromRial(PHP_INT_MAX)->times(2))->toThrow(OverflowException::class)
        ->and(fn () => Money::fromToman(PHP_INT_MAX))->toThrow(OverflowException::class);
});

it('computes a never-overstated discount percentage', function (): void {
    expect(Money::fromToman(390_000)->percentOff(Money::fromToman(460_000)))->toBe(15)
        ->and(Money::fromToman(420_000)->percentOff(Money::fromToman(490_000)))->toBe(14)
        ->and(Money::fromToman(500)->percentOff(Money::fromToman(500)))->toBeNull()
        ->and(Money::fromToman(600)->percentOff(Money::fromToman(500)))->toBeNull()
        ->and(Money::fromToman(999)->percentOff(Money::fromToman(1000)))->toBeNull();
});

it('parses stored and typed rial amounts', function (): void {
    expect(Money::parseRial('4850000')->rial)->toBe(4_850_000)
        ->and(Money::parseRial('۴٬۸۵۰٬۰۰۰')->rial)->toBe(4_850_000)
        ->and(Money::parseRial(12)->rial)->toBe(12)
        ->and(fn () => Money::parseRial('12.5'))->toThrow(InvalidArgumentException::class)
        ->and(fn () => Money::parseRial('abc'))->toThrow(InvalidArgumentException::class);
});
