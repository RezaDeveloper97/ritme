<?php

declare(strict_types=1);

namespace App\Filament\Resources\Faq\Pages;

use App\Filament\Resources\Faq\FaqGroupResource;
use Filament\Actions\CreateAction;
use Filament\Resources\Pages\ListRecords;

final class ListFaqGroups extends ListRecords
{
    protected static string $resource = FaqGroupResource::class;

    protected function getHeaderActions(): array
    {
        return [CreateAction::make()->label('گروه جدید')];
    }
}
