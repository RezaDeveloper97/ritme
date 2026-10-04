<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Queries;

use App\Domain\Shop\Catalog\Data\CategoryData;
use App\Domain\Shop\Catalog\Data\CategoryTree;
use App\Domain\Shop\Catalog\Models\Category;

/**
 * Builds the visible CategoryTree in one query: active categories whose every ancestor is active too (deactivating a
 * department hides its subcategories), ordered by sort_order, id.
 */
final class VisibleCategories
{
    public function get(): CategoryTree
    {
        $all = Category::query()->orderBy('sort_order')->orderBy('id')
            ->get(['id', 'parent_id', 'name', 'slug', 'intro', 'cover_media_id', 'illustration', 'sort_order', 'is_active'])
            ->keyBy('id');

        $visible = [];
        foreach ($all as $category) {
            $ok = true;
            $seen = [];
            for ($current = $category; $current !== null; $current = $current->parent_id === null ? null : $all->get($current->parent_id)) {
                if (! $current->is_active || isset($seen[$current->id])) {
                    $ok = false;
                    break;
                }
                $seen[$current->id] = true;
            }
            if ($ok) {
                $visible[] = CategoryData::fromModel($category);
            }
        }

        return new CategoryTree($visible);
    }
}
