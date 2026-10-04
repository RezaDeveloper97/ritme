<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Categories\Pages;

use App\Filament\Resources\Directory\Categories\PlaceCategoryResource;
use Filament\Actions\DeleteAction;
use Filament\Resources\Pages\EditRecord;

final class EditPlaceCategory extends EditRecord
{
    protected static string $resource = PlaceCategoryResource::class;

    protected function getHeaderActions(): array
    {
        return [DeleteAction::make()->disabled(fn (): bool => PlaceCategoryResource::inUse($this->getRecord()))];
    }
}
