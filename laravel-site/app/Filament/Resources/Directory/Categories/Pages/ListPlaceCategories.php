<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Categories\Pages;

use App\Filament\Resources\Directory\Categories\PlaceCategoryResource;
use Filament\Actions\CreateAction;
use Filament\Resources\Pages\ListRecords;

final class ListPlaceCategories extends ListRecords
{
    protected static string $resource = PlaceCategoryResource::class;

    protected function getHeaderActions(): array
    {
        return [CreateAction::make()->label('دسته جدید')];
    }
}
