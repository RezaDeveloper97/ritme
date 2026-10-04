<?php

declare(strict_types=1);

namespace App\Domain\Directory\Booking\Data;

use App\Domain\Directory\Booking\Enums\TimeWindow;
use Carbon\CarbonImmutable;

/**
 * A validated booking form (BookingRequest form request → CreateBookingRequest). `mobile` is normalised
 * (`09xxxxxxxxx`), `date` is a Tehran calendar day, `serviceId` belongs to the place (checked by the form request).
 */
final readonly class BookingSubmission
{
    public function __construct(
        public ?int $serviceId,
        public CarbonImmutable $date,
        public TimeWindow $window,
        public string $parentName,
        public string $mobile,
        public ?int $childAgeMonths = null,
        public ?string $note = null,
    ) {}
}
