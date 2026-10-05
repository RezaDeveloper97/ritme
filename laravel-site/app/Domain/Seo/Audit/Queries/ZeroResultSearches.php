<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit\Queries;

use App\Domain\Search\Support\SearchTermLog;
use Carbon\CarbonImmutable;

/**
 * Site-search terms that found nothing over the last days (anonymised daily tallies of SearchTermLog, L4-04), summed
 * per term — content ideas and missing synonyms.
 */
final class ZeroResultSearches
{
    public const DAYS = 7;

    public function __construct(private readonly SearchTermLog $log) {}

    /**
     * @return list<array{term: string, count: int}> most searched first
     */
    public function top(int $limit = 10, int $days = self::DAYS): array
    {
        $totals = [];
        $today = CarbonImmutable::now();
        for ($i = 0; $i < $days; $i++) {
            foreach ($this->log->zeroResults($today->subDays($i)) as $term => $count) {
                $totals[(string) $term] = ($totals[(string) $term] ?? 0) + $count;
            }
        }
        arsort($totals);

        $rows = [];
        foreach (array_slice($totals, 0, $limit, true) as $term => $count) {
            $rows[] = ['term' => (string) $term, 'count' => $count];
        }

        return $rows;
    }
}
