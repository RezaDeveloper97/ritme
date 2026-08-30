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
        $password = env('ADMIN_SEED_PASSWORD');

        // No default password: a blank env must fail the seed rather than
        // silently provision a well-known credential. This seeder runs on every
        // container start, so a default here would be a permanent backdoor.
        if (empty($password)) {
            if (app()->environment('local')) {
                $password = 'admin1234';
            } else {
                throw new \RuntimeException('ADMIN_SEED_PASSWORD must be set to seed the super admin.');
            }
        }

        // firstOrCreate — NOT updateOrCreate: once the admin exists, never
        // overwrite the password/role/is_active a real operator has set. Reseeding
        // on redeploy must be a no-op for an existing account.
        Admin::firstOrCreate(
            ['email' => $email],
            [
                'name' => env('ADMIN_SEED_NAME', 'مدیر ارشد'),
                'password' => $password,
                'role' => Admin::ROLE_SUPER,
                'is_active' => true,
            ]
        );
    }
}
