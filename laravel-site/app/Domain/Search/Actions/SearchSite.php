<?php

declare(strict_types=1);

namespace App\Domain\Search\Actions;

use App\Domain\Search\Data\SearchHit;
use App\Domain\Search\Data\SearchResults;
use App\Domain\Search\Support\SearchRegistry;
use App\Domain\Search\Support\SearchTermLog;
use App\Domain\Search\Support\SearchTerms;
use App\Support\Cache\CacheAside;
use App\Support\Cache\CacheKey;

/**
 * `/search?q=`: asks every registered provider, merges (title hits first, then provider order), paginates.
 * The merged list is cached briefly (TTL seconds) in the `pages` namespace, which every content change bumps, keyed
 * by the normalised query — «ي» and «ی» share one entry. First pages are recorded in the anonymised term log.
 */
final class SearchSite
{
    public const TTL = 300;

    public const PER_PROVIDER = 50;

    public function __construct(
        private readonly SearchRegistry $registry,
        private readonly CacheAside $cache,
        private readonly SearchTermLog $log,
    ) {}

    public function handle(?string $input, int $page = 1, int $perPage = 10): SearchResults
    {
        $terms = SearchTerms::fromInput($input);
        if ($terms->isEmpty()) {
            return SearchResults::empty($terms->query);
        }

        /** @var list<array<string, mixed>> $all */
        $all = $this->cache->remember(
            CacheKey::make('pages', 'search', sha1($terms->normalized)),
            self::TTL,
            fn (): array => array_map(static fn (SearchHit $hit): array => $hit->toArray(), $this->collect($terms)),
        );

        $page = max(1, $page);
        $perPage = max(1, min(50, $perPage));
        if ($page === 1) {
            $this->log->record($terms, count($all));
        }

        return new SearchResults(
            query: $terms->query,
            normalized: $terms->normalized,
            hits: array_map(SearchHit::fromArray(...), array_slice($all, ($page - 1) * $perPage, $perPage)),
            total: count($all),
            page: $page,
            perPage: $perPage,
        );
    }

    /**
     * @return list<SearchHit>
     */
    private function collect(SearchTerms $terms): array
    {
        $hits = [];
        foreach ($this->registry->all() as $order => $provider) {
            foreach ($provider->search($terms, self::PER_PROVIDER) as $rank => $hit) {
                $hits[] = ['hit' => $hit, 'order' => $order, 'rank' => $rank];
            }
        }

        usort($hits, static fn (array $a, array $b): int => [$b['hit']->score, $a['order'], $a['rank']] <=> [$a['hit']->score, $b['order'], $b['rank']]);

        return array_map(static fn (array $row): SearchHit => $row['hit'], $hits);
    }
}
