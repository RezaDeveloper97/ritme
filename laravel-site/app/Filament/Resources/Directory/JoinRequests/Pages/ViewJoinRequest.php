<?php

declare(strict_types=1);

namespace App\Filament\Resources\Directory\JoinRequests\Pages;

use App\Filament\Resources\Directory\JoinRequests\JoinRequestResource;
use Filament\Resources\Pages\ViewRecord;

/**
 * Reviewing one join request: convert it into a draft place, reject it, or delete it.
 */
final class ViewJoinRequest extends ViewRecord
{
    protected static string $resource = JoinRequestResource::class;

    protected function getHeaderActions(): array
    {
        return [
            JoinRequestResource::approveAction(),
            JoinRequestResource::rejectAction(),
            JoinRequestResource::deleteAction(),
        ];
    }
}
