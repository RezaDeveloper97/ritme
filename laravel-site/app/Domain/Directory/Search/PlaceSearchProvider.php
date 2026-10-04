<?php

declare(strict_types=1);

namespace App\Domain\Directory\Search;

use App\Domain\Directory\Contracts\PlaceRepository;
use App\Domain\Directory\Data\PlaceCardData;
use App\Domain\Directory\Data\PlaceSearchCriteria;
use App\Domain\Directory\Support\DirectoryUrls;
use App\Domain\Search\Contracts\SearchProvider;
use App\Domain\Search\Data\SearchHit;
use App\Domain\Search\Support\SearchTerms;
use Illuminate\Support\Str;

/**
 * Published directory places in site search (`/search`, L4-04), through the cached SearchPlaces query: name, summary,
 * description, address and category name. Registered with SearchRegistry::TAG by DirectoryServiceProvider.
 */
final class PlaceSearchProvider implements SearchProvider
{
    public function __construct(private readonly PlaceRepository $places, private readonly DirectoryUrls $urls) {}

    public function key(): string
    {
        return 'places';
    }

    public function label(): string
    {
        return 'خدمات مادر و کودک';
    }

    public function search(SearchTerms $terms, int $limit): array
    {
        if ($terms->isEmpty()) {
            return [];
        }

        $page = $this->places->search(new PlaceSearchCriteria(text: $terms->normalized, perPage: max(1, min(PlaceSearchCriteria::MAX_PER_PAGE, $limit))));

        return array_map(function (PlaceCardData $place) use ($terms): SearchHit {
            $where = implode('، ', array_filter([$place->district?->name, $place->city->name]));
            $summary = $place->summary === null || trim($place->summary) === '' ? null : Str::limit(trim($place->summary), 150);

            return new SearchHit(
                type: $this->key(),
                typeLabel: $this->label(),
                title: $place->name,
                url: $this->urls->place($place->slug),
                snippet: implode(' · ', array_filter([$place->category->name, $where, $summary])),
                score: $terms->touches($place->name) ? 2 : 1,
            );
        }, $page->items);
    }
}
