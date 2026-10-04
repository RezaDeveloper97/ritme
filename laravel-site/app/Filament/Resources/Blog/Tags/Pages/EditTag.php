<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Tags\Pages;

use App\Filament\Resources\Blog\SeoOnlyForSeoManagers;
use App\Filament\Resources\Blog\Tags\TagResource;
use Filament\Actions\DeleteAction;
use Filament\Resources\Pages\EditRecord;

final class EditTag extends EditRecord
{
    use SeoOnlyForSeoManagers;

    protected static string $resource = TagResource::class;

    protected function getHeaderActions(): array
    {
        return [DeleteAction::make()];
    }
}
