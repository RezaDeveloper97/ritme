<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Landings\Pages;

use App\Filament\Resources\Directory\Landings\LandingResource;
use Filament\Actions\DeleteAction;
use Filament\Resources\Pages\EditRecord;

final class EditLanding extends EditRecord
{
    protected static string $resource = LandingResource::class;

    protected function getHeaderActions(): array
    {
        return [DeleteAction::make()->disabled(fn (): bool => LandingResource::inUse($this->getRecord()))];
    }
}
