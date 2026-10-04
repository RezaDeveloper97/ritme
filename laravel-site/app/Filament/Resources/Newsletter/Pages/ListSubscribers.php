<?php

declare(strict_types=1);

namespace App\Filament\Resources\Newsletter\Pages;

use App\Filament\Resources\Newsletter\SubscriberResource;
use Filament\Resources\Pages\ListRecords;

final class ListSubscribers extends ListRecords
{
    protected static string $resource = SubscriberResource::class;

    protected function getHeaderActions(): array
    {
        return [SubscriberResource::exportAction()];
    }

    /**
     * The status filter value, passed on to the export link.
     */
    public function exportStatus(): ?string
    {
        $value = $this->tableFilters['status']['value'] ?? null;

        return is_string($value) && $value !== '' ? $value : null;
    }

    public function exportSearch(): ?string
    {
        $search = trim((string) $this->tableSearch);

        return $search !== '' ? $search : null;
    }
}
