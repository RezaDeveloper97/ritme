<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Observers;

use App\Domain\Shop\Catalog\Actions\RecalculateProductRating;
use App\Domain\Shop\Catalog\Enums\ReviewStatus;
use App\Domain\Shop\Catalog\Models\ProductReview;
use App\Support\Cache\CacheBumpingObserver;
use App\Support\Cache\NamespaceBumper;
use Illuminate\Database\Eloquent\Model;

/**
 * Stamps `approved_at` on approval, recalculates the product's cached rating (approved, non-demo reviews only) and
 * bumps `shop` (+ `pages`).
 */
final class ProductReviewObserver extends CacheBumpingObserver
{
    public function __construct(NamespaceBumper $bumper, private readonly RecalculateProductRating $recalculate)
    {
        parent::__construct($bumper);
    }

    protected function namespaces(Model $model): array
    {
        return ['shop'];
    }

    public function saving(ProductReview $review): void
    {
        if ($review->status === ReviewStatus::Approved) {
            $review->approved_at ??= now();
        } else {
            $review->approved_at = null;
        }
    }

    public function saved(Model $model): void
    {
        $this->refresh($model);
        parent::saved($model);
    }

    public function deleted(Model $model): void
    {
        $this->refresh($model);
        parent::deleted($model);
    }

    private function refresh(Model $model): void
    {
        if ($model instanceof ProductReview) {
            $this->recalculate->handle($model->product_id);
        }
    }
}
