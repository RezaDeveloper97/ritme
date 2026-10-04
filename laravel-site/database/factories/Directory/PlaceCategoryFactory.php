<?php

declare(strict_types=1);

namespace Database\Factories\Directory;

use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Seo\Schema\Enums\LocalBusinessType;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<PlaceCategory>
 */
final class PlaceCategoryFactory extends Factory
{
    protected $model = PlaceCategory::class;

    public function definition(): array
    {
        $n = fake()->unique()->numberBetween(1, 1_000_000);

        return [
            'name' => "دسته خدمات {$n}",
            'slug' => "place-category-{$n}",
            'schema_type' => LocalBusinessType::LocalBusiness,
            'sort_order' => 0,
            'is_active' => true,
        ];
    }
}
