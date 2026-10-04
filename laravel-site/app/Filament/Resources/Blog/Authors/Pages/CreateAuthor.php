<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Authors\Pages;

use App\Filament\Resources\Blog\Authors\AuthorResource;
use Filament\Resources\Pages\CreateRecord;

final class CreateAuthor extends CreateRecord
{
    protected static string $resource = AuthorResource::class;
}
