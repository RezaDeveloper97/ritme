<?php

declare(strict_types=1);

namespace App\Domain\Seo\Actions;

use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\Observers\SeoMetaObserver;

/**
 * "Reset to default" for many pages of the bulk SEO editor (L7-06): deletes their seo_meta rows (every override,
 * the cornerstone flag and the stored score), so they render their defaults again. Deletes go through the model
 * (SeoMetaObserver) inside one batch → caches bumped once. Scores are then recomputed for the defaults.
 */
final class ResetSeoMeta
{
    public function __construct(
        private readonly ResolveSeoTarget $targets,
        private readonly RecomputeSeoScores $scores,
    ) {}

    /**
     * @param  list<string>  $keys
     * @return list<string> keys whose overrides were removed
     */
    public function handle(array $keys): array
    {
        $reset = SeoMetaObserver::batch(fn (): array => SeoMeta::query()->getConnection()->transaction(function () use ($keys): array {
            $reset = [];
            foreach (array_values(array_unique($keys)) as $key) {
                $meta = $this->targets->handle($key);
                if ($meta !== null && $meta->exists && (bool) $meta->delete()) {
                    $reset[] = $key;
                }
            }

            return $reset;
        }));

        if ($reset !== []) {
            $this->scores->handle($reset);
        }

        return $reset;
    }
}
