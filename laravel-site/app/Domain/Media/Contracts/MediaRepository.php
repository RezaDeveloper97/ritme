<?php

declare(strict_types=1);

namespace App\Domain\Media\Contracts;

use App\Domain\Media\Data\MediaData;

/**
 * Hot read side of the media library (<x-picture>, OG images). Cached in the `media` namespace.
 */
interface MediaRepository
{
    public function find(int $id): ?MediaData;

    /**
     * @param  list<int>  $ids
     * @return array<int, MediaData> keyed by id, missing ids omitted
     */
    public function findMany(array $ids): array;
}
