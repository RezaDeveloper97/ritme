<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema\Data;

use App\Domain\Seo\Schema\Enums\LocalBusinessType;

/**
 * A place of the mother & child directory. `rating` only from real visitor reviews (see AggregateRatingData).
 */
final readonly class LocalBusinessData
{
    /**
     * @param  list<OpeningHoursData>  $openingHours
     * @param  list<string>  $imageUrls
     * @param  list<string>  $sameAs  the place's own site / social profiles
     */
    public function __construct(
        public string $url,
        public string $name,
        public PostalAddressData $address,
        public LocalBusinessType $type = LocalBusinessType::LocalBusiness,
        public ?string $description = null,
        public ?string $telephone = null,
        public array $imageUrls = [],
        public ?float $latitude = null,
        public ?float $longitude = null,
        public array $openingHours = [],
        public ?string $priceRange = null,
        public array $sameAs = [],
        public ?AggregateRatingData $rating = null,
    ) {}
}
