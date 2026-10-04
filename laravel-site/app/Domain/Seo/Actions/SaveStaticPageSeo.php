<?php

declare(strict_types=1);

namespace App\Domain\Seo\Actions;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\Support\Robots;

/**
 * Saves the admin SEO overrides of a static page into its `seo_meta` row (by route name; created on first save).
 * Empty values are stored as null (= inherit the page default). SeoMetaObserver bumps `seo`, `sitemap` and `pages`,
 * so the next request of the page — full-page cache included — renders the new head.
 */
final class SaveStaticPageSeo
{
    /** Columns the static-page editor may write. */
    public const FIELDS = [
        'title', 'description', 'focus_keyword', 'canonical_url', 'robots',
        'og_title', 'og_description', 'og_media_id', 'sitemap_include', 'sitemap_priority',
    ];

    /**
     * @param  array<string, mixed>  $data  seo_meta columns (unknown keys are ignored)
     */
    public function handle(StaticPage $page, array $data): SeoMeta
    {
        $values = [];
        foreach (self::FIELDS as $field) {
            if (array_key_exists($field, $data)) {
                $values[$field] = $data[$field];
            }
        }

        foreach (['title', 'description', 'focus_keyword', 'canonical_url', 'og_title', 'og_description'] as $field) {
            if (array_key_exists($field, $values)) {
                $text = is_scalar($values[$field]) ? trim((string) $values[$field]) : '';
                $values[$field] = $text === '' ? null : $text;
            }
        }
        if (array_key_exists('robots', $values)) {
            $robots = is_string($values['robots']) ? trim($values['robots']) : '';
            $values['robots'] = $robots === '' ? null : (string) Robots::parse($robots);
        }
        if (array_key_exists('og_media_id', $values)) {
            $values['og_media_id'] = is_numeric($values['og_media_id']) ? (int) $values['og_media_id'] : null;
        }
        if (array_key_exists('sitemap_include', $values)) {
            $values['sitemap_include'] = (bool) $values['sitemap_include'];
        }
        if (array_key_exists('sitemap_priority', $values)) {
            $priority = is_numeric($values['sitemap_priority']) ? (float) $values['sitemap_priority'] : null;
            $values['sitemap_priority'] = $priority === null ? null : max(0.0, min(1.0, $priority));
        }

        $meta = SeoMeta::query()->firstOrNew(['route_name' => $page->routeName()]);
        $meta->fill($values)->save();

        return $meta;
    }
}
