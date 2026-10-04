<?php

declare(strict_types=1);

namespace App\Domain\Directory\Booking\Actions;

use App\Domain\Contact\Enums\ContactTopic;
use App\Domain\Contact\Support\ContactRecipients;
use App\Domain\Contact\Support\ReplyChannel;
use App\Domain\Directory\Booking\Data\BookingData;
use App\Domain\Directory\Booking\Data\BookingSubmission;
use App\Domain\Directory\Booking\Models\BookingRequest;
use App\Domain\Directory\Booking\Support\BookingCode;
use App\Domain\Directory\Data\PlaceData;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Notifications\BookingRequestForPlace;
use App\Notifications\BookingRequestReceived;
use Illuminate\Contracts\Notifications\Dispatcher;
use Illuminate\Database\ConnectionInterface;
use Illuminate\Notifications\AnonymousNotifiable;
use Psr\Log\LoggerInterface;
use RuntimeException;
use Throwable;

/**
 * Stores a booking REQUEST (status new) for a published place in one transaction with a fresh unguessable code, then
 * queues (after commit) a data-minimal mail to the team mailbox (support email) and — when the place lists a mobile
 * among its phones and is not a demo place — an SMS to the place through the SmsSender contract. Notification
 * problems are logged, never shown: the request is already safe.
 */
final class CreateBookingRequest
{
    private const CODE_ATTEMPTS = 5;

    public function __construct(
        private readonly ConnectionInterface $db,
        private readonly SettingsRepository $settings,
        private readonly Dispatcher $notifications,
        private readonly LoggerInterface $log,
    ) {}

    public function handle(PlaceData $place, BookingSubmission $submission): BookingRequest
    {
        $service = BookingData::service($place, $submission->serviceId);

        /** @var BookingRequest $booking */
        $booking = $this->db->transaction(fn (): BookingRequest => BookingRequest::query()->create([
            'code' => $this->uniqueCode(),
            'place_id' => $place->id,
            'place_name' => $place->name,
            'service_id' => $service?->id,
            'service_name' => $service?->name,
            'service_price' => $service !== null && $service->price !== null && $service->price > 0 ? $service->price : null,
            'service_price_unit' => $service?->priceUnit,
            'preferred_date' => $submission->date->format('Y-m-d'),
            'time_window' => $submission->window,
            'parent_name' => $submission->parentName,
            'mobile' => $submission->mobile,
            'child_age_months' => $submission->childAgeMonths,
            'note' => $submission->note,
        ]));

        $this->notify($booking, $place);

        return $booking;
    }

    private function uniqueCode(): string
    {
        for ($i = 0; $i < self::CODE_ATTEMPTS; $i++) {
            $code = BookingCode::generate();
            if (! BookingRequest::query()->where('code', $code)->exists()) {
                return $code;
            }
        }

        throw new RuntimeException('Could not generate a unique booking code.');
    }

    private function notify(BookingRequest $booking, PlaceData $place): void
    {
        try {
            $team = ContactRecipients::for(ContactTopic::Support, $this->settings->all());
            if ($team === null) {
                $this->log->warning('Booking request stored without team notification: no valid support email in settings.', ['id' => $booking->id]);
            } else {
                $this->notifications->send((new AnonymousNotifiable)->route('mail', $team), new BookingRequestReceived($booking));
            }

            $mobile = $place->isDemo ? null : self::placeMobile($place);
            if ($mobile !== null) {
                $this->notifications->send((new AnonymousNotifiable)->route('sms', $mobile), new BookingRequestForPlace($booking));
            }
        } catch (Throwable $e) {
            report($e);
        }
    }

    /** The first of the place's phones that is a mobile number (landlines cannot receive the SMS). */
    private static function placeMobile(PlaceData $place): ?string
    {
        foreach ($place->phones as $phone) {
            $mobile = ReplyChannel::parse(str_contains($phone, '@') ? null : $phone)?->phone;
            if ($mobile !== null) {
                return $mobile;
            }
        }

        return null;
    }
}
