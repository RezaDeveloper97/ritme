<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\Places\Pages;

use App\Domain\Directory\Actions\SavePlace;
use App\Domain\Directory\Enums\PlaceStatus;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Support\DirectoryUrls;
use App\Filament\Resources\Directory\DirectoryAdmin;
use App\Filament\Resources\Directory\Places\PlaceFormData;
use App\Filament\Resources\Directory\Places\PlaceResource;
use Filament\Actions\Action;
use Filament\Actions\DeleteAction;
use Filament\Resources\Pages\EditRecord;
use Filament\Support\Icons\Heroicon;
use Illuminate\Database\Eloquent\Model;

/**
 * Editing a place; header actions publish / move back to draft / suspend it (ChangePlaceStatus, activity log) and
 * open the public page of a published place.
 *
 * @property Place $record
 */
final class EditPlace extends EditRecord
{
    protected static string $resource = PlaceResource::class;

    protected function mutateFormDataBeforeFill(array $data): array
    {
        return [...$data, ...PlaceFormData::fill($this->record)];
    }

    protected function handleRecordUpdate(Model $record, array $data): Model
    {
        /** @var Place $record */
        return app(SavePlace::class)->handle(
            $record,
            PlaceFormData::attributes($data),
            PlaceFormData::amenityIds($data),
            PlaceFormData::galleryIds($data),
            DirectoryAdmin::user(),
        );
    }

    protected function getHeaderActions(): array
    {
        return [
            Action::make('view')
                ->label('مشاهده در سایت')
                ->icon(Heroicon::OutlinedArrowTopRightOnSquare)
                ->color('gray')
                ->url(fn (): string => app(DirectoryUrls::class)->place($this->record->slug), shouldOpenInNewTab: true)
                ->visible(fn (): bool => $this->record->status === PlaceStatus::Published),
            PlaceResource::statusAction('publish', PlaceStatus::Published, 'انتشار', Heroicon::OutlinedEye),
            PlaceResource::statusAction('draft', PlaceStatus::Draft, 'بازگشت به پیش‌نویس', Heroicon::OutlinedEyeSlash),
            PlaceResource::statusAction('suspend', PlaceStatus::Suspended, 'تعلیق', Heroicon::OutlinedNoSymbol),
            DeleteAction::make(),
        ];
    }
}
