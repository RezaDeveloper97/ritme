<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Queries;

use App\Domain\Shop\Catalog\Models\Product;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Query\JoinClause;

/**
 * Indexable products for the sitemap: published, not demo, and no seo_meta row or one with sitemap_include and no
 * `noindex` robots.
 *
 * @phpstan-type SitemapProductRow array{slug: string, title: string, lastmod: string|null, coverMediaId: int|null}
 */
final class SitemapProducts
{
    /**
     * @return list<SitemapProductRow>
     */
    public function get(): array
    {
        $product = new Product;
        $table = $product->getTable();

        return Product::query()
            ->published()
            ->where("{$table}.is_demo", false)
            ->leftJoin('seo_meta', static function (JoinClause $join) use ($product, $table): void {
                $join->on('seo_meta.seoable_id', '=', "{$table}.id")
                    ->where('seo_meta.seoable_type', '=', $product->getMorphClass());
            })
            ->where(static function (Builder $q): void {
                $q->whereNull('seo_meta.id')->orWhere(static function (Builder $q): void {
                    $q->where('seo_meta.sitemap_include', true)
                        ->where(static fn (Builder $r) => $r->whereNull('seo_meta.robots')->orWhere('seo_meta.robots', 'not like', '%noindex%'));
                });
            })
            ->orderByDesc("{$table}.updated_at")
            ->orderByDesc("{$table}.id")
            ->get(["{$table}.slug", "{$table}.title", "{$table}.updated_at", "{$table}.cover_media_id"])
            ->map(static fn (Product $row): array => [
                'slug' => $row->slug,
                'title' => $row->title,
                'lastmod' => $row->updated_at?->toIso8601String(),
                'coverMediaId' => $row->cover_media_id,
            ])
            ->values()
            ->all();
    }
}
