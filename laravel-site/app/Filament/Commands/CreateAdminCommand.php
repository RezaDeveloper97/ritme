<?php

declare(strict_types=1);

namespace App\Filament\Commands;

use App\Filament\Auth\AdminRole;
use App\Models\User;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Validator;
use Illuminate\Validation\Rules\Password;
use Spatie\Permission\Models\Role;
use Spatie\Permission\PermissionRegistrar;

/**
 * Creates an admin user interactively. No password is ever seeded or accepted as an option (it would end up in the
 * shell history); it is always asked for, hidden.
 */
final class CreateAdminCommand extends Command
{
    protected $signature = 'admin:create
        {--name= : Display name}
        {--email= : Login email}
        {--role=super-admin : One of: super-admin, editor, seo-manager, shop-manager, directory-manager, support}';

    protected $description = 'Create an admin-panel user (asks for the password)';

    public function handle(): int
    {
        $name = $this->stringOption('name') ?? (string) $this->ask('Name');
        $email = $this->stringOption('email') ?? (string) $this->ask('Email');
        $role = AdminRole::tryFrom((string) $this->option('role'));

        if ($role === null) {
            $this->error('Unknown role. Valid roles: '.implode(', ', AdminRole::values()));

            return self::FAILURE;
        }

        $password = (string) $this->secret('Password (min. 12 characters)');
        $confirmation = (string) $this->secret('Repeat password');

        $validator = Validator::make(
            ['name' => $name, 'email' => $email, 'password' => $password, 'password_confirmation' => $confirmation],
            [
                'name' => ['required', 'string', 'max:255'],
                'email' => ['required', 'email', 'max:255', 'unique:users,email'],
                'password' => ['required', 'confirmed', Password::min(12)->letters()->numbers()],
            ],
        );

        if ($validator->fails()) {
            foreach ($validator->errors()->all() as $message) {
                $this->error($message);
            }

            return self::FAILURE;
        }

        $user = DB::transaction(static function () use ($name, $email, $password, $role): User {
            foreach (AdminRole::values() as $roleName) {
                Role::findOrCreate($roleName, 'web');
            }
            app(PermissionRegistrar::class)->forgetCachedPermissions();

            $user = User::query()->create([
                'name' => $name,
                'email' => mb_strtolower($email),
                'password' => $password,
                'is_active' => true,
            ]);
            $user->assignRole($role->value);

            return $user;
        });

        $this->info("Admin user #{$user->id} <{$user->email}> created with role {$role->value}.");
        if ($role === AdminRole::SuperAdmin) {
            $this->line('Super-admins must set up an authenticator app (TOTP) on first login.');
        }

        return self::SUCCESS;
    }

    private function stringOption(string $key): ?string
    {
        $value = $this->option($key);

        return is_string($value) && trim($value) !== '' ? trim($value) : null;
    }
}
