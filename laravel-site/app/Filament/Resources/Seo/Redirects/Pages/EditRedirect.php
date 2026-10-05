<?php

declare(strict_types=1);

namespace App\Filament\Resources\Seo\Redirects\Pages;

use App\Domain\Seo\Redirects\Actions\DeleteRedirects;
use App\Domain\Seo\Redirects\Actions\SaveRedirect;
use App\Domain\Seo\Redirects\InvalidRedirect;
use App\Domain\Seo\Redirects\Models\Redirect;
use App\Filament\Resources\Seo\Redirects\RedirectForm;
use App\Filament\Resources\Seo\Redirects\RedirectResource;
use Filament\Actions\Action;
use Filament\Facades\Filament;
use Filament\Resources\Pages\EditRecord;
use Filament\Support\Icons\Heroicon;
use Illuminate\Database\Eloquent\Model;

final class EditRedirect extends EditRecord
{
    protected static string $resource = RedirectResource::class;

    protected function handleRecordUpdate(Model $record, array $data): Model
    {
        assert($record instanceof Redirect);

        try {
            return app(SaveRedirect::class)->handle($record, $data, Filament::auth()->user());
        } catch (InvalidRedirect $e) {
            throw RedirectForm::fail($e);
        }
    }

    protected function getHeaderActions(): array
    {
        return [
            Action::make('delete')->label('حذف')->icon(Heroicon::OutlinedTrash)->color('danger')->requiresConfirmation()
                ->authorize(fn (): bool => RedirectResource::allows('delete', $this->getRecord()))
                ->action(function (DeleteRedirects $delete): void {
                    $delete->handle([(int) $this->getRecord()->getKey()], Filament::auth()->user());
                    $this->redirect(RedirectResource::getUrl('index'));
                }),
        ];
    }

    protected function getRedirectUrl(): string
    {
        return RedirectResource::getUrl('index');
    }
}
