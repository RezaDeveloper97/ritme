<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Amenities\Pages;

use App\Filament\Resources\Directory\Amenities\AmenityResource;
use Filament\Actions\DeleteAction;
use Filament\Resources\Pages\EditRecord;

final class EditAmenity extends EditRecord
{
    protected static string $resource = AmenityResource::class;

    protected function getHeaderActions(): array
    {
        return [DeleteAction::make()->disabled(fn (): bool => AmenityResource::inUse($this->getRecord()))];
    }
}
