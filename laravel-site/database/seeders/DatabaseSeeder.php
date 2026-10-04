<?php

declare(strict_types=1);

namespace Database\Seeders;

use Illuminate\Database\Console\Seeds\WithoutModelEvents;
use Illuminate\Database\Seeder;

class DatabaseSeeder extends Seeder
{
    use WithoutModelEvents;

    /**
     * Seed the application's database. No users (and no default password) are seeded — the first admin is created
     * with `php artisan admin:create`.
     */
    public function run(): void
    {
        $this->call(SettingsSeeder::class);
        $this->call(AdminRolesSeeder::class);
    }
}
