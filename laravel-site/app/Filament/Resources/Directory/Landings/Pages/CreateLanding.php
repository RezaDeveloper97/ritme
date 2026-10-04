<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Landings\Pages;

use App\Filament\Resources\Directory\Landings\LandingResource;
use Filament\Resources\Pages\CreateRecord;

final class CreateLanding extends CreateRecord
{
    protected static string $resource = LandingResource::class;
}
