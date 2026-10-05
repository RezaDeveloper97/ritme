<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Products\Pages;

use App\Domain\Shop\Catalog\Actions\SaveProduct;
use App\Domain\Shop\Catalog\Models\Product;
use App\Filament\Resources\Shop\Products\ProductFormData;
use App\Filament\Resources\Shop\Products\ProductResource;
use App\Filament\Resources\Shop\ShopAdmin;
use Filament\Resources\Pages\CreateRecord;
use Illuminate\Database\Eloquent\Model;

/**
 * A new product (shop managers + super-admins; SaveProduct, activity log). Save it as a draft until it is complete.
 */
final class CreateProduct extends CreateRecord
{
    protected static string $resource = ProductResource::class;

    protected function handleRecordCreation(array $data): Model
    {
        return app(SaveProduct::class)->handle(
            new Product,
            ProductFormData::attributes($data),
            ProductFormData::variants($data),
            ProductFormData::categoryIds($data),
            ProductFormData::primaryCategoryId($data),
            ProductFormData::galleryIds($data),
            ProductFormData::crossSellIds($data),
            ShopAdmin::user(),
        );
    }
}
