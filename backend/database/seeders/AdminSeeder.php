<?php

namespace Database\Seeders;

use App\Models\Admin;
use Illuminate\Database\Seeder;

class AdminSeeder extends Seeder
{
    /**
     * Creates the initial super admin. Credentials come from env so production
     * secrets are never committed; defaults are for local dev only.
     */
    public function run(): void
    {
        $email = env('ADMIN_SEED_EMAIL', 'admin@ritmeapp.ir');

        Admin::updateOrCreate(
            ['email' => $email],
            [
                'name' => env('ADMIN_SEED_NAME', 'مدیر ارشد'),
                'password' => env('ADMIN_SEED_PASSWORD', 'admin1234'),
                'role' => Admin::ROLE_SUPER,
                'is_active' => true,
            ]
        );
    }
}
