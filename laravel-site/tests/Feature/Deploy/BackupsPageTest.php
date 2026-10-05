<?php

declare(strict_types=1);

use App\Filament\Auth\AdminRole;
use App\Filament\Pages\System\Backups;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Filament\Facades\Filament;
use Illuminate\Support\Facades\Storage;
use Spatie\Activitylog\Models\Activity;
use Tests\Feature\Admin\AdminMfa;

/**
 * L10-02: admin backups page — super-admins only (the archives hold every order, booking and message), streamed
 * downloads of listed backups only, each download in the activity log.
 */
function backupsAdmin(?AdminRole $role): User
{
    $user = User::factory()->create();
    if ($role !== null) {
        $user->assignRole($role->value);
    }

    return AdminMfa::enrol($user);
}

beforeEach(function (): void {
    $this->seed([AdminRolesSeeder::class]);
    Filament::setCurrentPanel('admin');
    Storage::fake('local');
    $this->file = now()->subHour()->format('Y-m-d-H-i-s').'.zip';
    Storage::disk('local')->put('ritme-site/'.$this->file, 'PK-zip-bytes');
});

it('lists backups for super-admins only', function (): void {
    $url = Backups::getUrl();
    expect($url)->toEndWith('/admin/system/backups');

    $this->get($url)->assertRedirect('/admin/login');

    $this->actingAs(backupsAdmin(AdminRole::SuperAdmin))->get($url)->assertOk()
        ->assertSee($this->file)
        ->assertSee(Backups::downloadUrl('local', $this->file), false);

    foreach ([AdminRole::Editor, AdminRole::SeoManager, AdminRole::ShopManager, AdminRole::DirectoryManager, AdminRole::Support, null] as $role) {
        $this->actingAs(backupsAdmin($role))->get($url)->assertForbidden();
    }
});

it('streams a listed backup to a super-admin and logs the download', function (): void {
    $url = Backups::downloadUrl('local', $this->file);
    expect($url)->toContain('/admin/system/backups/local/');

    $this->get($url)->assertRedirect('/admin/login');
    $this->actingAs(backupsAdmin(AdminRole::ShopManager))->get($url)->assertForbidden();

    $admin = backupsAdmin(AdminRole::SuperAdmin);
    $response = $this->actingAs($admin)->get($url)->assertOk()
        ->assertHeader('Content-Type', 'application/zip')
        ->assertHeader('X-Content-Type-Options', 'nosniff');
    expect((string) $response->headers->get('Content-Disposition'))->toContain($this->file)
        ->and((string) $response->headers->get('Cache-Control'))->toContain('no-store')
        ->and($response->streamedContent())->toBe('PK-zip-bytes');

    $log = Activity::query()->where('log_name', 'backups')->latest('id')->first();
    expect($log?->causer_id)->toBe($admin->id)
        ->and($log?->properties['file'] ?? null)->toBe($this->file);
});

it('refuses unknown disks, unlisted files and traversal', function (): void {
    $admin = backupsAdmin(AdminRole::SuperAdmin);
    Storage::disk('local')->put('elsewhere/secret.zip', 'x');

    $this->actingAs($admin)->get('/admin/system/backups/public/'.$this->file)->assertNotFound();
    $this->actingAs($admin)->get('/admin/system/backups/local/missing.zip')->assertNotFound();
    $this->actingAs($admin)->get('/admin/system/backups/local/secret.zip')->assertNotFound();
    $this->actingAs($admin)->get('/admin/system/backups/local/..%2F..%2F.env')->assertNotFound();
    $this->actingAs($admin)->get('/admin/system/backups/local/laravel.log')->assertNotFound();
});
