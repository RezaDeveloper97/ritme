<?php

declare(strict_types=1);

namespace Database\Factories\Directory;

use App\Domain\Directory\Models\City;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<City>
 */
final class CityFactory extends Factory
{
    protected $model = City::class;

    public function definition(): array
    {
        $n = fake()->unique()->numberBetween(1, 1_000_000);

        return ['name' => "شهر {$n}", 'slug' => "city-{$n}", 'sort_order' => 0, 'is_active' => true];
    }
}
