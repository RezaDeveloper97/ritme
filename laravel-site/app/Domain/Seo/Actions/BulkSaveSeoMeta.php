<?php

declare(strict_types=1);

namespace App\Domain\Seo\Actions;

use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\Observers\SeoMetaObserver;
use App\Domain\Seo\Support\Robots;

/**
 * Saves many pages' SEO overrides in one go (bulk SEO editor, L7-06): title, description, robots and the cornerstone
 * flag, keyed by ResolveSeoTarget keys. One transaction; every row is saved through the model (SeoMetaObserver), inside
 * SeoMetaObserver::batch() so `seo`, `sitemap` and `pages` are bumped once for the whole batch. Unchanged rows are not
 * written. The analyser score of every changed page is recomputed afterwards (RecomputeSeoScores).
 *
 * Empty title / description / robots = inherit (stored as null), like the per-page editors.
 */
final class BulkSaveSeoMeta
{
    public const FIELDS = ['title', 'description', 'robots', 'cornerstone'];

    public function __construct(
        private readonly ResolveSeoTarget $targets,
        private readonly RecomputeSeoScores $scores,
    ) {}

    /**
     * @param  array<string, mixed>  $changes  key => array, subset of FIELDS (unknown keys / fields / non-arrays ignored)
     * @return array<string, array{old: array<string, mixed>, new: array<string, mixed>}> changed keys with old/new values
     */
    public function handle(array $changes): array
    {
        $changed = SeoMetaObserver::batch(fn (): array => SeoMeta::query()->getConnection()->transaction(function () use ($changes): array {
            $changed = [];
            foreach ($changes as $key => $fields) {
                if (! is_array($fields)) {
                    continue;
                }

                $meta = $this->targets->handle((string) $key);
                $values = self::normalize($fields);
                if ($meta === null || $values === [] || (! $meta->exists && array_filter($values) === [])) {
                    continue; // unknown page, nothing to write, or "inherit everything" on a page without a row
                }

                $old = [];
                foreach ($values as $field => $value) {
                    $old[$field] = $meta->getAttribute($field);
                }
                $meta->fill($values);
                if ($meta->exists && ! $meta->isDirty()) {
                    continue;
                }

                $dirty = $meta->exists ? array_keys($meta->getDirty()) : array_keys($values);
                $meta->save();
                $changed[(string) $key] = [
                    'old' => array_intersect_key($old, array_flip($dirty)),
                    'new' => array_intersect_key($values, array_flip($dirty)),
                ];
            }

            return $changed;
        }));

        if ($changed !== []) {
            $this->scores->handle(array_keys($changed));
        }

        return $changed;
    }

    /**
     * @param  array<string, mixed>  $fields
     * @return array<string, mixed>
     */
    public static function normalize(array $fields): array
    {
        $values = [];
        foreach (['title' => 255, 'description' => 500] as $field => $max) {
            if (array_key_exists($field, $fields)) {
                $text = is_scalar($fields[$field]) ? trim((string) preg_replace('/\s+/u', ' ', (string) $fields[$field])) : '';
                $values[$field] = $text === '' ? null : mb_substr($text, 0, $max);
            }
        }
        if (array_key_exists('robots', $fields)) {
            $robots = is_string($fields['robots']) ? trim($fields['robots']) : '';
            $values['robots'] = $robots === '' ? null : (string) Robots::parse($robots);
        }
        if (array_key_exists('cornerstone', $fields)) {
            $values['cornerstone'] = filter_var($fields['cornerstone'], FILTER_VALIDATE_BOOLEAN);
        }

        return $values;
    }
}
