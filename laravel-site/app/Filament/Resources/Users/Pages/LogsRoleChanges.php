<?php

declare(strict_types=1);

namespace App\Filament\Resources\Users\Pages;

use App\Models\User;

/**
 * Role sync goes through a pivot (no model event), so it is written to the activity log explicitly.
 */
trait LogsRoleChanges
{
    protected function logRoles(string $event): void
    {
        $record = $this->getRecord();
        if (! $record instanceof User) {
            return;
        }

        activity('admin')
            ->performedOn($record)
            ->event($event)
            ->withProperties(['attributes' => ['roles' => $record->roles()->pluck('name')->implode(', ')]])
            ->log('roles');
    }
}
