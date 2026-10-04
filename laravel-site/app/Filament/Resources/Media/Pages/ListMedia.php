<?php

declare(strict_types=1);

namespace App\Filament\Resources\Media\Pages;

use App\Domain\Media\Exceptions\InvalidMediaException;
use App\Domain\Media\Models\Media;
use App\Filament\Resources\Media\MediaResource;
use App\Filament\Resources\Media\MediaUploads;
use Filament\Actions\Action;
use Filament\Facades\Filament;
use Filament\Forms\Components\FileUpload;
use Filament\Forms\Components\TextInput;
use Filament\Notifications\Notification;
use Filament\Resources\Pages\ListRecords;
use Filament\Support\Icons\Heroicon;
use Illuminate\Http\UploadedFile;

final class ListMedia extends ListRecords
{
    protected static string $resource = MediaResource::class;

    protected function getHeaderActions(): array
    {
        return [
            Action::make('upload')
                ->label('بارگذاری تصویر')
                ->icon(Heroicon::OutlinedArrowUpTray)
                ->authorize(static fn (): bool => Filament::auth()->user()?->can('create', Media::class) ?? false)
                ->modalHeading('بارگذاری تصویر')
                ->modalSubmitActionLabel('بارگذاری')
                ->schema([
                    FileUpload::make('files')
                        ->label('فایل‌ها')
                        ->multiple()
                        ->required()
                        ->storeFiles(false)
                        ->acceptedFileTypes(MediaUploads::acceptedTypes())
                        ->maxSize(MediaUploads::maxKilobytes())
                        ->maxFiles(20)
                        ->helperText('JPG، PNG، WebP، GIF، AVIF یا SVG. هر فایل خودکار بهینه و در اندازه‌های موبایل و دسکتاپ ساخته می‌شود.'),
                    TextInput::make('alt')
                        ->label('متن جایگزین (اختیاری، برای همه فایل‌ها)')
                        ->maxLength(255)
                        ->helperText('بعداً برای هر تصویر جداگانه قابل ویرایش است؛ تصاویر بدون متن جایگزین با هشدار مشخص می‌شوند.'),
                ])
                ->action(function (array $data): void {
                    $alt = is_string($data['alt'] ?? null) ? $data['alt'] : null;
                    $stored = 0;
                    $errors = [];

                    foreach ((array) ($data['files'] ?? []) as $file) {
                        if (! $file instanceof UploadedFile) {
                            continue;
                        }
                        try {
                            MediaUploads::store($file, $alt);
                            $stored++;
                        } catch (InvalidMediaException $e) {
                            $errors[] = $file->getClientOriginalName().': '.$e->getMessage();
                        }
                    }

                    if ($stored > 0) {
                        Notification::make()->success()->title(sprintf('%d تصویر بارگذاری شد.', $stored))->send();
                    }
                    if ($errors !== []) {
                        Notification::make()->danger()->title('بعضی فایل‌ها بارگذاری نشدند')->body(implode("\n", $errors))->persistent()->send();
                    }
                }),
        ];
    }
}
