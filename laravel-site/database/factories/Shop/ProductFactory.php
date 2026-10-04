<?php

declare(strict_types=1);

namespace Database\Factories\Shop;

use App\Domain\Shop\Catalog\Models\Product;
use App\Support\Money\Money;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * Prices are Money (integer rials); the default is 250٬000 toman.
 *
 * @extends Factory<Product>
 */
final class ProductFactory extends Factory
{
    protected $model = Product::class;

    public function definition(): array
    {
        $n = fake()->unique()->numberBetween(1, 1_000_000);

        return [
            'title' => "محصول آزمایشی {$n}",
            'slug' => "test-product-{$n}",
            'short_description' => 'توضیح کوتاه آزمایشی.',
            'description' => '<p>توضیح آزمایشی محصول.</p>',
            'price' => Money::fromToman(250_000),
            'stock_qty' => 10,
            'is_published' => false,
        ];
    }

    public function published(): self
    {
        return $this->state(fn (): array => ['is_published' => true]);
    }

    public function demo(): self
    {
        return $this->state(fn (): array => ['is_demo' => true]);
    }

    public function featured(): self
    {
        return $this->state(fn (): array => ['is_featured' => true]);
    }

    public function outOfStock(): self
    {
        return $this->state(fn (): array => ['stock_qty' => 0]);
    }

    public function priced(int $toman, ?int $compareAtToman = null): self
    {
        return $this->state(fn (): array => [
            'price' => Money::fromToman($toman),
            'compare_at_price' => $compareAtToman === null ? null : Money::fromToman($compareAtToman),
        ]);
    }
}
