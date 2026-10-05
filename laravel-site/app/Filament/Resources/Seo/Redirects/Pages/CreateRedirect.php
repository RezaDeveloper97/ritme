<?php

declare(strict_types=1);

namespace App\Filament\Resources\Seo\Redirects\Pages;

use App\Domain\Seo\Redirects\Actions\SaveRedirect;
use App\Domain\Seo\Redirects\InvalidRedirect;
use App\Domain\Seo\Redirects\Models\Redirect;
use App\Filament\Resources\Seo\Redirects\RedirectForm;
use App\Filament\Resources\Seo\Redirects\RedirectResource;
use Filament\Facades\Filament;
use Filament\Resources\Pages\CreateRecord;
use Illuminate\Database\Eloquent\Model;

final class CreateRedirect extends CreateRecord
{
    protected static string $resource = RedirectResource::class;

    protected function handleRecordCreation(array $data): Model
    {
        try {
            return app(SaveRedirect::class)->handle(new Redirect, $data, Filament::auth()->user());
        } catch (InvalidRedirect $e) {
            throw RedirectForm::fail($e);
        }
    }

    protected function getRedirectUrl(): string
    {
        return RedirectResource::getUrl('index');
    }
}
