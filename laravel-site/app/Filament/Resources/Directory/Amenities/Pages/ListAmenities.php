<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Amenities\Pages;

use App\Filament\Resources\Directory\Amenities\AmenityResource;
use Filament\Actions\CreateAction;
use Filament\Resources\Pages\ListRecords;

final class ListAmenities extends ListRecords
{
    protected static string $resource = AmenityResource::class;

    protected function getHeaderActions(): array
    {
        return [CreateAction::make()->label('امکان جدید')];
    }
}
