<?php

declare(strict_types=1);

namespace Database\Factories\Shop;

use App\Domain\Shop\Catalog\Enums\ReviewStatus;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Models\ProductReview;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<ProductReview>
 */
final class ProductReviewFactory extends Factory
{
    protected $model = ProductReview::class;

    public function definition(): array
    {
        return [
            'product_id' => Product::factory(),
            'author_name' => 'خریدار آزمایشی',
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
