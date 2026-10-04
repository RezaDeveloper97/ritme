<?php

declare(strict_types=1);

namespace App\Filament\Resources\Seo\StaticPageSeo\Pages;

use App\Filament\Resources\Seo\StaticPageSeo\StaticPageSeoResource;
use Filament\Resources\Pages\ListRecords;
use Filament\Tables\Table;

/**
 * The static-pages SEO list (see StaticPageSeoResource::table()).
 */
final class ListStaticPageSeoPages extends ListRecords
{
    protected static string $resource = StaticPageSeoResource::class;

    protected ?string $subheading = 'همه صفحه‌های ثابت قابل نمایه؛ عنوان و توضیح مؤثر (سفارشی یا پیش‌فرض) و وضعیت سئوی هر صفحه.';

    /**
     * Custom-data table (array records from the StaticPage registry): skip ListRecords' Eloquent query and
     * Model-typed record action / URL defaults; the resource's table() supplies records(), recordUrl() and actions.
     */
    protected function makeTable(): Table
    {
        return $this->makeBaseTable();
    }

    public function table(Table $table): Table
    {
        return StaticPageSeoResource::table($table);
    }

    protected function getHeaderActions(): array
    {
        return [];
    }
}
