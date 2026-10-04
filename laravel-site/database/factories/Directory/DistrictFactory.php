<?php

declare(strict_types=1);

namespace Database\Factories\Directory;

use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\District;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<District>
 */
final class DistrictFactory extends Factory
{
    protected $model = District::class;

    public function definition(): array
    {
        $n = fake()->unique()->numberBetween(1, 1_000_000);

        return ['city_id' => City::factory(), 'name' => "محله {$n}", 'slug' => "district-{$n}", 'sort_order' => 0];
    }
}
