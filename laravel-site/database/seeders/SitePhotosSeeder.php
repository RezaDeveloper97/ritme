<?php

declare(strict_types=1);

namespace Database\Seeders;

use App\Domain\Content\Enums\SitePhoto;
use App\Domain\Media\Actions\StoreMedia;
use App\Domain\Media\Data\MediaUpload;
use Illuminate\Database\Seeder;

/**
 * Imports the committed marketing photos (SitePhoto, resources/images/site) into the media library. Idempotent:
 * StoreMedia dedupes by content hash, so a re-run returns the existing rows and only new or replaced files are stored.
 */
class SitePhotosSeeder extends Seeder
{
    public function run(StoreMedia $store): void
    {
        foreach (SitePhoto::cases() as $photo) {
            if (! is_file($photo->path())) {
                continue;
            }

            $store->handle(new MediaUpload(
                path: $photo->path(),
                originalName: basename($photo->path()),
                alt: $photo->alt(),
            ));
        }
    }
}
