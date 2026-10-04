<?php

declare(strict_types=1);

namespace App\Domain\Directory\Actions;

use App\Domain\Directory\Enums\PlaceStatus;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Support\DirectoryActivity;
use Illuminate\Database\Eloquent\Model;

/**
 * Publishes, unpublishes (back to draft) or suspends a place. Saved through the model so PlaceObserver bumps
 * `directory` + `sitemap` + `pages` (the place appears on / disappears from /directory at once); the change is
 * written to the activity log (`directory.place.status`). Returns false when the place already had that status.
 */
final class ChangePlaceStatus
{
    public function handle(Place $place, PlaceStatus $status, ?Model $causer = null): bool
    {
        if ($place->status === $status) {
            return false;
        }

        $from = $place->status;
        $place->status = $status;
        $place->save();

        DirectoryActivity::status($place, 'directory.place.status', $from->value, $status->value, $causer);

        return true;
    }
}
