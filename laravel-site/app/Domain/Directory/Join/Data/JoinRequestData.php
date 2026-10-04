<?php

declare(strict_types=1);

namespace App\Domain\Directory\Join\Data;

use App\Domain\Directory\Join\Enums\AgeGroup;
use App\Domain\Directory\Join\Enums\BookingMode;
use App\Domain\Media\Data\MediaUpload;

/**
 * A validated join form (App\Http\Requests\JoinRequest::toData()) — the input of SubmitJoinRequest. `openingHours` is
 * already in the directory_places JSON shape; `photos` are local upload paths for the media pipeline.
 */
final readonly class JoinRequestData
{
    /**
     * @param  list<AgeGroup>  $ageGroups
     * @param  list<int>  $amenityIds
     * @param  array<string, list<array{opens: string, closes: string}>>  $openingHours
     * @param  list<MediaUpload>  $photos
     */
    public function __construct(
        public string $name,
        public int $categoryId,
        public int $cityId,
        public ?int $districtId,
        public string $contactName,
        public string $mobile,
        public ?string $email,
        public string $address,
        public ?string $phone,
        public ?float $latitude,
        public ?float $longitude,
        public ?string $about,
        public array $ageGroups,
        public array $amenityIds,
        public array $openingHours,
        public ?string $services,
        public BookingMode $bookingMode,
        public array $photos = [],
    ) {}
}
