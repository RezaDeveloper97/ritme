<?php

declare(strict_types=1);

use App\Domain\Shop\Catalog\Data\CategoryData;
use App\Domain\Shop\Catalog\Data\CategoryTree;

function shopTree(): CategoryTree
{
    return new CategoryTree([
        new CategoryData(1, null, 'سیسمونی و نوزاد', 'baby'),
        new CategoryData(2, 1, 'لباس نوزاد', 'baby-clothes'),
        new CategoryData(3, 2, 'بادی', 'bodysuits'),
        new CategoryData(4, 1, 'تغذیه', 'feeding'),
        new CategoryData(5, null, 'آرایشی و بهداشتی', 'beauty'),
    ]);
}

it('navigates the category tree', function (): void {
    $tree = shopTree();

    expect(array_map(static fn ($c): string => $c->slug, $tree->roots()))->toBe(['baby', 'beauty'])
        ->and(array_map(static fn ($c): int => $c->id, $tree->children(1)))->toBe([2, 4])
        ->and(array_map(static fn ($c): string => $c->slug, $tree->path(3)))->toBe(['baby', 'baby-clothes', 'bodysuits'])
        ->and($tree->descendantIds(1))->toBe([1, 2, 4, 3])
        ->and($tree->descendantIds(5))->toBe([5])
        ->and($tree->descendantIds(99))->toBe([])
        ->and($tree->path(99))->toBe([])
        ->and($tree->findBySlug('feeding')?->id)->toBe(4)
        ->and($tree->findBySlug('nope'))->toBeNull();
});

it('round-trips through arrays for the cache', function (): void {
    $tree = shopTree();

    expect(CategoryTree::fromArray(json_decode(json_encode($tree->toArray(), JSON_THROW_ON_ERROR), true))->toArray())->toBe($tree->toArray());
});
