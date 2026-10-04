<?php

declare(strict_types=1);

namespace App\Filament\Resources\Faq\Pages;

use App\Filament\Resources\Faq\FaqGroupResource;
use Filament\Resources\Pages\CreateRecord;

final class CreateFaqGroup extends CreateRecord
{
    protected static string $resource = FaqGroupResource::class;

    protected function getRedirectUrl(): string
    {
        return self::getResource()::getUrl('edit', ['record' => $this->getRecord()]);
    }
}
