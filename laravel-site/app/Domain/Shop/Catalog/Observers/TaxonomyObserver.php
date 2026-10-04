<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Observers;

use App\Domain\Shop\Catalog\Models\Category;
use App\Support\Cache\CacheBumpingObserver;
use App\Support\Html\TextSlug;
use Illuminate\Database\Eloquent\Model;
use InvalidArgumentException;

/**
 * Categories and brands: an empty slug is filled from the name (unique within the table), a category's parent may
 * not be the category itself or one of its descendants, and any change bumps `shop` + `sitemap` (+ `pages`) —
 * product cards and listings embed these names.
 */
final class TaxonomyObserver extends CacheBumpingObserver
{
    protected function namespaces(Model $model): array
    {
        return ['shop', 'sitemap'];
    }

    public function saving(Model $model): void
    {
        if ($model instanceof Category) {
            $this->guardParent($model);
        }

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

    private function guardParent(Category $category): void
    {
        if ($category->parent_id === null || ! $category->exists || ! $category->isDirty('parent_id')) {
            return;
        }

        $seen = [];
        for ($id = $category->parent_id; $id !== null; $id = Category::query()->whereKey($id)->value('parent_id')) {
            $id = (int) $id;
            if ($id === $category->id || isset($seen[$id])) {
                throw new InvalidArgumentException('A category cannot be moved under itself or one of its subcategories.');
            }
            $seen[$id] = true;
        }
    }

    private function taken(Model $model, string $slug): bool
    {
        return $model->newQuery()
            ->where('slug', $slug)
            ->when($model->exists, static fn ($q) => $q->whereKeyNot($model->getKey()))
            ->exists();
    }
}
