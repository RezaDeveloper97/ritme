<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Categories\Pages;

use App\Filament\Resources\Directory\Categories\PlaceCategoryResource;
use Filament\Resources\Pages\CreateRecord;

final class CreatePlaceCategory extends CreateRecord
{
    protected static string $resource = PlaceCategoryResource::class;
}
