<?php

declare(strict_types=1);

namespace Database\Factories\Blog;

use App\Domain\Blog\Models\Category;
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

        return [
            'name' => "دسته {$n}",
            'slug' => "category-{$n}",
            'sort_order' => 0,
        ];
    }
}
