<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Bookings\Pages;

use App\Domain\Directory\Booking\Enums\BookingStatus;
use App\Domain\Directory\Booking\Models\BookingRequest;
use App\Filament\Resources\Directory\Bookings\BookingRequestResource;
use Filament\Resources\Pages\ListRecords;
use Filament\Schemas\Components\Tabs\Tab;
use Illuminate\Database\Eloquent\Builder;

/**
 * The board: one tab per status (with its count) in flow order, plus «همه».
 */
final class ListBookingRequests extends ListRecords
{
    protected static string $resource = BookingRequestResource::class;

    protected function getHeaderActions(): array
    {
        return [BookingRequestResource::exportAction()];
    }

    /**
     * @return array<string, Tab>
     */
    public function getTabs(): array
    {
        $counts = BookingRequest::query()->toBase()->selectRaw('status, COUNT(*) as aggregate')->groupBy('status')->pluck('aggregate', 'status');

        $tabs = [];
        foreach ([BookingStatus::New, BookingStatus::Confirmed, BookingStatus::Done, BookingStatus::Cancelled] as $status) {
            $tabs[$status->value] = Tab::make($status->label())
                ->badge(fa_digits((int) ($counts[$status->value] ?? 0)))
                ->badgeColor($status->color())
                ->modifyQueryUsing(static fn (Builder $query): Builder => $query->where('status', $status->value));
        }
        $tabs['all'] = Tab::make('همه');

        return $tabs;
    }

    public function getDefaultActiveTab(): string
    {
        return BookingStatus::New->value;
    }

    /**
     * The current tab, place filter, preferred-day range and search, passed on to the export link.
     *
     * @return array<string, string>
     */
    public function exportParameters(): array
    {
        $status = BookingStatus::tryFrom((string) $this->activeTab);
        $place = $this->tableFilters['place_id']['value'] ?? null;
        $from = $this->tableFilters['preferred']['from'] ?? null;
        $until = $this->tableFilters['preferred']['until'] ?? null;
        $search = trim((string) $this->tableSearch);

        return array_filter([
            'status' => $status?->value,
            'place' => is_numeric($place) ? (string) $place : null,
            'from' => is_string($from) && $from !== '' ? $from : null,
            'until' => is_string($until) && $until !== '' ? $until : null,
            'search' => $search !== '' ? $search : null,
        ], static fn (?string $value): bool => $value !== null);
    }
}
