<?php

declare(strict_types=1);

namespace App\Domain\Directory\Data;

use App\Domain\Directory\Enums\PlaceSort;
use App\Domain\Search\Support\SearchTerms;
use Carbon\CarbonImmutable;
use Carbon\CarbonInterface;

/**
 * Filters of the directory listing (SearchPlaces). Controllers resolve slugs to ids through TaxonomyRepository.
 * Amenities are AND-ed. `openAt` keeps only places open at that moment (rounded down to 5 minutes so results can be
 * cached briefly). `text` is normalised like site search (SearchTerms). A reference point enables `Nearest` sorting
 * and card distances.
 */
final readonly class PlaceSearchCriteria
{
    public const OPEN_AT_BUCKET_MINUTES = 5;

    public const MAX_PER_PAGE = 60;

    /** @var list<int> */
    public array $amenityIds;

    public ?CarbonImmutable $openAt;

    public int $page;

    public int $perPage;

    /** @var list<string> */
    public array $tokens;

    /**
     * @param  list<int>  $amenityIds
     */
    public function __construct(
        public ?int $cityId = null,
        public ?int $districtId = null,
        public ?int $categoryId = null,
        public ?int $ageMonths = null,
        array $amenityIds = [],
        ?CarbonInterface $openAt = null,
        public ?string $text = null,
        public PlaceSort $sort = PlaceSort::Recommended,
        public ?float $latitude = null,
        public ?float $longitude = null,
        int $page = 1,
        int $perPage = 12,
    ) {
        $ids = array_values(array_unique(array_filter(array_map(intval(...), $amenityIds), static fn (int $id): bool => $id > 0)));
        sort($ids);
        $this->amenityIds = $ids;

        if ($openAt === null) {
            $this->openAt = null;
        } else {
            $local = CarbonImmutable::instance($openAt)->setTimezone(date_default_timezone_get())->startOfMinute();
            $this->openAt = $local->setTime($local->hour, $local->minute - $local->minute % self::OPEN_AT_BUCKET_MINUTES);
        }

        $this->page = max(1, $page);
        $this->perPage = max(1, min(self::MAX_PER_PAGE, $perPage));
        $this->tokens = SearchTerms::fromInput($text)->tokens;
    }

    public function hasReferencePoint(): bool
    {
        return $this->latitude !== null && $this->longitude !== null;
    }

    /**
     * Stable key of everything that changes the result.
     */
    public function cacheKey(): string
    {
        return md5(json_encode([
            $this->cityId, $this->districtId, $this->categoryId, $this->ageMonths, $this->amenityIds,
            $this->openAt?->format('Y-m-d H:i'), $this->tokens, $this->sort->value,
            $this->latitude === null ? null : round($this->latitude, 3),
            $this->longitude === null ? null : round($this->longitude, 3),
            $this->page, $this->perPage,
        ], JSON_THROW_ON_ERROR));
    }
}
