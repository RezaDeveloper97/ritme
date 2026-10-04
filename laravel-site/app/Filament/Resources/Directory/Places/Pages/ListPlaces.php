<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Places\Pages;

use App\Filament\Resources\Directory\Places\PlaceResource;
use Filament\Actions\CreateAction;
use Filament\Resources\Pages\ListRecords;

final class ListPlaces extends ListRecords
{
    protected static string $resource = PlaceResource::class;

    protected function getHeaderActions(): array
    {
        return [CreateAction::make()->label('مجموعه جدید')];
    }
}
