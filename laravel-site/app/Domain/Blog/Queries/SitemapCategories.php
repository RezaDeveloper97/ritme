<?php

declare(strict_types=1);

namespace App\Domain\Blog\Queries;

use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Query\JoinClause;
use Illuminate\Support\Carbon;

/**
 * Indexable categories with at least one published post (empty categories are thin pages); lastmod = the newest
 * content date of their published posts.
 *
 * @phpstan-type SitemapCategoryRow array{slug: string, lastmod: string|null}
 */
final class SitemapCategories
{
    /**
     * @return list<SitemapCategoryRow>
     */
    public function get(): array
    {
        $category = new Category;
        $table = $category->getTable();
        $published = static function (Builder $q): void {
            /** @var Builder<Post> $q */
            $q->published();
        };

        return array_values(Category::query()
            ->leftJoin('seo_meta', static function (JoinClause $join) use ($category, $table): void {
                $join->on('seo_meta.seoable_id', '=', "{$table}.id")
                    ->where('seo_meta.seoable_type', '=', $category->getMorphClass());
            })
            ->where(static function (Builder $q): void {
                $q->whereNull('seo_meta.id')->orWhere(static function (Builder $q): void {
                    $q->where('seo_meta.sitemap_include', true)
                        ->where(static fn (Builder $r) => $r->whereNull('seo_meta.robots')->orWhere('seo_meta.robots', 'not like', '%noindex%'));
                });
            })
            ->select(["{$table}.id", "{$table}.slug", "{$table}.sort_order"])
            ->withCount(['posts as published_posts_count' => $published])
            ->withMax(['posts as last_content_at' => $published], 'updated_content_at')
            ->orderBy("{$table}.sort_order")
            ->orderBy("{$table}.id")
            ->get()
            ->filter(static fn (Category $row): bool => (int) $row->getAttribute('published_posts_count') > 0)
            ->map(static function (Category $row): array {
                $last = $row->getAttribute('last_content_at');

                return [
                    'slug' => $row->slug,
                    'lastmod' => $last === null ? null : Carbon::parse((string) $last)->toIso8601String(),
                ];
            })
            ->all());
    }
}
