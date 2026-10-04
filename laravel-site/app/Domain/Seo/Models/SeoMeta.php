<?php

declare(strict_types=1);

namespace App\Domain\Seo\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\MorphTo;

/**
 * Per-page SEO overrides for a model (morph `seoable`) or a static page (`route_name`). Null columns inherit.
 * Read through SeoMetaRepository, never directly.
 *
 * @property int $id
 * @property string|null $seoable_type
 * @property int|null $seoable_id
 * @property string|null $route_name
 * @property string|null $title
 * @property string|null $description
 * @property string|null $canonical_url
 * @property string|null $robots
 * @property string|null $og_title
 * @property string|null $og_description
 * @property int|null $og_media_id
 * @property string|null $og_type
 * @property string|null $twitter_card
 * @property string|null $focus_keyword
 * @property array<string, mixed>|null $schema_overrides
 * @property bool $sitemap_include
 * @property string|null $sitemap_priority
 * @property string|null $sitemap_changefreq
 */
final class SeoMeta extends Model
{
    protected $table = 'seo_meta';

    protected $fillable = [
        'seoable_type', 'seoable_id', 'route_name',
        'title', 'description', 'canonical_url', 'robots',
        'og_title', 'og_description', 'og_media_id', 'og_type', 'twitter_card',
        'focus_keyword', 'schema_overrides',
        'sitemap_include', 'sitemap_priority', 'sitemap_changefreq',
    ];

    protected $attributes = [
        'sitemap_include' => true,
    ];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'seoable_id' => 'integer',
            'og_media_id' => 'integer',
            'schema_overrides' => 'json:unicode',
            'sitemap_include' => 'boolean',
            'sitemap_priority' => 'decimal:1',
        ];
    }

    /**
     * @return MorphTo<Model, $this>
     */
    public function seoable(): MorphTo
    {
        return $this->morphTo();
    }
}
