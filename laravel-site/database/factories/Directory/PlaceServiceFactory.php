<?php

declare(strict_types=1);

namespace Database\Factories\Directory;

use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceService;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<PlaceService>
 */
final class PlaceServiceFactory extends Factory
{
    protected $model = PlaceService::class;

    public function definition(): array
    {
        return [
            'place_id' => Place::factory(),
            'name' => 'جلسه آزمایشی',
            'duration_minutes' => 45,
            'price' => 300_000,
            'price_unit' => 'هر جلسه',
            'sort_order' => 0,
        ];
    }
}
