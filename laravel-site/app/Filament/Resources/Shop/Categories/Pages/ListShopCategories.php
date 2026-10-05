<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Categories\Pages;

use App\Filament\Resources\Shop\Categories\ShopCategoryResource;
use Filament\Actions\CreateAction;
use Filament\Resources\Pages\ListRecords;

final class ListShopCategories extends ListRecords
{
    protected static string $resource = ShopCategoryResource::class;

    protected function getHeaderActions(): array
    {
        return [CreateAction::make()->label('دسته جدید')];
    }
}
