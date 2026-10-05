<?php

declare(strict_types=1);

namespace App\Filament\Widgets\Shop;

use App\Domain\Shop\Catalog\Models\Product;
use App\Filament\Auth\AdminAccess;
use App\Filament\Auth\AdminRole;
use App\Filament\Resources\Shop\LowStock;
use App\Filament\Resources\Shop\Products\ProductResource;
use Filament\Facades\Filament;
use Filament\Tables\Columns\IconColumn;
use Filament\Tables\Columns\TextColumn;
use Filament\Tables\Table;
use Filament\Widgets\TableWidget;
use Illuminate\Database\Eloquent\Builder;

/**
 * Low-stock list (L6-06): products (and their active variants) with at most `low_stock_threshold` units left (shop
 * settings, default 3), emptiest first, each linking to its edit page. Shop managers + super-admins only.
 */
final class LowStockProducts extends TableWidget
{
    protected static ?int $sort = 3;

    protected int|string|array $columnSpan = 'full';

    public static function canView(): bool
    {
        return AdminAccess::allows(Filament::auth()->user(), [AdminRole::ShopManager]);
    }

    public function table(Table $table): Table
    {
        $threshold = LowStock::threshold();

        return $table
            ->heading('موجودی کم (حداکثر '.fa_digits($threshold).' عدد)')
            ->query(static fn (): Builder => LowStock::products($threshold)->with('variants'))
            ->defaultSort('stock_qty')
            ->paginated([10])
            ->emptyStateHeading('موجودی همه محصولات کافی است.')
            ->recordUrl(static fn (Product $record): string => ProductResource::getUrl('edit', ['record' => $record]))
            ->columns([
                TextColumn::make('title')->label('محصول')->limit(50),
                TextColumn::make('stock_qty')->label('موجودی کل')
                    ->formatStateUsing(static fn (int $state): string => fa_digits($state))
                    ->color(static fn (int $state): string => $state === 0 ? 'danger' : 'warning'),
                TextColumn::make('low_variants')->label('تنوع‌های کم‌موجود')->placeholder('—')->wrap()
                    ->state(static fn (Product $record): ?string => LowStock::variantSummary($record, $threshold)),
                IconColumn::make('is_published')->label('منتشر')->boolean(),
            ]);
    }
}
