<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Categories\Pages;

use App\Filament\Resources\Blog\Categories\CategoryResource;
use Filament\Actions\CreateAction;
use Filament\Resources\Pages\ListRecords;

final class ListCategories extends ListRecords
{
    protected static string $resource = CategoryResource::class;

    protected function getHeaderActions(): array
    {
        return [CreateAction::make()->label('دسته جدید')];
    }
}
