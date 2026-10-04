<?php

declare(strict_types=1);

use App\Filament\Auth\AdminRole;
use App\Models\User;
use Illuminate\Support\Facades\Hash;
use Spatie\Permission\Models\Role;

it('creates an admin interactively without a default password', function (): void {
    $this->artisan('admin:create', ['--role' => 'super-admin'])
        ->expectsQuestion('Name', 'Reza')
        ->expectsQuestion('Email', 'Admin@Ritme.test')
        ->expectsQuestion('Password (min. 12 characters)', 'a-strong-pass-123')
        ->expectsQuestion('Repeat password', 'a-strong-pass-123')
        ->assertSuccessful();

    $user = User::query()->where('email', 'admin@ritme.test')->firstOrFail();

    expect($user->hasRole(AdminRole::SuperAdmin->value))->toBeTrue()
        ->and($user->is_active)->toBeTrue()
        ->and(Hash::check('a-strong-pass-123', $user->password))->toBeTrue();
});

it('rejects weak or mismatched passwords and unknown roles', function (): void {
    $this->artisan('admin:create', ['--name' => 'A', '--email' => 'a@ritme.test'])
        ->expectsQuestion('Password (min. 12 characters)', 'short')
        ->expectsQuestion('Repeat password', 'short')
        ->assertFailed();

    $this->artisan('admin:create', ['--name' => 'A', '--email' => 'a@ritme.test', '--role' => 'root'])
        ->assertFailed();

    expect(User::query()->count())->toBe(0);
});

it('seeds roles but never users', function (): void {
    $this->seed();

    expect(User::query()->count())->toBe(0)
        ->and(Role::query()->pluck('name')->sort()->values()->all())
        ->toBe(collect(AdminRole::values())->sort()->values()->all());
});
