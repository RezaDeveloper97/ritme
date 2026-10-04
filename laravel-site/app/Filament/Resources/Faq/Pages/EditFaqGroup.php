<?php

declare(strict_types=1);

namespace App\Filament\Resources\Faq\Pages;

use App\Filament\Resources\Faq\FaqGroupResource;
use Filament\Actions\DeleteAction;
use Filament\Resources\Pages\EditRecord;

final class EditFaqGroup extends EditRecord
{
    protected static string $resource = FaqGroupResource::class;

    protected function getHeaderActions(): array
    {
        return [FaqGroupResource::viewOnSiteAction(), DeleteAction::make()];
    }
}
