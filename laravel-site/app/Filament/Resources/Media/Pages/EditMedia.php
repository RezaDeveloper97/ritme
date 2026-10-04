<?php

declare(strict_types=1);

namespace App\Filament\Resources\Media\Pages;

use App\Domain\Media\Actions\DeleteUnusedMedia;
use App\Domain\Media\Actions\FindMediaUsages;
use App\Domain\Media\Actions\RegenerateMediaVariants;
use App\Domain\Media\Actions\UpdateMediaDetails;
use App\Domain\Media\Models\Media;
use App\Filament\Resources\Media\MediaResource;
use Filament\Actions\Action;
use Filament\Facades\Filament;
use Filament\Notifications\Notification;
use Filament\Resources\Pages\EditRecord;
use Filament\Support\Icons\Heroicon;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\HtmlString;
use Throwable;

/**
 * @property Media $record
 */
final class EditMedia extends EditRecord
{
    protected static string $resource = MediaResource::class;

    protected function mutateFormDataBeforeFill(array $data): array
    {
        $data['focal'] = ['x' => $this->record->focal_x, 'y' => $this->record->focal_y];

        return $data;
    }

    protected function handleRecordUpdate(Model $record, array $data): Model
    {
        /** @var Media $record */
        $focal = is_array($data['focal'] ?? null) ? $data['focal'] : [];
        $before = $record->only(['alt', 'title', 'caption', 'focal_x', 'focal_y']);

        $record = app(UpdateMediaDetails::class)->handle(
            $record,
            self::string($data['alt'] ?? null),
            self::string($data['title'] ?? null),
            self::string($data['caption'] ?? null),
            is_numeric($focal['x'] ?? null) ? (float) $focal['x'] : null,
            is_numeric($focal['y'] ?? null) ? (float) $focal['y'] : null,
        );

        $after = $record->only(['alt', 'title', 'caption', 'focal_x', 'focal_y']);
        if ($after !== $before) {
            activity('media')
                ->causedBy(Filament::auth()->user())
                ->performedOn($record)
                ->event('updated')
                ->withProperties(['old' => $before, 'attributes' => $after])
                ->log('media.updated');
        }

        return $record->refresh();
    }

    protected function getHeaderActions(): array
    {
        return [
            Action::make('regenerate')
                ->label('ساخت دوباره نسخه‌ها')
                ->icon(Heroicon::OutlinedArrowPath)
                ->color('gray')
                ->requiresConfirmation()
                ->modalDescription('همه نسخه‌های موبایل، دسکتاپ، بندانگشتی و اشتراک‌گذاری از روی فایل اصلی دوباره ساخته می‌شوند.')
                ->authorize(fn (): bool => Filament::auth()->user()?->can('update', $this->record) ?? false)
                ->action(function (RegenerateMediaVariants $regenerate): void {
                    try {
                        $regenerate->handle($this->record);
                    } catch (Throwable $e) {
                        report($e);
                        Notification::make()->danger()->title('ساخت نسخه‌ها انجام نشد')->body($e->getMessage())->send();

                        return;
                    }

                    $this->record->refresh();
                    Notification::make()->success()
                        ->title($this->record->optimized_at === null ? 'ساخت نسخه‌ها در صف قرار گرفت.' : 'نسخه‌ها دوباره ساخته شدند.')
                        ->send();
                }),
            Action::make('usages')
                ->label('محل‌های استفاده')
                ->icon(Heroicon::OutlinedMagnifyingGlass)
                ->color('gray')
                ->modalHeading('محل‌های استفاده از این تصویر')
                ->modalSubmitAction(false)
                ->modalCancelActionLabel('بستن')
                ->modalContent(function (FindMediaUsages $usages): HtmlString {
                    $list = $usages->handle([$this->record->id])[$this->record->id] ?? [];
                    if ($list === []) {
                        return new HtmlString('<p class="text-sm text-gray-500 dark:text-gray-400">این تصویر جایی استفاده نشده است.</p>');
                    }

                    return new HtmlString('<ul class="list-disc space-y-1 ps-5 text-sm">'
                        .implode('', array_map(static fn (string $line): string => '<li>'.e($line).'</li>', $list))
                        .'</ul>');
                }),
            Action::make('delete')
                ->label('حذف')
                ->icon(Heroicon::OutlinedTrash)
                ->color('danger')
                ->requiresConfirmation()
                ->modalDescription('تصویر و همه نسخه‌هایش حذف می‌شوند. تصویری که در سایت استفاده شده حذف نمی‌شود.')
                ->authorize(fn (): bool => Filament::auth()->user()?->can('delete', $this->record) ?? false)
                ->action(function (DeleteUnusedMedia $delete): void {
                    $result = $delete->handle([$this->record->id]);
                    if ($result['deleted'] === []) {
                        Notification::make()->warning()->title('این تصویر در سایت استفاده شده و حذف نشد.')->send();

                        return;
                    }

                    MediaResource::logDeleted($result['deleted']);
                    Notification::make()->success()->title('تصویر حذف شد.')->send();
                    $this->redirect(MediaResource::getUrl('index'));
                }),
        ];
    }

    private static function string(mixed $value): ?string
    {
        return is_string($value) ? $value : null;
    }
}
