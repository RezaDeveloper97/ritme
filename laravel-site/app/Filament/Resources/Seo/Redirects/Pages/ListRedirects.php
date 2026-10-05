<?php

declare(strict_types=1);

namespace App\Filament\Resources\Seo\Redirects\Pages;

use App\Filament\Resources\Seo\Redirects\RedirectResource;
use Filament\Actions\CreateAction;
use Filament\Resources\Pages\ListRecords;

final class ListRedirects extends ListRecords
{
    protected static string $resource = RedirectResource::class;

    protected function getHeaderActions(): array
    {
        return [
            CreateAction::make()->label('ریدایرکت جدید'),
            RedirectResource::importAction(),
            RedirectResource::exportAction(),
        ];
    }
}
