<?php

declare(strict_types=1);

namespace App\Filament\Forms\Components;

use App\Domain\Media\Exceptions\InvalidMediaException;
use App\Domain\Media\Models\Media;
use App\Filament\Resources\Media\MediaPresenter;
use App\Filament\Resources\Media\MediaUploads;
use Filament\Forms\Components\FileUpload;
use Filament\Forms\Components\Select;
use Filament\Forms\Components\TextInput;
use Filament\Notifications\Notification;
use Filament\Support\Exceptions\Halt;
use Illuminate\Http\UploadedFile;

/**
 * Reusable media field for every resource: stores a media id (`*_media_id` column or setting). Pick an existing item
 * from the library (searchable by alt, title or file name, with thumbnails) or upload a new one in a modal (alt text
 * required) — the upload goes through StoreMedia, so it is optimised like any library upload.
 *
 *     MediaPicker::make('featured_media_id'),
 *     MediaPicker::make('featured_mobile_media_id')->mobileOverride(),   // art direction for <x-picture>
 */
final class MediaPicker extends Select
{
    private const RESULTS_LIMIT = 30;

    protected function setUp(): void
    {
        parent::setUp();

        $this->label('تصویر');
        $this->placeholder('انتخاب از کتابخانه رسانه');
        $this->searchable();
        $this->allowHtml();
        $this->native(false);
        $this->searchPrompt('نام فایل یا متن جایگزین را جست‌وجو کنید');
        $this->noSearchResultsMessage('تصویری پیدا نشد.');
        $this->options(static fn (): array => self::optionsFor(Media::query()->latest('id')->limit(self::RESULTS_LIMIT)->get()));
        $this->getSearchResultsUsing(static fn (string $search): array => self::searchMedia($search));
        $this->getOptionLabelUsing(static function (mixed $value): ?string {
            $media = is_numeric($value) ? Media::query()->find((int) $value) : null;

            return $media === null ? null : self::optionLabel($media);
        });
        $this->rule('integer');
        $this->rule('exists:media,id');

        $this->createOptionForm([
            FileUpload::make('file')
                ->label('فایل تصویر')
                ->required()
                ->storeFiles(false)
                ->acceptedFileTypes(MediaUploads::acceptedTypes())
                ->maxSize(MediaUploads::maxKilobytes()),
            TextInput::make('alt')
                ->label('متن جایگزین (alt)')
                ->required()
                ->maxLength(255)
                ->helperText('توضیح کوتاه آنچه در تصویر دیده می‌شود؛ برای دسترس‌پذیری و سئو.'),
            TextInput::make('title')->label('عنوان')->maxLength(255),
        ]);
        $this->createOptionModalHeading('بارگذاری تصویر جدید');
        $this->createOptionUsing(static function (array $data): int {
            $file = $data['file'] ?? null;
            if (is_array($file)) {
                $file = reset($file);
            }
            if (! $file instanceof UploadedFile) {
                throw new Halt;
            }

            try {
                return MediaUploads::store($file, is_string($data['alt'] ?? null) ? $data['alt'] : null, is_string($data['title'] ?? null) ? $data['title'] : null)->id;
            } catch (InvalidMediaException $e) {
                Notification::make()->danger()->title('بارگذاری انجام نشد')->body($e->getMessage())->send();

                throw new Halt;
            }
        });
    }

    /**
     * Separate image for small screens (art direction): rendered by <x-picture> as the mobile <source>.
     */
    public function mobileOverride(bool $condition = true): static
    {
        if ($condition) {
            $this->label('تصویر موبایل (اختیاری)');
            $this->helperText('فقط اگر روی موبایل برش یا تصویر دیگری لازم است؛ خالی بماند همان تصویر اصلی نمایش داده می‌شود.');
            $this->placeholder('همان تصویر اصلی');
        }

        return $this;
    }

    /**
     * @return array<int, string>
     */
    private static function searchMedia(string $search): array
    {
        $term = '%'.addcslashes(trim($search), '%_\\').'%';

        return self::optionsFor(Media::query()
            ->where(static fn ($q) => $q->where('alt', 'like', $term)->orWhere('title', 'like', $term)->orWhere('original_name', 'like', $term))
            ->latest('id')
            ->limit(self::RESULTS_LIMIT)
            ->get());
    }

    /**
     * @param  iterable<Media>  $items
     * @return array<int, string>
     */
    private static function optionsFor(iterable $items): array
    {
        $options = [];
        foreach ($items as $media) {
            $options[$media->id] = self::optionLabel($media);
        }

        return $options;
    }

    private static function optionLabel(Media $media): string
    {
        $name = e(filled($media->alt) ? (string) $media->alt : (string) ($media->original_name ?? $media->filename));
        $meta = e(MediaPresenter::dimensions($media));
        $warning = blank($media->alt) ? ' <span class="text-warning-600 dark:text-warning-400">· بدون متن جایگزین</span>' : '';

        return '<span class="flex items-center gap-2">'
            .'<img src="'.e(MediaPresenter::thumbUrl($media)).'" alt="" width="40" height="40" class="size-10 shrink-0 rounded object-cover" loading="lazy">'
            .'<span>'.$name.' <span class="text-gray-500 dark:text-gray-400">· #'.$media->id.' · '.$meta.'</span>'.$warning.'</span>'
            .'</span>';
    }
}
