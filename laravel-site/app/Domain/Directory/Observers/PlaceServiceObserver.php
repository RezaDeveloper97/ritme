<?php

declare(strict_types=1);

namespace App\Domain\Directory\Observers;

use App\Domain\Directory\Actions\RecalculatePlacePrice;
use App\Domain\Directory\Models\PlaceService;
use App\Support\Cache\CacheBumpingObserver;
use App\Support\Cache\NamespaceBumper;
use Illuminate\Database\Eloquent\Model;

/**
 * A service change recalculates the place's `price_from` and bumps `directory` (+ `pages`).
 */
final class PlaceServiceObserver extends CacheBumpingObserver
{
    public function __construct(NamespaceBumper $bumper, private readonly RecalculatePlacePrice $recalculate)
    {
        parent::__construct($bumper);
    }

    protected function namespaces(Model $model): array
    {
        return ['directory'];
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
        if ($model instanceof PlaceService) {
            $this->recalculate->handle($model->place_id);
            if ($model->wasChanged('place_id') && $model->getOriginal('place_id') !== null) {
                $this->recalculate->handle((int) $model->getOriginal('place_id'));
            }
        }
    }
}
