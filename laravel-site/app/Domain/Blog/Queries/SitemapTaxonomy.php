<?php

declare(strict_types=1);

namespace App\Domain\Blog\Queries;

use App\Domain\Blog\Models\Author;
use App\Domain\Blog\Models\Post;
use App\Domain\Blog\Models\Tag;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Query\JoinClause;
use Illuminate\Support\Carbon;

/**
 * Sitemap rows for tag pages (`forTags`, at least `$minPosts` published posts) and author pages (`forAuthors`, at
 * least one published post), skipping rows an admin set to noindex / out of the sitemap in seo_meta; lastmod = the
 * newest content date of their published posts.
 *
 * @phpstan-type SitemapTaxonomyRow array{slug: string, lastmod: string|null}
 */
final class SitemapTaxonomy
{
    /**
     * @return list<SitemapTaxonomyRow>
     */
    public static function forTags(int $minPosts): array
    {
        return self::rows(new Tag, max(1, $minPosts));
    }

    /**
     * @return list<SitemapTaxonomyRow>
     */
    public static function forAuthors(): array
    {
        return self::rows(new Author, 1);
    }

    /**
     * @return list<SitemapTaxonomyRow>
     */
    private static function rows(Tag|Author $model, int $minPosts): array
    {
        $table = $model->getTable();
        $published = static function (Builder $q): void {
            /** @var Builder<Post> $q */
            $q->published();
        };

        return $model->newQuery()
            ->leftJoin('seo_meta', static function (JoinClause $join) use ($model, $table): void {
                $join->on('seo_meta.seoable_id', '=', "{$table}.id")
                    ->where('seo_meta.seoable_type', '=', $model->getMorphClass());
            })
            ->where(static function (Builder $q): void {
                $q->whereNull('seo_meta.id')->orWhere(static function (Builder $q): void {
                    $q->where('seo_meta.sitemap_include', true)
                        ->where(static fn (Builder $r) => $r->whereNull('seo_meta.robots')->orWhere('seo_meta.robots', 'not like', '%noindex%'));
                });
            })
            ->select(["{$table}.id", "{$table}.slug"])
            ->withCount(['posts as published_posts_count' => $published])
            ->withMax(['posts as last_content_at' => $published], 'updated_content_at')
            ->orderBy("{$table}.id")
            ->get()
            ->filter(static fn (Model $row): bool => (int) $row->getAttribute('published_posts_count') >= $minPosts)
            ->map(static function (Model $row): array {
                $last = $row->getAttribute('last_content_at');

                return [
                    'slug' => (string) $row->getAttribute('slug'),
                    'lastmod' => $last === null ? null : Carbon::parse((string) $last)->toIso8601String(),
                ];
            })
            ->values()
            ->all();
    }
}
