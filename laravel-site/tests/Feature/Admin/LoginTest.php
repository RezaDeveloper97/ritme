<?php

declare(strict_types=1);

use App\Filament\Auth\AdminRole;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Filament\Auth\Pages\Login;
use Filament\Facades\Filament;
use Livewire\Livewire;

beforeEach(function (): void {
    $this->seed(AdminRolesSeeder::class);
    Filament::setCurrentPanel('admin');
});

it('logs an active admin in and stamps last_login_at', function (): void {
    $user = User::factory()->create(['password' => 'correct-horse-12']);
    $user->assignRole(AdminRole::Editor->value);

    Livewire::test(Login::class)
        ->fillForm(['email' => $user->email, 'password' => 'correct-horse-12'])
        ->call('authenticate')
        ->assertHasNoFormErrors();

    $this->assertAuthenticatedAs($user);
    expect($user->refresh()->last_login_at)->not->toBeNull();
});

it('rejects inactive admins at login', function (): void {
    $user = User::factory()->create(['password' => 'correct-horse-12', 'is_active' => false]);
    $user->assignRole(AdminRole::Editor->value);

    Livewire::test(Login::class)
        ->fillForm(['email' => $user->email, 'password' => 'correct-horse-12'])
        ->call('authenticate')
        ->assertHasFormErrors(['email']);

    $this->assertGuest();
});

it('throttles repeated failed logins', function (): void {
    $user = User::factory()->create(['password' => 'correct-horse-12']);
    $user->assignRole(AdminRole::Editor->value);

    for ($i = 0; $i < 5; $i++) {
        Livewire::test(Login::class)
            ->fillForm(['email' => $user->email, 'password' => 'wrong-password'])
            ->call('authenticate');
    }

    Livewire::test(Login::class)
        ->fillForm(['email' => $user->email, 'password' => 'correct-horse-12'])
        ->call('authenticate')
        ->assertNotified();

    $this->assertGuest();
});
