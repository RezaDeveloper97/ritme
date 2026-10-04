<?php

declare(strict_types=1);

namespace App\Domain\Directory\Observers;

use App\Support\Cache\CacheBumpingObserver;
use Illuminate\Database\Eloquent\Model;

/**
 * Landing copy changes bump `directory` (+ `pages`).
 */
final class LandingObserver extends CacheBumpingObserver
{
    protected function namespaces(Model $model): array
    {
        return ['directory'];
    }
}
