<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Brands\Pages;

use App\Domain\Shop\Catalog\Models\Brand;
use App\Filament\Resources\Shop\Brands\BrandResource;
use Filament\Actions\DeleteAction;
use Filament\Resources\Pages\EditRecord;

/**
 * @property Brand $record
 */
final class EditBrand extends EditRecord
{
    protected static string $resource = BrandResource::class;

    protected function getHeaderActions(): array
    {
        return [DeleteAction::make()->disabled(fn (): bool => $this->record->products()->exists())];
    }
}
