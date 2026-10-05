<?php

declare(strict_types=1);

use App\Domain\Directory\Booking\Sms\LogSmsSender;
use App\Domain\Seo\Redirects\Data\RedirectMap;
use App\Domain\Seo\Redirects\Enums\RedirectCode;
use App\Domain\Settings\Data\AppLinksSettings;
use App\Filament\Auth\AdminRole;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Str;
use Psr\Log\AbstractLogger;

/*
 * L9-04 regression tests for the audit findings fixed in this task (docs/SECURITY.md, findings table).
 */

it('keeps regex redirects on this site even when a captured group starts with slashes (no open redirect)', function (): void {
    $map = new RedirectMap([], [
        [1, '~^/old(/.*)$~iu', '/$1', RedirectCode::Permanent->value],
        [2, '~^/ext/(.*)$~iu', 'https://ritme.ir/$1', RedirectCode::Permanent->value],
        [3, '~^/glue(.*)$~iu', 'https://ritme.ir$1', RedirectCode::Permanent->value],
    ]);

    expect($map->match('/old/evil.com')?->target)->toBe('/evil.com')
        ->and($map->match('/old/\\evil.com')?->target)->toBe('/evil.com')
        ->and($map->match('/ext/blog/x')?->target)->toBe('https://ritme.ir/blog/x')
        ->and($map->match('/glue@evil.com')?->target)->toBeNull()   // https://ritme.ir@evil.com → user-info trick
        ->and($map->match('/glue.evil.com')?->target)->toBeNull()   // https://ritme.ir.evil.com → other host
        ->and($map->match('/glue/ok')?->target)->toBe('https://ritme.ir/ok');
});

it('logs out an admin whose login was restored from the remember-me cookie (timeout + MFA cannot be skipped)', function (): void {
    $this->seed([SettingsSeeder::class, AdminRolesSeeder::class]);
    $user = User::factory()->create(['remember_token' => Str::random(60)]);
    $user->assignRole(AdminRole::Editor->value);

    $recaller = Auth::guard('web')->getRecallerName();

    $this->withCookie($recaller, $user->id.'|'.$user->remember_token.'|'.$user->getAuthPassword())
        ->get('/admin')
        ->assertRedirect('/admin/login');

    $this->assertGuest();
});

it('only keeps http(s) app links, so no javascript: URL reaches an href', function (): void {
    $links = AppLinksSettings::fromArray([
        'bazaar' => 'javascript:alert(1)',
        'myket' => 'https://myket.ir/app/ir.ritmeapp.ritme',
        'google_play' => 'data:text/html,x',
        'app_store' => ' JaVaScRiPt:alert(1)',
        'web_app' => 'https://web.ritme.ir',
    ]);

    expect($links->bazaar)->toBeNull()
        ->and($links->myket)->toBe('https://myket.ir/app/ir.ritmeapp.ritme')
        ->and($links->googlePlay)->toBeNull()
        ->and($links->appStore)->toBeNull()
        ->and($links->webApp)->toBe('https://web.ritme.ir');
});

it('never redirects a refused cart change to the Referer', function (): void {
    $this->seed(SettingsSeeder::class);

    $response = $this->withHeader('Referer', 'https://evil.example/phish')
        ->post(route('shop.cart.update', ['line' => 'p999999']), ['quantity' => 1]);

    expect((string) $response->headers->get('Location'))->not->toContain('evil.example');
});

it('redacts order / booking codes in the SMS log driver', function (): void {
    $logger = new class extends AbstractLogger
    {
        /** @var list<array<string, mixed>> */
        public array $records = [];

        public function log($level, Stringable|string $message, array $context = []): void
        {
            $this->records[] = $context;
        }
    };

    (new LogSmsSender($logger))->send('09121234567', 'کد پیگیری: K3M9-QX2T-D7XA');

    expect($logger->records[0]['text'])->toBe('کد پیگیری: ****-****-****')
        ->and($logger->records[0]['to'])->toBe('*******4567');
});
