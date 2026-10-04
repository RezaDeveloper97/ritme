<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Amenities\Pages;

use App\Filament\Resources\Directory\Amenities\AmenityResource;
use Filament\Resources\Pages\CreateRecord;

final class CreateAmenity extends CreateRecord
{
    protected static string $resource = AmenityResource::class;
}
