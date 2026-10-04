<?php

declare(strict_types=1);

namespace App\Domain\Blog\Observers;

use App\Support\Cache\CacheBumpingObserver;
use App\Support\Html\TextSlug;
use Illuminate\Database\Eloquent\Model;

/**
 * Categories, tags and authors: an empty slug is filled from the name (unique within the table), and any change
 * bumps `blog` + `sitemap` (+ `pages` via cacheaside.always_bump) — post cards embed category data, articles embed
 * tags and author/reviewer boxes.
 */
final class TaxonomyObserver extends CacheBumpingObserver
{
    protected function namespaces(Model $model): array
    {
        return ['blog', 'sitemap'];
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
        return $model->newQuery()
            ->where('slug', $slug)
            ->when($model->exists, static fn ($q) => $q->whereKeyNot($model->getKey()))
            ->exists();
    }
}
