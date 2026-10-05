<?php

declare(strict_types=1);

namespace App\Domain\Seo\Actions;

use App\Domain\Seo\Analysis\SeoAnalyzer;
use App\Domain\Settings\Contracts\SettingsRepository;

/**
 * Runs the content analyser (L7-02) for the given bulk-editor keys (or every page) and stores the score on the page's
 * seo_meta row (`score`, `score_checked_at`; a row is created when the page has none). The score never changes what a
 * page renders, so it is written quietly: no SeoMetaObserver, no cache bump, no activity entry.
 */
final class RecomputeSeoScores
{
    public function __construct(
        private readonly ResolveSeoTarget $targets,
        private readonly SeoAnalyzer $analyzer,
        private readonly SettingsRepository $settings,
        private readonly ListBulkSeoRows $rows,
    ) {}

    /**
     * @param  list<string>|null  $keys  null = every page of the bulk editor
     * @return array<string, int> key => score
     */
    public function handle(?array $keys = null): array
    {
        $keys ??= array_map(static fn (array $row): string => (string) $row['key'], $this->rows->handle());
        $seo = $this->settings->all()->seo;
        $scores = [];

        foreach (array_values(array_unique($keys)) as $key) {
            $meta = $this->targets->handle($key);
            if ($meta === null) {
                continue;
            }

            $input = $this->targets->analysisInput($key, $meta, $seo);
            if ($input === null) {
                continue;
            }

            $score = $this->analyzer->analyze($input)->score;
            $meta->forceFill(['score' => $score, 'score_checked_at' => now()])->saveQuietly();
            $scores[$key] = $score;
        }

        return $scores;
    }
}
