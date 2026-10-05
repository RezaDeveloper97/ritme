<?php

declare(strict_types=1);

use App\Domain\Directory\Join\Actions\SubmitJoinRequest;
use App\Domain\Newsletter\Enums\SubscriptionStatus;
use App\Domain\Newsletter\Models\Subscriber;
use App\Filament\Auth\AdminRole;
use App\Http\Controllers\Blog\NewsletterController;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Auth\Pages\Login;
use Filament\Facades\Filament;
use Illuminate\Auth\SessionGuard;
use Illuminate\Support\Facades\Cookie;
use Livewire\Livewire;

/*
 * L9-04b — follow-ups of the L9-04 audit (docs/SECURITY.md): F19 newsletter confirm by POST, F17 private pending
 * uploads, no remember-me on the admin login. PII-role MFA: tests/Feature/Admin/AdminAccessTest.php.
 */

it('confirms a newsletter subscription only by POST, with the lookup budget on the POST as well (F19)', function (): void {
    $this->seed(SettingsSeeder::class);
    $subscriber = Subscriber::query()->create(['email' => 's@example.com', 'token' => Subscriber::newToken(), 'consent_at' => now()]);

    // A scanner/prefetcher that follows the mailed link (even repeatedly) confirms nothing.
    $this->get(route('newsletter.confirm', [$subscriber->token]))->assertOk();
    $this->head(route('newsletter.confirm', [$subscriber->token]));
    expect($subscriber->fresh()?->status())->toBe(SubscriptionStatus::Pending);

    for ($i = 0; $i < NewsletterController::LOOKUPS_PER_MINUTE; $i++) {
        expect($this->post(route('newsletter.confirm.store', ['guess'.$i]))->status())->not->toBe(429);
    }
    $this->post(route('newsletter.confirm.store', [$subscriber->token]))->assertStatus(429);
    expect($subscriber->fresh()?->status())->toBe(SubscriptionStatus::Pending);
});

it('keeps the pending upload disk outside the web root and points its URLs at the admin route (F17)', function (): void {
    $config = config('filesystems.disks.'.SubmitJoinRequest::PHOTO_DISK);

    expect($config['driver'])->toBe('local')
        ->and($config['visibility'])->toBe('private')
        ->and($config['root'])->toStartWith(storage_path('app'))
        ->and($config['root'])->not->toStartWith(public_path())
        ->and($config['url'])->toBe('/admin/pending-media')
        ->and($config)->not->toHaveKey('serve');
});

it('does not offer remember-me on the admin login and ignores a forged remember flag', function (): void {
    $this->seed(AdminRolesSeeder::class);
    Filament::setCurrentPanel('admin');

    $html = (string) $this->get('/admin/login')->assertOk()->getContent();
    expect($html)->not->toContain('type="checkbox"')
        ->and($html)->not->toContain(__('filament-panels::auth/pages/login.form.remember.label'));

    $user = User::factory()->create(['password' => 'correct-horse-12']);
    $user->assignRole(AdminRole::Editor->value);

    Livewire::test(Login::class)
        ->assertFormFieldHidden('remember')
        ->set('data.remember', true)
        ->fillForm(['email' => $user->email, 'password' => 'correct-horse-12'])
        ->call('authenticate')
        ->assertHasNoFormErrors();

    $this->assertAuthenticatedAs($user);
    $guard = auth()->guard('web');
    assert($guard instanceof SessionGuard);
    expect(Cookie::hasQueued($guard->getRecallerName()))->toBeFalse(); // no remember-me cookie issued
});
