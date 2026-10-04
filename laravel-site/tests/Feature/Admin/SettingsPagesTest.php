<?php

declare(strict_types=1);

use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Enums\SettingGroup;
use App\Filament\Auth\AdminRole;
use App\Filament\Pages\Settings\ContactSettingsPage;
use App\Filament\Pages\Settings\GeneralSettingsPage;
use App\Models\User;
use App\Support\Cache\NamespaceVersions;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Facades\Filament;
use Livewire\Livewire;
use Spatie\Activitylog\Models\Activity;

beforeEach(function (): void {
    $this->seed([SettingsSeeder::class, AdminRolesSeeder::class]);
    Filament::setCurrentPanel('admin');
    $this->editor = User::factory()->create();
    $this->editor->assignRole(AdminRole::Editor->value);
});

it('saves a settings group, bumps the caches and logs the change', function (): void {
    $settings = app(SettingsRepository::class);
    $versions = app(NamespaceVersions::class);
    $settings->all(); // warm
    $before = ['settings' => $versions->version('settings'), 'seo' => $versions->version('seo'), 'pages' => $versions->version('pages')];

    $this->actingAs($this->editor);
    Livewire::test(GeneralSettingsPage::class)
        ->assertSchemaStateSet(['site_name' => 'ریتمی'], 'form')
        ->fillForm(['tagline' => 'همراه هر روزِ بدن تو'])
        ->call('save')
        ->assertHasNoFormErrors()
        ->assertNotified();

    expect($versions->version('settings'))->toBeGreaterThan($before['settings'])
        ->and($versions->version('seo'))->toBeGreaterThan($before['seo'])
        ->and($versions->version('pages'))->toBeGreaterThan($before['pages'])
        ->and($settings->get(SettingGroup::General, 'tagline'))->toBe('همراه هر روزِ بدن تو');

    $activity = Activity::query()->where('log_name', 'settings')->latest('id')->first();
    expect($activity?->causer_id)->toBe($this->editor->id)
        ->and($activity?->properties->get('attributes'))->toBe(['tagline' => 'همراه هر روزِ بدن تو']);
});

it('validates settings input', function (): void {
    $this->actingAs($this->editor);

    Livewire::test(ContactSettingsPage::class)
        ->fillForm(['support_email' => 'not-an-email'])
        ->call('save')
        ->assertHasFormErrors(['support_email' => 'email']);
});

it('forbids saving settings without the right role', function (): void {
    $support = User::factory()->create();
    $support->assignRole(AdminRole::Support->value);
    $this->actingAs($support);

    Livewire::test(GeneralSettingsPage::class)->assertForbidden();
});
