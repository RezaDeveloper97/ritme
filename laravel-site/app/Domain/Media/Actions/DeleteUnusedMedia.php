<?php

declare(strict_types=1);

namespace App\Domain\Media\Actions;

use App\Domain\Media\Models\Media;

/**
 * Deletes the given media items that nothing references (FindMediaUsages); referenced ones are kept. MediaObserver
 * removes the files and bumps the caches.
 */
final class DeleteUnusedMedia
{
    public function __construct(private readonly FindMediaUsages $usages) {}

    /**
     * @param  list<int>  $ids
     * @return array{deleted: list<int>, kept: list<int>}
     */
    public function handle(array $ids): array
    {
        $ids = array_values(array_unique(array_map('intval', $ids)));
        $kept = $this->usages->used($ids);

        $deleted = [];
        foreach (Media::query()->whereIn('id', array_diff($ids, $kept))->get() as $media) {
            $media->delete();
            $deleted[] = $media->id;
        }

        return ['deleted' => $deleted, 'kept' => $kept];
    }
}
