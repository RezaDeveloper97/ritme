<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Cities\Pages;

use App\Filament\Resources\Directory\Cities\CityResource;
use Filament\Actions\DeleteAction;
use Filament\Resources\Pages\EditRecord;

final class EditCity extends EditRecord
{
    protected static string $resource = CityResource::class;

    protected function getHeaderActions(): array
    {
        return [DeleteAction::make()->disabled(fn (): bool => CityResource::inUse($this->getRecord()))];
    }
}
