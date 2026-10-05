<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Categories\Pages;

use App\Domain\Shop\Catalog\Models\Category;
use App\Filament\Resources\Shop\Categories\ShopCategoryResource;
use App\Filament\Resources\Shop\ShopAdmin;
use Filament\Actions\DeleteAction;
use Filament\Resources\Pages\EditRecord;

/**
 * Editing a shop category. SEO managers only save the SEO tab (content ignored here, whatever the payload holds).
 *
 * @property Category $record
 */
final class EditShopCategory extends EditRecord
{
    protected static string $resource = ShopCategoryResource::class;

    /**
     * @param  array<string, mixed>  $data
     * @return array<string, mixed>
     */
    protected function mutateFormDataBeforeSave(array $data): array
    {
        return ShopAdmin::canEditContent() ? $data : [];
    }

    protected function getHeaderActions(): array
    {
        return [DeleteAction::make()->disabled(fn (): bool => ShopCategoryResource::inUse($this->record))];
    }
}
