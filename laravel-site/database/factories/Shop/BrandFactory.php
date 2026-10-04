<?php

declare(strict_types=1);

namespace Database\Factories\Shop;

use App\Domain\Shop\Catalog\Models\Brand;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<Brand>
 */
final class BrandFactory extends Factory
{
    protected $model = Brand::class;

    public function definition(): array
    {
        $n = fake()->unique()->numberBetween(1, 1_000_000);

        return ['name' => "برند آزمایشی {$n}", 'slug' => "test-brand-{$n}"];
    }
}
