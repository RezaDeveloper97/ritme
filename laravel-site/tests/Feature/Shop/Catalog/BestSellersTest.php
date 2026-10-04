<?php

declare(strict_types=1);

use App\Domain\Shop\Catalog\Actions\SyncCrossSells;
use App\Domain\Shop\Catalog\Actions\SyncProductCategories;
use App\Domain\Shop\Catalog\Contracts\ProductRepository;
use App\Domain\Shop\Catalog\Models\Category;
use App\Domain\Shop\Catalog\Models\Product;

beforeEach(function (): void {
    $this->baby = Category::factory()->create();
    $this->clothes = Category::factory()->childOf($this->baby)->create();
    $this->beauty = Category::factory()->create();
    $this->make = function (string $slug, Category $category, array $extra = []): Product {
        $product = Product::factory()->published()->create(['slug' => $slug, ...$extra]);
        app(SyncProductCategories::class)->handle($product, [$category->id]);

        return $product;
    };
});

it('ranks purchasable best sellers by sales, then featured and sort order', function (): void {
    ($this->make)('a', $this->clothes, ['sales_count' => 3]);
    ($this->make)('b', $this->clothes, ['sales_count' => 10]);
    ($this->make)('c', $this->clothes, ['is_featured' => true]);
    ($this->make)('d', $this->beauty, ['sales_count' => 50]);
    ($this->make)('gone', $this->clothes, ['sales_count' => 99, 'stock_qty' => 0]);
    Product::factory()->create(['slug' => 'draft', 'sales_count' => 100]);

    $repo = app(ProductRepository::class);

    expect(array_map(static fn ($c): string => $c->slug, $repo->bestSellers(10, $this->baby->id)))->toBe(['b', 'a', 'c'])
        ->and(array_map(static fn ($c): string => $c->slug, $repo->bestSellers(2)))->toBe(['d', 'b']);
});

it('lists hand-picked cross-sells first, then category and shop best sellers', function (): void {
    $main = ($this->make)('main', $this->clothes);
    $picked = ($this->make)('picked', $this->beauty);
    $hidden = Product::factory()->create(['slug' => 'hidden']);
    ($this->make)('sibling', $this->clothes, ['sales_count' => 2]);
    ($this->make)('cousin', $this->baby, ['sales_count' => 1]);
    ($this->make)('elsewhere', $this->beauty, ['sales_count' => 40]);

    app(SyncCrossSells::class)->handle($main, [$hidden->id, $picked->id]);

    $slugs = array_map(static fn ($c): string => $c->slug, app(ProductRepository::class)->frequentlyBoughtWith($main->id, 3));
    expect($slugs)->toBe(['picked', 'sibling', 'elsewhere']);

    $all = array_map(static fn ($c): string => $c->slug, app(ProductRepository::class)->frequentlyBoughtWith($main->id, 10));
    expect($all)->not->toContain('main')->not->toContain('hidden')
        ->and(app(ProductRepository::class)->frequentlyBoughtWith(999_999))->toBe([]);
});
