<?php

declare(strict_types=1);

namespace App\Domain\Content;

use App\Domain\Content\Enums\SitePhoto;
use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Media\Data\MediaData;

/**
 * Media of a committed site photo (SitePhoto), looked up by the file's content hash — the same hash StoreMedia
 * dedupes on, so the seeded row is found without storing ids anywhere. Null until SitePhotosSeeder has run (the
 * pages then keep their tinted placeholder box). Hashes are memoised per process; the media rows come from the
 * cached repository.
 */
final class SitePhotos
{
    /** @var array<string, string> */
    private array $hashes = [];

    public function __construct(private readonly MediaRepository $media) {}

    public function get(SitePhoto $photo): ?MediaData
    {
        $hash = $this->hash($photo);

        return $hash === null ? null : $this->media->findByHash($hash);
    }

    private function hash(SitePhoto $photo): ?string
    {
        if (! array_key_exists($photo->value, $this->hashes)) {
            $path = $photo->path();
            $this->hashes[$photo->value] = is_file($path) ? (string) hash_file('sha256', $path) : '';
        }

        return $this->hashes[$photo->value] !== '' ? $this->hashes[$photo->value] : null;
    }
}
