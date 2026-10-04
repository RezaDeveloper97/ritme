<?php

declare(strict_types=1);

namespace App\Support\Cache;

use Illuminate\Database\Eloquent\Model;

/**
 * Base for domain observers that only (or also) invalidate caches:
 *
 *     final class PostObserver extends CacheBumpingObserver
 *     {
 *         protected function namespaces(Model $model): array { return ['blog', 'sitemap']; }
 *     }
 *
 * Registered via a context provider's $observers. `pages` is always added (cacheaside.always_bump).
 */
abstract class CacheBumpingObserver
{
    public function __construct(private readonly NamespaceBumper $bumper) {}

    /**
     * @return list<string>
     */
    abstract protected function namespaces(Model $model): array;

    public function saved(Model $model): void
    {
        $this->bump($model);
    }

    public function deleted(Model $model): void
    {
        $this->bump($model);
    }

    public function restored(Model $model): void
    {
        $this->bump($model);
    }

    protected function bump(Model $model): void
    {
        $this->bumper->bumpFor($model, $this->namespaces($model));
    }
}
