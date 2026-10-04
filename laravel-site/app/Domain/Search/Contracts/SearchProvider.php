<?php

declare(strict_types=1);

namespace App\Domain\Search\Contracts;

use App\Domain\Search\Data\SearchHit;
use App\Domain\Search\Support\SearchTerms;

/**
 * One searchable content type. Implementations read published content only, match every token (AND) against the
 * normalised text and return at most $limit hits, best first. Register with `SearchRegistry::TAG`.
 */
interface SearchProvider
{
    /** Stable machine key (`posts`, `faq`, …), used for ordering and in result markup. */
    public function key(): string;

    /** Persian label of the content type shown next to each hit. */
    public function label(): string;

    /**
     * @return list<SearchHit>
     */
    public function search(SearchTerms $terms, int $limit): array;
}
