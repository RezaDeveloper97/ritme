<?php

declare(strict_types=1);

namespace App\Filament\Resources\Media;

use App\Domain\Media\Actions\StoreMedia;
use App\Domain\Media\Actions\UpdateMediaDetails;
use App\Domain\Media\Data\MediaUpload;
use App\Domain\Media\Models\Media;
use Filament\Facades\Filament;
use Illuminate\Http\UploadedFile;

/**
 * Delivery-side adapter: Livewire/HTTP upload → MediaUpload → StoreMedia. A re-upload of an existing file (sha256
 * dedupe) returns the existing item; it receives the new alt/title only when it had none.
 */
final class MediaUploads
{
    public static function maxKilobytes(): int
    {
        return intdiv((int) config('media.max_bytes'), 1024);
    }

    /**
     * @return list<string>
     */
    public static function acceptedTypes(): array
    {
        /** @var list<string> $mimes */
        $mimes = (array) config('media.mimes', []);
        if ((bool) config('media.svg', true)) {
            $mimes[] = 'image/svg+xml';
        }

        return $mimes;
    }

    public static function store(UploadedFile $file, ?string $alt = null, ?string $title = null): Media
    {
        $user = Filament::auth()->id();

        $media = app(StoreMedia::class)->handle(new MediaUpload(
            path: (string) $file->getRealPath(),
            originalName: $file->getClientOriginalName(),
            alt: filled($alt) ? trim((string) $alt) : null,
            title: filled($title) ? trim((string) $title) : null,
            uploadedBy: is_numeric($user) ? (int) $user : null,
        ));

        if (! $media->wasRecentlyCreated && (blank($media->alt) && filled($alt) || blank($media->title) && filled($title))) {
            $media = app(UpdateMediaDetails::class)->handle(
                $media,
                filled($media->alt) ? $media->alt : $alt,
                filled($media->title) ? $media->title : $title,
                $media->caption,
            );
        }

        activity('media')
            ->causedBy(Filament::auth()->user())
            ->performedOn($media)
            ->event('created')
            ->log('media.uploaded');

        return $media;
    }
}
