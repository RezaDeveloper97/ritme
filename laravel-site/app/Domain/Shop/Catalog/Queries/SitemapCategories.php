<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Queries;

use App\Domain\Shop\Catalog\Models\Category;
use App\Domain\Shop\Catalog\Models\Product;
use Carbon\CarbonImmutable;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Query\JoinClause;

/**
 * Indexable category listings for the sitemap: visible categories (VisibleCategories) whose subtree holds at least one
 * published, non-demo product — a list of demo samples only is thin content — and that have no seo_meta row or one
 * with sitemap_include and no `noindex`. lastmod = the newest such product change in the subtree.
 *
 * @phpstan-type SitemapCategoryRow array{slug: string, name: string, lastmod: string|null, coverMediaId: int|null}
 */
final class SitemapCategories
{
    /**
     * @return list<SitemapCategoryRow>
     */
    public function get(): array
    {
        $tree = (new VisibleCategories)->get();
        if ($tree->categories === []) {
            return [];
        }

        $products = (new Product)->getTable();
        $latest = Product::query()->getConnection()->table('shop_category_product')
            ->join($products, "{$products}.id", '=', 'shop_category_product.product_id')
            ->where("{$products}.is_published", true)
            ->where("{$products}.is_demo", false)
            ->groupBy('shop_category_product.category_id')
            ->selectRaw("shop_category_product.category_id as category_id, MAX({$products}.updated_at) as lastmod")
            ->pluck('lastmod', 'category_id');

        $excluded = array_flip($this->excludedIds());

        $rows = [];
        foreach ($tree->categories as $category) {
            if (isset($excluded[$category->id])) {
                continue;
            }
            $lastmod = null;
            foreach ($tree->descendantIds($category->id) as $id) {
                $value = $latest[$id] ?? null;
                if (is_string($value) && ($lastmod === null || $value > $lastmod)) {
                    $lastmod = $value;
                }
            }
            if ($lastmod === null) {
                continue;
            }
            $rows[] = [
                'slug' => $category->slug,
                'name' => $category->name,
                'lastmod' => CarbonImmutable::parse($lastmod)->toIso8601String(),
                'coverMediaId' => $category->coverMediaId,
            ];
        }

        return $rows;
    }

    /**
     * Categories whose admin SEO settings keep them out of the sitemap.
     *
     * @return list<int>
     */
    private function excludedIds(): array
    {
        $category = new Category;
        $table = $category->getTable();

        return Category::query()
            ->join('seo_meta', static function (JoinClause $join) use ($category, $table): void {
                $join->on('seo_meta.seoable_id', '=', "{$table}.id")
                    ->where('seo_meta.seoable_type', '=', $category->getMorphClass());
            })
            ->where(static fn (Builder $q) => $q->where('seo_meta.sitemap_include', false)->orWhere('seo_meta.robots', 'like', '%noindex%'))
            ->pluck("{$table}.id")
            ->map(intval(...))
            ->values()
            ->all();
    }
}
