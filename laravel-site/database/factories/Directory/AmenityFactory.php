<?php

declare(strict_types=1);

namespace Database\Factories\Directory;

use App\Domain\Directory\Models\Amenity;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<Amenity>
 */
final class AmenityFactory extends Factory
{
    protected $model = Amenity::class;

    public function definition(): array
    {
        $n = fake()->unique()->numberBetween(1, 1_000_000);

        return ['name' => "امکان {$n}", 'slug' => "amenity-{$n}", 'icon' => 'check', 'is_filter' => false, 'sort_order' => 0];
    }

    public function filter(): self
    {
        return $this->state(fn (): array => ['is_filter' => true]);
    }
}
