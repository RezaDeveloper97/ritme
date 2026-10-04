<?php

declare(strict_types=1);

namespace Database\Factories\Shop;

use App\Domain\Shop\Catalog\Models\Category;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<Category>
 */
final class CategoryFactory extends Factory
{
    protected $model = Category::class;

    public function definition(): array
    {
        $n = fake()->unique()->numberBetween(1, 1_000_000);

        return ['name' => "دسته آزمایشی {$n}", 'slug' => "test-category-{$n}"];
    }

    public function childOf(Category $parent): self
    {
        return $this->state(fn (): array => ['parent_id' => $parent->id]);
    }
}
