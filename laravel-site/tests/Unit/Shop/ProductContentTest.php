<?php

declare(strict_types=1);

use App\Domain\Shop\Catalog\Support\ProductContent;

it('normalises specs from lists or maps and drops empty rows', function (): void {
    expect(ProductContent::specs([
        ['label' => ' جنس ', 'value' => '<b>پنبه ۱۰۰٪</b>'],
        ['label' => 'خالی', 'value' => ''],
        ['value' => 'بدون برچسب'],
    ]))->toBe([['label' => 'جنس', 'value' => 'پنبه ۱۰۰٪']])
        ->and(ProductContent::specs(['تعداد' => '۳ عدد', 'وزن' => null]))->toBe([['label' => 'تعداد', 'value' => '۳ عدد']])
        ->and(ProductContent::specs([]))->toBeNull()
        ->and(ProductContent::specs('nope'))->toBeNull();
});

it('makes the size chart rectangular', function (): void {
    expect(ProductContent::sizeChart([
        'columns' => ['سایز', 'سن', ''],
        'rows' => [['نوزاد', 'تا ۱ ماه', 'اضافه'], ['۰-۳ ماه'], ['', ''], 'bad'],
    ]))->toBe([
        'columns' => ['سایز', 'سن'],
        'rows' => [['نوزاد', 'تا ۱ ماه'], ['۰-۳ ماه', '']],
    ])
        ->and(ProductContent::sizeChart(['columns' => ['سایز'], 'rows' => []]))->toBeNull()
        ->and(ProductContent::sizeChart(null))->toBeNull();
});
