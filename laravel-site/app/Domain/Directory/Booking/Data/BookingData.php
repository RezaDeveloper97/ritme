<?php

declare(strict_types=1);

namespace App\Domain\Directory\Booking\Data;

use App\Domain\Directory\Booking\Enums\BookingStatus;
use App\Domain\Directory\Booking\Enums\TimeWindow;
use App\Domain\Directory\Booking\Models\BookingRequest;
use App\Domain\Directory\Booking\Support\MobileMask;
use App\Domain\Directory\Data\PlaceData;
use App\Domain\Directory\Data\PlaceServiceData;
use Carbon\CarbonImmutable;

/**
 * A booking request for its /directory/booked/{code} page. The mobile is masked here already; the note is not
 * carried (it is for the place, not for whoever holds the link). `placeSlug` is set while the place is published.
 */
final readonly class BookingData
{
    public function __construct(
        public string $code,
        public BookingStatus $status,
        public ?string $placeSlug,
        public string $placeName,
        public ?string $serviceName,
        public ?int $servicePrice,
        public ?string $servicePriceUnit,
        public CarbonImmutable $preferredDate,
        public TimeWindow $timeWindow,
        public string $parentName,
        public string $maskedMobile,
        public ?int $childAgeMonths,
        public CarbonImmutable $createdAt,
    ) {}

    /** Expects `place` loaded (its slug is used only while the place is published). */
    public static function fromModel(BookingRequest $booking, ?string $publishedSlug): self
    {
        return new self(
            code: $booking->code,
            status: $booking->status,
            placeSlug: $publishedSlug,
            placeName: $booking->place_name,
            serviceName: $booking->service_name,
            servicePrice: $booking->service_price,
            servicePriceUnit: $booking->service_price_unit,
            preferredDate: CarbonImmutable::parse($booking->preferred_date->format('Y-m-d'), 'Asia/Tehran'),
            timeWindow: $booking->time_window,
            parentName: $booking->parent_name,
            maskedMobile: MobileMask::mask($booking->mobile),
            childAgeMonths: $booking->child_age_months,
            createdAt: CarbonImmutable::instance($booking->created_at ?? now()),
        );
    }

    /**
     * What a stored request would look like — the answer to a spam post, which stores nothing but must look the same.
     */
    public static function fromSubmission(string $code, PlaceData $place, BookingSubmission $submission): self
    {
        $service = self::service($place, $submission->serviceId);

        return new self(
            code: $code,
            status: BookingStatus::New,
            placeSlug: $place->slug,
            placeName: $place->name,
            serviceName: $service?->name,
            servicePrice: $service?->price,
            servicePriceUnit: $service?->priceUnit,
            preferredDate: $submission->date,
            timeWindow: $submission->window,
            parentName: $submission->parentName,
            maskedMobile: MobileMask::mask($submission->mobile),
            childAgeMonths: $submission->childAgeMonths,
            createdAt: CarbonImmutable::now(),
        );
    }

    public static function service(PlaceData $place, ?int $serviceId): ?PlaceServiceData
    {
        foreach ($place->services as $service) {
            if ($service->id === $serviceId) {
                return $service;
            }
        }

        return null;
    }
}
