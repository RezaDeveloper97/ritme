<?php

declare(strict_types=1);

namespace App\Domain\Search\Support;

use Carbon\CarbonImmutable;
use Illuminate\Contracts\Cache\Repository as Cache;

/**
 * Anonymised daily tally of searched terms for the admin «جستجوهای بی‌نتیجه» report (L7-05): only the normalised
 * query is kept — no IP, user agent, session or user id — and long digit runs (phone / card numbers) are masked.
 * Lives in the cache store for KEEP_DAYS (no table needed on cPanel); counts are best-effort (no locking).
 */
final class SearchTermLog
{
    public const KEEP_DAYS = 35;

    /** Distinct terms kept per day; later new terms are dropped, known ones still count. */
    public const MAX_TERMS_PER_DAY = 1000;

    public function __construct(private readonly Cache $cache) {}

    public function record(SearchTerms $terms, int $results): void
    {
        if ($terms->isEmpty()) {
            return;
        }

        $term = self::anonymise($terms->normalized);
        $key = self::key(CarbonImmutable::now());
        $day = $this->day(CarbonImmutable::now());

        if (! isset($day[$term]) && count($day) >= self::MAX_TERMS_PER_DAY) {
            return;
        }

        $day[$term] = ['count' => ($day[$term]['count'] ?? 0) + 1, 'results' => $results];
        $this->cache->put($key, $day, CarbonImmutable::now()->addDays(self::KEEP_DAYS));
    }

    /**
     * Terms of one day: term => [count, results of the latest search].
     *
     * @return array<string, array{count: int, results: int}>
     */
    public function day(CarbonImmutable $date): array
    {
        $day = $this->cache->get(self::key($date));

        return is_array($day) ? $day : [];
    }

    /**
     * Terms that found nothing on that day, most searched first.
     *
     * @return array<string, int> term => count
     */
    public function zeroResults(CarbonImmutable $date): array
    {
        $terms = [];
        foreach ($this->day($date) as $term => $row) {
            if ($row['results'] === 0) {
                $terms[$term] = $row['count'];
            }
        }
        arsort($terms);

        return $terms;
    }

    public static function anonymise(string $term): string
    {
        return preg_replace('/\d{6,}/', '#', $term) ?? $term;
    }

    private static function key(CarbonImmutable $date): string
    {
        return 'search-terms:'.$date->format('Y-m-d');
    }
}
