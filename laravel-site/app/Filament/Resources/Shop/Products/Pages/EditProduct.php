<?php

declare(strict_types=1);

namespace App\Filament\Resources\Shop\Products\Pages;

use App\Domain\Shop\Catalog\Actions\SaveProduct;
use App\Domain\Shop\Catalog\Models\Product;
use App\Domain\Shop\Catalog\Support\ShopUrls;
use App\Filament\Resources\Shop\Products\ProductFormData;
use App\Filament\Resources\Shop\Products\ProductResource;
use App\Filament\Resources\Shop\ShopAdmin;
use Filament\Actions\Action;
use Filament\Actions\DeleteAction;
use Filament\Resources\Pages\EditRecord;
use Filament\Support\Icons\Heroicon;
use Illuminate\Database\Eloquent\Model;

/**
 * Editing a product. SEO managers can open this page but only their SEO tab is saved: content tabs are disabled in
 * the form and ignored here (ShopAdmin::canEditContent()), so a tampered Livewire payload changes nothing.
 *
 * @property Product $record
 */
final class EditProduct extends EditRecord
{
    protected static string $resource = ProductResource::class;

    protected function mutateFormDataBeforeFill(array $data): array
    {
        unset($data['price'], $data['compare_at_price']);

        return [...$data, ...ProductFormData::fill($this->record)];
    }

    /**
     * @param  array<string, mixed>  $data
     * @return array<string, mixed>
     */
    protected function mutateFormDataBeforeSave(array $data): array
    {
        return ShopAdmin::canEditContent() ? $data : [];
    }

    protected function handleRecordUpdate(Model $record, array $data): Model
    {
        /** @var Product $record */
        if (! ShopAdmin::canEditContent()) {
            return $record;
        }

        return app(SaveProduct::class)->handle(
            $record,
            ProductFormData::attributes($data),
            ProductFormData::variants($data),
            ProductFormData::categoryIds($data),
            ProductFormData::primaryCategoryId($data),
            ProductFormData::galleryIds($data),
            ProductFormData::crossSellIds($data),
            ShopAdmin::user(),
        );
    }

    protected function getHeaderActions(): array
    {
        return [
            Action::make('view')
                ->label('مشاهده در سایت')
                ->icon(Heroicon::OutlinedArrowTopRightOnSquare)
                ->color('gray')
                ->url(fn (): string => app(ShopUrls::class)->product($this->record->slug), shouldOpenInNewTab: true)
                ->visible(fn (): bool => $this->record->is_published),
            // Products that were ordered stay (unpublish them instead): order lines keep only a snapshot.
            DeleteAction::make()->disabled(fn (): bool => ProductResource::hasOrders($this->record)),
        ];
    }
}
