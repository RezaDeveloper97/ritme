<?php

declare(strict_types=1);

namespace App\Domain\Directory\Observers;

use App\Domain\Directory\Actions\RecalculatePlaceRating;
use App\Domain\Directory\Enums\ReviewStatus;
use App\Domain\Directory\Models\PlaceReview;
use App\Support\Cache\CacheBumpingObserver;
use App\Support\Cache\NamespaceBumper;
use Illuminate\Database\Eloquent\Model;

/**
 * Stamps `approved_at` on approval, recalculates the place's cached rating (approved, non-demo reviews only) and bumps
 * `directory` (+ `pages`).
 */
final class PlaceReviewObserver extends CacheBumpingObserver
{
    public function __construct(NamespaceBumper $bumper, private readonly RecalculatePlaceRating $recalculate)
    {
        parent::__construct($bumper);
    }

    protected function namespaces(Model $model): array
    {
        return ['directory'];
    }

    public function saving(PlaceReview $review): void
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
        if ($model instanceof PlaceReview) {
            $this->recalculate->handle($model->place_id);
        }
    }
}
