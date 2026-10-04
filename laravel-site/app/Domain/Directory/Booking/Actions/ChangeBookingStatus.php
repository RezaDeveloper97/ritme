<?php

declare(strict_types=1);

namespace App\Domain\Directory\Booking\Actions;

use App\Domain\Directory\Booking\Enums\BookingStatus;
use App\Domain\Directory\Booking\Models\BookingRequest;
use App\Domain\Directory\Support\DirectoryActivity;
use Illuminate\Database\Eloquent\Model;
use InvalidArgumentException;

/**
 * Moves a booking request along its flow (admin board, L5-06):
 *
 *   new → confirmed | cancelled,  confirmed → done | cancelled,  cancelled / done → (final)
 *
 * The booked page reads the status, so the parent sees the change. Every change is written to the activity log
 * (`directory.booking.status`, no personal data in the properties).
 */
final class ChangeBookingStatus
{
    private const FLOW = [
        'new' => [BookingStatus::Confirmed, BookingStatus::Cancelled],
        'confirmed' => [BookingStatus::Done, BookingStatus::Cancelled],
        'cancelled' => [],
        'done' => [],
    ];

    /**
     * @return list<BookingStatus>
     */
    public static function next(BookingStatus $from): array
    {
        return self::FLOW[$from->value];
    }

    public static function allowed(BookingStatus $from, BookingStatus $to): bool
    {
        return in_array($to, self::next($from), true);
    }

    /**
     * @throws InvalidArgumentException when the flow does not allow the change
     */
    public function handle(BookingRequest $booking, BookingStatus $status, ?Model $causer = null): void
    {
        $from = $booking->status;
        if (! self::allowed($from, $status)) {
            throw new InvalidArgumentException("Booking status cannot change from {$from->value} to {$status->value}.");
        }

        $booking->status = $status;
        $booking->save();

        DirectoryActivity::status($booking, 'directory.booking.status', $from->value, $status->value, $causer);
    }
}
