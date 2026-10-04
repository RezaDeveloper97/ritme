<?php

declare(strict_types=1);

namespace App\Domain\Directory\Booking\Queries;

use App\Domain\Directory\Booking\Data\BookingData;
use App\Domain\Directory\Booking\Models\BookingRequest;
use App\Domain\Directory\Booking\Support\BookingCode;
use App\Domain\Directory\Enums\PlaceStatus;

/**
 * The booking request behind a /directory/booked/{code} URL (case-insensitive), or null. Not cached: the page is
 * `no-store` and rarely visited, and its status changes from the admin.
 */
final class FindBookingByCode
{
    public function handle(string $code): ?BookingData
    {
        $code = BookingCode::normalize($code);
        if (preg_match('/^[A-Z0-9-]{1,20}$/', $code) !== 1) {
            return null;
        }

        $booking = BookingRequest::query()->with('place:id,slug,status')->where('code', $code)->first();
        if ($booking === null) {
            return null;
        }

        $place = $booking->place;
        $slug = $place !== null && $place->status === PlaceStatus::Published ? $place->slug : null;

        return BookingData::fromModel($booking, $slug);
    }
}
