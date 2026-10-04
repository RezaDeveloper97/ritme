<?php

declare(strict_types=1);

use App\Domain\Pwa\Version\BuildInfo;
use App\Domain\Settings\Actions\UpdateSettings;
use App\Domain\Settings\Enums\SettingGroup;
use App\Filament\Auth\AdminRole;
use App\Filament\Pages\Settings\PwaSettings;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Facades\Filament;
use Livewire\Livewire;

/** Points BuildInfo at a temporary build-id.json (null = no build deployed). */
function fakeBuild(?string $buildId): void
{
    $file = tempnam(sys_get_temp_dir(), 'build-id');
    if ($buildId === null) {
        @unlink($file);
    } else {
        file_put_contents($file, json_encode(['build_id' => $buildId]));
    }
    app()->instance(BuildInfo::class, new BuildInfo($file));
}

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
});

it('serves /pwa/version.json uncached, without cookies, with the deployed build', function (): void {
    fakeBuild('20261004120000-abc1234');

    $response = $this->get('/pwa/version.json')->assertOk()
        ->assertExactJson(['build_id' => '20261004120000-abc1234', 'min_build_id' => null, 'message' => '']);

    expect($response->headers->get('Cache-Control'))->toContain('no-store')
        ->and($response->headers->getCookies())->toBe([])
        ->and($response->headers->get('Content-Type'))->toContain('application/json');
});

it('reports the admin minimum build and message, capped at the deployed build', function (): void {
    fakeBuild('20261004120000-abc1234');
    app(UpdateSettings::class)->handle(SettingGroup::Pwa, ['min_build_id' => '20261001000000-0000aaa', 'update_message' => 'رفع یک مشکل مهم']);

    $this->get('/pwa/version.json')->assertJson(['min_build_id' => '20261001000000-0000aaa', 'message' => 'رفع یک مشکل مهم']);

    // A minimum newer than the deployed build would lock everyone out: capped to the deployed build.
    app(UpdateSettings::class)->handle(SettingGroup::Pwa, ['min_build_id' => '20991231000000']);
    $this->get('/pwa/version.json')->assertJson(['min_build_id' => '20261004120000-abc1234']);
});

it('ignores malformed minimum builds and reports no build when none is deployed', function (): void {
    fakeBuild(null);
    app(UpdateSettings::class)->handle(SettingGroup::Pwa, ['min_build_id' => 'v1.2.3']);

    $this->get('/pwa/version.json')->assertExactJson(['build_id' => null, 'min_build_id' => null, 'message' => '']);
});

it('renders the offline page on the site layout, noindex, with one h1 and the retry button', function (): void {
    $html = (string) $this->get('/offline')->assertOk()->getContent();

    expect(substr_count($html, '<h1'))->toBe(1)
        ->and($html)->toContain('اتصال اینترنت برقرار نیست')
        ->and($html)->toContain('<meta name="robots" content="noindex')
        ->and($html)->toContain('data-pwa-retry')
        ->and($html)->toContain('data-module="pwa"')
        ->and($html)->not->toMatch('/\sstyle\s*=/i')
        ->and($html)->not->toMatch('/<script(?![^>]*(type="module"|application\/ld\+json))[^>]*>[^<]+/i');
});

it('keeps the strict CSP (self-only scripts and workers) on public pages', function (): void {
    $csp = (string) $this->get('/offline')->headers->get('Content-Security-Policy');

    expect($csp)->toContain("script-src 'self'")
        ->toContain("worker-src 'self'")
        ->toContain("manifest-src 'self'")
        ->not->toContain('unsafe-inline');
});

it('lets an editor set the minimum build and force the deployed build from the PWA settings page', function (): void {
    fakeBuild('20261004120000-abc1234');
    $this->seed(AdminRolesSeeder::class);
    Filament::setCurrentPanel('admin');
    $editor = User::factory()->create();
    $editor->assignRole(AdminRole::Editor->value);
    $this->actingAs($editor);

    Livewire::test(PwaSettings::class)
        ->fillForm(['min_build_id' => 'not-a-build'])
        ->call('save')
        ->assertHasFormErrors(['min_build_id']);

    Livewire::test(PwaSettings::class)
        ->assertSee('20261004120000-abc1234')
        ->callAction('forceUpdate')
        ->assertNotified();

    $this->get('/pwa/version.json')->assertJson(['min_build_id' => '20261004120000-abc1234']);
});
