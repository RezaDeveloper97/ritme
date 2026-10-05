<?php

declare(strict_types=1);

namespace App\Domain\Directory\Queries;

use App\Domain\Directory\Models\Place;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Query\JoinClause;

/**
 * Indexable places for the sitemap: published, not demo, and no seo_meta row or one with sitemap_include and no
 * `noindex` robots.
 *
 * @phpstan-type SitemapPlaceRow array{slug: string, name: string, lastmod: string|null, coverMediaId: int|null}
 */
final class SitemapPlaces
{
    /**
     * @return list<SitemapPlaceRow>
     */
    public function get(): array
    {
        $place = new Place;
        $table = $place->getTable();

        return array_values(Place::query()
            ->published()
            ->where("{$table}.is_demo", false)
            ->leftJoin('seo_meta', static function (JoinClause $join) use ($place, $table): void {
                $join->on('seo_meta.seoable_id', '=', "{$table}.id")
                    ->where('seo_meta.seoable_type', '=', $place->getMorphClass());
            })
            ->where(static function (Builder $q): void {
                $q->whereNull('seo_meta.id')->orWhere(static function (Builder $q): void {
                    $q->where('seo_meta.sitemap_include', true)
                        ->where(static fn (Builder $r) => $r->whereNull('seo_meta.robots')->orWhere('seo_meta.robots', 'not like', '%noindex%'));
                });
            })
            ->orderByDesc("{$table}.updated_at")
            ->orderByDesc("{$table}.id")
            ->get(["{$table}.slug", "{$table}.name", "{$table}.updated_at", "{$table}.cover_media_id"])
            ->map(static fn (Place $row): array => [
                'slug' => $row->slug,
                'name' => $row->name,
                'lastmod' => $row->updated_at?->toIso8601String(),
                'coverMediaId' => $row->cover_media_id,
            ])
            ->all());
    }
}
