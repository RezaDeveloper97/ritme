<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Brands\Pages;

use App\Filament\Resources\Shop\Brands\BrandResource;
use Filament\Actions\CreateAction;
use Filament\Resources\Pages\ListRecords;

final class ListBrands extends ListRecords
{
    protected static string $resource = BrandResource::class;

    protected function getHeaderActions(): array
    {
        return [CreateAction::make()->label('برند جدید')];
    }
}
