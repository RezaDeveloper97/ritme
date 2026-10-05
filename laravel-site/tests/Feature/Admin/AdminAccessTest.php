<?php

declare(strict_types=1);

use App\Filament\Auth\AdminRole;
use App\Filament\Http\Middleware\EnforceSessionTimeout;
use App\Filament\Resources\Activities\ActivityResource;
use App\Filament\Resources\Activities\Pages\ListActivities;
use App\Filament\Resources\Users\Pages\ListUsers;
use App\Filament\Resources\Users\UserResource;
use App\Filament\Widgets\AdminOverview;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Illuminate\Auth\Events\Login;
use Spatie\Activitylog\Models\Activity;
use Tests\Feature\Admin\AdminMfa;

function adminUser(?AdminRole $role = AdminRole::Editor, array $attributes = []): User
{
    $user = User::factory()->create($attributes);
    if ($role !== null) {
        $user->assignRole($role->value);
    }

    return AdminMfa::enrol($user); // PII roles must have MFA (L9-04b)
}

/**
 * A super-admin who has already set up TOTP (MFA is required for that role).
 */
function superAdmin(): User
{
    return adminUser(AdminRole::SuperAdmin);
}

beforeEach(function (): void {
    $this->seed([SettingsSeeder::class, AdminRolesSeeder::class]);
});

it('redirects guests to the admin login', function (string $path): void {
    $this->get($path)->assertRedirect('/admin/login');
})->with(['/admin', '/admin/users', '/admin/activity', '/admin/settings/general']);

function externalHosts(string $html): array
{
    preg_match_all('/(?:src|href|url\()\s*=?\s*["\']?(https?:\/\/[^"\'\s)]+)/', $html, $matches);
    $hosts = array_unique(array_map(static fn (string $url): string => (string) parse_url($url, PHP_URL_HOST), $matches[1]));

    return array_values(array_diff($hosts, [(string) parse_url((string) config('app.url'), PHP_URL_HOST)]));
}

it('renders the panel in Persian RTL with the local Vazirmatn theme and no external hosts', function (): void {
    $pages = [
        (string) $this->get('/admin/login')->assertOk()->getContent(),
        (string) $this->actingAs(adminUser())->get('/admin')->assertOk()->getContent(),
    ];

    foreach ($pages as $html) {
        expect($html)
            ->toContain('lang="fa"')
            ->toContain('dir="rtl"')
            ->toContain('Vazirmatn')
            ->not->toContain('fonts.bunny.net')
            ->not->toContain('fonts.googleapis.com')
            ->and(externalHosts($html))->toBe([]);
    }
});

it('denies users without an admin role and inactive admins', function (): void {
    $this->actingAs(adminUser(null))->get('/admin')->assertForbidden();
    $this->actingAs(adminUser(AdminRole::Editor, ['is_active' => false]))->get('/admin')->assertForbidden();
});

it('lets any admin role open the dashboard', function (AdminRole $role): void {
    $user = adminUser($role);

    $this->actingAs($user)->get('/admin')->assertOk();
})->with(AdminRole::cases());

it('enforces per-resource policies (deny by default)', function (): void {
    $editor = adminUser(AdminRole::Editor);
    $support = adminUser(AdminRole::Support);

    $this->actingAs($editor)->get(UserResource::getUrl('index'))->assertForbidden();
    $this->actingAs($editor)->get(ActivityResource::getUrl('index'))->assertForbidden();
    $this->actingAs($support)->get(ActivityResource::getUrl('index'))->assertOk();
    $this->actingAs($support)->get(UserResource::getUrl('index'))->assertForbidden();
    $this->actingAs($support)->get('/admin/settings/general')->assertForbidden();
    $this->actingAs($editor)->get('/admin/settings/general')->assertOk();
    $this->actingAs($editor)->get('/admin/settings/legal')->assertForbidden();

    $admin = superAdmin();
    $this->actingAs($admin)->get(UserResource::getUrl('index'))->assertOk();
    $this->actingAs($admin)->get('/admin/settings/legal')->assertOk();
});

it('keeps the activity log read-only even for super-admins', function (): void {
    $admin = superAdmin();
    $activity = activity('admin')->log('test');

    expect($admin->can('viewAny', Activity::class))->toBeTrue()
        ->and($admin->can('update', $activity))->toBeFalse()
        ->and($admin->can('delete', $activity))->toBeFalse()
        ->and($admin->can('create', Activity::class))->toBeFalse()
        ->and($admin->can('delete', $admin))->toBeFalse();
});

it('requires every role that reads personal data to set up app MFA before using the panel (L9-04b)', function (): void {
    expect(config('filament.admin.mfa_required_roles'))->toBe(['super-admin', 'shop-manager', 'directory-manager', 'support']);

    foreach ([AdminRole::SuperAdmin, AdminRole::ShopManager, AdminRole::DirectoryManager, AdminRole::Support] as $role) {
        $user = User::factory()->create();
        $user->assignRole($role->value); // not enrolled yet
        $this->actingAs($user)->get('/admin')->assertRedirectContains('multi-factor-authentication');
        $this->actingAs(AdminMfa::enrol($user))->get('/admin')->assertOk();
    }

    foreach ([AdminRole::Editor, AdminRole::SeoManager] as $role) {
        $user = adminUser($role);
        expect($user->getAppAuthenticationSecret())->toBeNull();
        $this->actingAs($user)->get('/admin')->assertOk();
    }
});

it('shows placeholder counts on the dashboard', function (): void {
    $this->actingAs(superAdmin());

    Livewire\Livewire::test(AdminOverview::class)->assertSee('کاربران فعال مدیریت');
});

it('logs an admin out after the inactivity timeout', function (): void {
    config(['filament.admin.session_timeout' => 30]);
    $user = adminUser();

    $this->actingAs($user)
        ->withSession([EnforceSessionTimeout::SESSION_KEY => now()->subMinutes(31)->getTimestamp()])
        ->get('/admin')
        ->assertRedirect('/admin/login');

    $this->assertGuest();
});

it('records last_login_at on login without an activity entry', function (): void {
    $user = adminUser();

    event(new Login('web', $user, false));

    expect($user->refresh()->last_login_at)->not->toBeNull()
        ->and(Activity::query()->where('subject_id', $user->id)->where('event', 'updated')->count())->toBe(0);
});

it('logs admin user mutations to the activity log', function (): void {
    $user = adminUser();
    $user->update(['name' => 'نام تازه']);

    $activity = Activity::query()->where('log_name', 'admin')->where('event', 'updated')->latest('id')->first();

    expect($activity)->not->toBeNull()
        ->and($activity?->properties->get('attributes'))->toBe(['name' => 'نام تازه']);
});

it('keeps the admin brand colour in sync with the design token', function (): void {
    $css = (string) file_get_contents(resource_path('css/app.css'));
    preg_match('/--color-primary:\s*(#[0-9a-f]{6});/i', $css, $match);

    expect(strtolower((string) config('filament.admin.brand_color')))->toBe(strtolower($match[1] ?? ''));
});

it('lists admin users and activity entries for a super-admin', function (): void {
    $admin = superAdmin();
    $other = adminUser(AdminRole::SeoManager);
    $other->update(['name' => 'ویراستار']);
    $this->actingAs($admin);

    Livewire\Livewire::test(ListUsers::class)
        ->assertCanSeeTableRecords([$admin, $other]);

    Livewire\Livewire::test(ListActivities::class)
        ->assertCanSeeTableRecords(Activity::query()->get());
});
