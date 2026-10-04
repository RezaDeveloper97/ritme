<?php

declare(strict_types=1);

namespace Database\Seeders;

use App\Filament\Auth\AdminRole;
use Illuminate\Database\Seeder;
use Spatie\Permission\Models\Role;
use Spatie\Permission\PermissionRegistrar;

/**
 * Creates the admin roles (idempotent). Users are never seeded: create the first one with `php artisan admin:create`.
 */
class AdminRolesSeeder extends Seeder
{
    public function run(): void
    {
        foreach (AdminRole::values() as $role) {
            Role::findOrCreate($role, 'web');
        }

        app(PermissionRegistrar::class)->forgetCachedPermissions();
    }
}
