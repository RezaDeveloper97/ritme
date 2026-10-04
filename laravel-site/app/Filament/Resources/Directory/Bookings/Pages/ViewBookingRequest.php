<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Bookings\Pages;

use App\Domain\Directory\Booking\Models\BookingRequest;
use App\Filament\Resources\Directory\Bookings\BookingRequestResource;
use Filament\Actions\Action;
use Filament\Resources\Pages\ViewRecord;
use Filament\Support\Icons\Heroicon;

/**
 * One booking request with the parent's full number (call back to confirm the time) and the status flow actions.
 */
final class ViewBookingRequest extends ViewRecord
{
    protected static string $resource = BookingRequestResource::class;

    protected function getHeaderActions(): array
    {
        $record = $this->getRecord();

        return [
            Action::make('call')
                ->label('تماس با والد')
                ->icon(Heroicon::OutlinedPhone)
                ->color('gray')
                ->url($record instanceof BookingRequest ? 'tel:'.$record->mobile : null),
            ...BookingRequestResource::statusActions(),
        ];
    }
}
