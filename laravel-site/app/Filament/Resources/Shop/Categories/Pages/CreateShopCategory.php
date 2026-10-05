<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Categories\Pages;

use App\Filament\Resources\Shop\Categories\ShopCategoryResource;
use Filament\Resources\Pages\CreateRecord;

final class CreateShopCategory extends CreateRecord
{
    protected static string $resource = ShopCategoryResource::class;
}
