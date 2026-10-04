<?php

declare(strict_types=1);

namespace App\Filament\Resources\Blog\Tags\Pages;

use App\Filament\Resources\Blog\Tags\TagResource;
use Filament\Resources\Pages\CreateRecord;

final class CreateTag extends CreateRecord
{
    protected static string $resource = TagResource::class;
}
