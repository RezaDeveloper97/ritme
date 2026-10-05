<?php

declare(strict_types=1);

namespace App\Filament\Resources\Seo\NotFoundLogs\Pages;

use App\Filament\Resources\Seo\NotFoundLogs\NotFoundLogResource;
use Filament\Resources\Pages\ListRecords;

final class ListNotFoundLogs extends ListRecords
{
    protected static string $resource = NotFoundLogResource::class;
}
