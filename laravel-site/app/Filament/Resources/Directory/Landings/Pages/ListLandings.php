<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Landings\Pages;

use App\Filament\Resources\Directory\Landings\LandingResource;
use Filament\Actions\CreateAction;
use Filament\Resources\Pages\ListRecords;

final class ListLandings extends ListRecords
{
    protected static string $resource = LandingResource::class;

    protected function getHeaderActions(): array
    {
        return [CreateAction::make()->label('متن صفحه جدید')];
    }
}
