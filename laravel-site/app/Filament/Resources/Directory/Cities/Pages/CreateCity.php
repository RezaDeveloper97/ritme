<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Cities\Pages;

use App\Filament\Resources\Directory\Cities\CityResource;
use Filament\Resources\Pages\CreateRecord;

final class CreateCity extends CreateRecord
{
    protected static string $resource = CityResource::class;
}
