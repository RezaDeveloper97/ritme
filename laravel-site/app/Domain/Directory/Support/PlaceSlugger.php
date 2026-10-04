<?php

declare(strict_types=1);

namespace App\Domain\Directory\Support;

use App\Domain\Directory\Enums\PlaceStatus;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceSlug;
use App\Support\Html\TextSlug;

/**
 * Place slugs (`/directory/place/{slug}`): normalised (Persian kept, TextSlug), unique against current slugs AND other
 * places' history; a changed slug of a published place goes into history so the old URL 301s (L5-03). Re-using one
 * of the place's own old slugs removes it from history.
 */
final class PlaceSlugger
{
    public const MAX_LENGTH = 120;

    public function assign(Place $place): void
    {
        $current = trim((string) $place->getAttribute('slug'));

        if ($place->exists && $current !== '' && ! $place->isDirty('slug')) {
            return;
        }

        $base = TextSlug::make($current !== '' ? $current : $place->name, self::MAX_LENGTH);
        $place->slug = $this->unique($base === '' ? 'place' : $base, $place->exists ? $place->id : null);
    }

    public function recordChange(Place $place): void
    {
        if (! $place->wasChanged('slug')) {
            return;
        }

        PlaceSlug::query()->where('place_id', $place->id)->where('slug', $place->slug)->delete();

        $previous = (string) $place->getOriginal('slug');
        $wasPublished = $place->getOriginal('status') === PlaceStatus::Published || $place->status === PlaceStatus::Published;
        if ($previous === '' || ! $wasPublished) {
            return; // a draft that was never public has no URL worth redirecting
        }

        if (! PlaceSlug::query()->where('slug', $previous)->exists()) {
            PlaceSlug::query()->create(['place_id' => $place->id, 'slug' => $previous]);
        }
    }

    private function unique(string $base, ?int $ignoreId): string
    {
        $candidate = $base;
        for ($n = 2; $this->taken($candidate, $ignoreId); $n++) {
            $suffix = '-'.$n;
            $candidate = mb_substr($base, 0, self::MAX_LENGTH - strlen($suffix), 'UTF-8').$suffix;
        }

        return $candidate;
    }

    private function taken(string $slug, ?int $ignoreId): bool
    {
        $live = Place::query()->where('slug', $slug)->when($ignoreId !== null, static fn ($q) => $q->whereKeyNot($ignoreId))->exists();

        return $live || PlaceSlug::query()->where('slug', $slug)->when($ignoreId !== null, static fn ($q) => $q->where('place_id', '!=', $ignoreId))->exists();
    }
}
