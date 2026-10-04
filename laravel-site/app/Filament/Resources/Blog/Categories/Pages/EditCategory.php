<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Categories\Pages;

use App\Filament\Resources\Blog\Categories\CategoryResource;
use App\Filament\Resources\Blog\SeoOnlyForSeoManagers;
use Filament\Actions\DeleteAction;
use Filament\Resources\Pages\EditRecord;

final class EditCategory extends EditRecord
{
    use SeoOnlyForSeoManagers;

    protected static string $resource = CategoryResource::class;

    protected function getHeaderActions(): array
    {
        return [DeleteAction::make()];
    }
}
