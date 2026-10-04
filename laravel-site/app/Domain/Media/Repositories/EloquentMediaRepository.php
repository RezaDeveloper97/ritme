<?php

declare(strict_types=1);

namespace App\Domain\Media\Repositories;

use App\Domain\Media\Contracts\MediaRepository;
use App\Domain\Media\Data\MediaData;
use App\Domain\Media\Models\Media;
use App\Domain\Media\Support\MediaDataFactory;

final class EloquentMediaRepository implements MediaRepository
{
    public function __construct(private readonly MediaDataFactory $factory) {}

    public function find(int $id): ?MediaData
    {
        $media = Media::query()->find($id);

        return $media === null ? null : $this->factory->make($media);
    }

    public function findMany(array $ids): array
    {
        if ($ids === []) {
            return [];
        }

        $result = [];
        foreach (Media::query()->whereIn('id', array_values(array_unique($ids)))->get() as $media) {
            $result[$media->id] = $this->factory->make($media);
        }

        return $result;
    }
}
