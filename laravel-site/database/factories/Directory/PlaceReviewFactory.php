<?php

declare(strict_types=1);

namespace Database\Factories\Directory;

use App\Domain\Directory\Enums\ReviewStatus;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceReview;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<PlaceReview>
 */
final class PlaceReviewFactory extends Factory
{
    protected $model = PlaceReview::class;

    public function definition(): array
    {
        return [
            'place_id' => Place::factory(),
            'author_name' => 'کاربر آزمایشی',
            'rating' => 5,
            'body' => 'نظر آزمایشی.',
            'status' => ReviewStatus::Pending,
        ];
    }

    public function approved(): self
    {
        return $this->state(fn (): array => ['status' => ReviewStatus::Approved]);
    }

    public function rejected(): self
    {
        return $this->state(fn (): array => ['status' => ReviewStatus::Rejected]);
    }

    public function demo(): self
    {
        return $this->state(fn (): array => ['is_demo' => true]);
    }
}
