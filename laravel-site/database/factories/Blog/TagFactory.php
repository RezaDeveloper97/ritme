<?php

declare(strict_types=1);

namespace Database\Factories\Blog;

use App\Domain\Blog\Models\Tag;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<Tag>
 */
final class TagFactory extends Factory
{
    protected $model = Tag::class;

    public function definition(): array
    {
        $n = fake()->unique()->numberBetween(1, 1_000_000);

        return [
            'name' => "برچسب {$n}",
            'slug' => "tag-{$n}",
        ];
    }
}
