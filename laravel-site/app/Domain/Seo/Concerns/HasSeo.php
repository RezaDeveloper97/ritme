<?php

declare(strict_types=1);

namespace App\Domain\Seo\Concerns;

use App\Domain\Seo\Models\SeoMeta;
use Illuminate\Database\Eloquent\Relations\MorphOne;

/**
 * For models with a public page (posts, products, places…). Gives the model its `seoMeta` row; pages hand the
 * model to SeoManager::for($model), which reads the row through the cached SeoMetaRepository (not this relation).
 *
 * @phpstan-ignore trait.unused (first users: Blog/Shop/Directory models, L4–L6)
 */
trait HasSeo
{
    /**
     * @return MorphOne<SeoMeta, $this>
     */
    public function seoMeta(): MorphOne
    {
        return $this->morphOne(SeoMeta::class, 'seoable');
    }
}
