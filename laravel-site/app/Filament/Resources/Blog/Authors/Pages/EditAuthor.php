<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Authors\Pages;

use App\Filament\Resources\Blog\Authors\AuthorResource;
use App\Filament\Resources\Blog\SeoOnlyForSeoManagers;
use Filament\Actions\DeleteAction;
use Filament\Resources\Pages\EditRecord;

final class EditAuthor extends EditRecord
{
    use SeoOnlyForSeoManagers;

    protected static string $resource = AuthorResource::class;

    protected function getHeaderActions(): array
    {
        return [DeleteAction::make()];
    }
}
