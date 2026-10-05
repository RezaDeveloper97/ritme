<?php

declare(strict_types=1);

namespace App\Domain\Blog\Queries;

use App\Domain\Blog\Models\Post;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Query\JoinClause;

/**
 * Indexable published posts for the sitemap: no seo_meta row, or one with sitemap_include and no `noindex` robots.
 *
 * @phpstan-type SitemapPostRow array{slug: string, title: string, lastmod: string|null, coverMediaId: int|null}
 */
final class SitemapPosts
{
    /**
     * @return list<SitemapPostRow>
     */
    public function get(): array
    {
        $post = new Post;
        $table = $post->getTable();

        return array_values(Post::query()
            ->published()
            ->leftJoin('seo_meta', static function (JoinClause $join) use ($post, $table): void {
                $join->on('seo_meta.seoable_id', '=', "{$table}.id")
                    ->where('seo_meta.seoable_type', '=', $post->getMorphClass());
            })
            ->where(static function (Builder $q): void {
                $q->whereNull('seo_meta.id')->orWhere(static function (Builder $q): void {
                    $q->where('seo_meta.sitemap_include', true)
                        ->where(static fn (Builder $r) => $r->whereNull('seo_meta.robots')->orWhere('seo_meta.robots', 'not like', '%noindex%'));
                });
            })
            ->orderByDesc("{$table}.published_at")
            ->orderByDesc("{$table}.id")
            ->get(["{$table}.slug", "{$table}.title", "{$table}.published_at", "{$table}.updated_content_at", "{$table}.cover_media_id"])
            ->map(static function (Post $row): array {
                $lastmod = $row->updated_content_at !== null && $row->published_at !== null
                    ? $row->updated_content_at->max($row->published_at)
                    : ($row->updated_content_at ?? $row->published_at);

                return [
                    'slug' => $row->slug,
                    'title' => $row->title,
                    'lastmod' => $lastmod?->toIso8601String(),
                    'coverMediaId' => $row->cover_media_id,
                ];
            })
            ->all());
    }
}
