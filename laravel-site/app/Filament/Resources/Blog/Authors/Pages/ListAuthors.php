<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Authors\Pages;

use App\Filament\Resources\Blog\Authors\AuthorResource;
use Filament\Actions\CreateAction;
use Filament\Resources\Pages\ListRecords;

final class ListAuthors extends ListRecords
{
    protected static string $resource = AuthorResource::class;

    protected function getHeaderActions(): array
    {
        return [CreateAction::make()->label('نویسنده جدید')];
    }
}
