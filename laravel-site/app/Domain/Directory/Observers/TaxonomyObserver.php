<?php

declare(strict_types=1);

namespace App\Domain\Directory\Observers;

use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\District;
use App\Support\Cache\CacheBumpingObserver;
use App\Support\Html\TextSlug;
use Illuminate\Database\Eloquent\Model;

/**
 * Cities, districts, categories and amenities: an empty slug is filled from the name (unique within the table;
 * districts within their city; city slugs never shadow the fixed `/directory/*` routes), and any change bumps
 * `directory` + `sitemap` (+ `pages`) — place cards and landing pages embed these names.
 */
final class TaxonomyObserver extends CacheBumpingObserver
{
    /** Fixed routes under /directory that a city slug must never shadow. */
    public const RESERVED_CITY_SLUGS = ['place', 'business', 'join', 'booked', 'page', 'search'];

    protected function namespaces(Model $model): array
    {
        return ['directory', 'sitemap'];
    }

    public function saving(Model $model): void
    {
        $slug = trim((string) $model->getAttribute('slug'));
        if ($slug !== '' && ! $model->isDirty('slug')) {
            return;
        }

        $base = TextSlug::make($slug !== '' ? $slug : (string) $model->getAttribute('name'));
        $base = $base === '' ? 'item' : $base;

        $candidate = $base;
        for ($n = 2; $this->taken($model, $candidate); $n++) {
            $candidate = "{$base}-{$n}";
        }
        $model->setAttribute('slug', $candidate);
    }

    private function taken(Model $model, string $slug): bool
    {
        if ($model instanceof City && in_array($slug, self::RESERVED_CITY_SLUGS, true)) {
            return true;
        }

        return $model->newQuery()
            ->where('slug', $slug)
            ->when($model instanceof District, static fn ($q) => $q->where('city_id', $model->getAttribute('city_id')))
            ->when($model->exists, static fn ($q) => $q->whereKeyNot($model->getKey()))
            ->exists();
    }
}
