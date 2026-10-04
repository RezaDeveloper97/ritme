<?php

declare(strict_types=1);

namespace Database\Factories\Directory;

use App\Domain\Directory\Enums\PlaceStatus;
use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceCategory;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<Place>
 */
final class PlaceFactory extends Factory
{
    protected $model = Place::class;

    public function definition(): array
    {
        $n = fake()->unique()->numberBetween(1, 1_000_000);

        return [
            'name' => "مجموعه آزمایشی {$n}",
            'slug' => "test-place-{$n}",
            'category_id' => PlaceCategory::factory(),
            'city_id' => City::factory(),
            'summary' => 'مجموعه‌ای آزمایشی برای مادر و کودک.',
            'description' => 'توضیح آزمایشی درباره مجموعه.',
            'address' => 'خیابان آزمایشی، پلاک ۱',
            'status' => PlaceStatus::Draft,
        ];
    }

    public function published(): self
    {
        return $this->state(fn (): array => ['status' => PlaceStatus::Published]);
    }

    public function verified(): self
    {
        return $this->state(fn (): array => ['is_verified' => true]);
    }

    public function demo(): self
    {
        return $this->state(fn (): array => ['is_demo' => true]);
    }

    /**
     * Saturday–Wednesday 09–20, Thursday 09–14, Friday closed (like the design's آب‌پری).
     */
    public function withHours(): self
    {
        $weekday = [['opens' => '09:00', 'closes' => '20:00']];

        return $this->state(fn (): array => ['opening_hours' => [
            'saturday' => $weekday, 'sunday' => $weekday, 'monday' => $weekday, 'tuesday' => $weekday, 'wednesday' => $weekday,
            'thursday' => [['opens' => '09:00', 'closes' => '14:00']],
            'friday' => [],
        ]]);
    }
}
