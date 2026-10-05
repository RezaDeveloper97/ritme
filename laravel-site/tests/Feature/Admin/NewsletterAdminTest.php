<?php

declare(strict_types=1);

use App\Domain\Newsletter\Models\Subscriber;
use App\Filament\Auth\AdminRole;
use App\Filament\Resources\Newsletter\Pages\ListSubscribers;
use App\Filament\Resources\Newsletter\SubscriberResource;
use App\Models\User;
use Database\Seeders\AdminRolesSeeder;
use Database\Seeders\SettingsSeeder;
use Filament\Facades\Filament;
use Illuminate\Support\Carbon;
use Livewire\Livewire;
use Spatie\Activitylog\Models\Activity;
use Tests\Feature\Admin\AdminMfa;

function newsletterAdmin(?AdminRole $role = AdminRole::Editor): User
{
    $user = User::factory()->create();
    if ($role !== null) {
        $user->assignRole($role->value);
    }

    return AdminMfa::enrol($user); // PII roles must have MFA (L9-04b)
}

/**
 * @param  array<string, mixed>  $attributes
 */
function subscriber(string $email, array $attributes = []): Subscriber
{
    return Subscriber::query()->create([
        'email' => $email,
        'source' => 'blog',
        'token' => Subscriber::newToken(),
        'consent_at' => Carbon::parse('2026-03-21 08:30:00', 'Asia/Tehran'),
        ...$attributes,
    ]);
}

beforeEach(function (): void {
    $this->seed([SettingsSeeder::class, AdminRolesSeeder::class]);
    Filament::setCurrentPanel('admin');

    $this->active = subscriber('active@example.com', ['confirmed_at' => Carbon::parse('2026-03-22 10:00:00', 'Asia/Tehran')]);
    $this->pending = subscriber('pending@example.com');
    $this->gone = subscriber('gone@example.com', [
        'confirmed_at' => Carbon::parse('2026-03-22 10:00:00', 'Asia/Tehran'),
        'unsubscribed_at' => Carbon::parse('2026-04-01 12:00:00', 'Asia/Tehran'),
    ]);
});

it('lets editors and super-admins list and export subscribers and denies everyone else', function (): void {
    $index = SubscriberResource::getUrl('index');
    $export = SubscriberResource::getUrl('export');

    $this->get($index)->assertRedirect('/admin/login');
    $this->get($export)->assertRedirect('/admin/login');

    $editor = newsletterAdmin();
    $this->actingAs($editor)->get($index)->assertOk()->assertSee('active@example.com')->assertSee('خروجی CSV');
    $this->actingAs($editor)->get($export)->assertOk();

    // Super-admins must enrol MFA before any admin page (RequireMultiFactorForRoles); the policy allows them.
    $super = newsletterAdmin(AdminRole::SuperAdmin);
    expect($super->can('viewAny', Subscriber::class))->toBeTrue()
        ->and($super->can('export', Subscriber::class))->toBeTrue();

    foreach ([AdminRole::SeoManager, AdminRole::Support, AdminRole::ShopManager, AdminRole::DirectoryManager, null] as $role) {
        $user = newsletterAdmin($role);
        $this->actingAs($user)->get($index)->assertForbidden();
        $this->actingAs($user)->get($export)->assertForbidden();
    }

    $inactive = newsletterAdmin();
    $inactive->forceFill(['is_active' => false])->save();
    expect($inactive->can('export', Subscriber::class))->toBeFalse()
        ->and(newsletterAdmin()->can('create', Subscriber::class))->toBeFalse()
        ->and(newsletterAdmin()->can('update', $this->active))->toBeFalse();
});

it('searches by email and filters by status', function (): void {
    $this->actingAs(newsletterAdmin());

    Livewire::test(ListSubscribers::class)
        ->assertCanSeeTableRecords([$this->active, $this->pending, $this->gone])
        ->assertSee('در انتظار تأیید')
        ->assertSee('۱۴۰۵/۰۱/۰۱ ۰۸:۳۰')
        ->searchTable('pending@')
        ->assertCanSeeTableRecords([$this->pending])
        ->assertCanNotSeeTableRecords([$this->active, $this->gone]);

    Livewire::test(ListSubscribers::class)
        ->filterTable('status', 'active')
        ->assertCanSeeTableRecords([$this->active])
        ->assertCanNotSeeTableRecords([$this->pending, $this->gone])
        ->assertActionHasUrl('export', SubscriberResource::getUrl('export', ['status' => 'active']));

    Livewire::test(ListSubscribers::class)
        ->filterTable('status', 'unsubscribed')
        ->assertCanSeeTableRecords([$this->gone])
        ->assertCanNotSeeTableRecords([$this->active, $this->pending]);
});

it('streams a minimal CSV with BOM, Persian headers and Jalali dates and logs the export', function (): void {
    $editor = newsletterAdmin();

    $response = $this->actingAs($editor)->get(SubscriberResource::getUrl('export'));
    $response->assertOk()
        ->assertHeader('Content-Type', 'text/csv; charset=UTF-8')
        ->assertDownload();

    $csv = $response->streamedContent();
    expect($csv)->toStartWith("\xEF\xBB\xBF");

    $lines = array_values(array_filter(explode("\n", substr($csv, 3))));
    expect($lines)->toHaveCount(4)
        ->and($lines[0])->toBe('ایمیل,وضعیت,"تاریخ ثبت‌نام","تاریخ تأیید","تاریخ لغو"')
        ->and($lines)->toContain('active@example.com,فعال,"1405/01/01 08:30","1405/01/02 10:00",')
        ->and($lines)->toContain('pending@example.com,"در انتظار تأیید","1405/01/01 08:30",,')
        ->and($lines)->toContain('gone@example.com,"لغو شده","1405/01/01 08:30","1405/01/02 10:00","1405/01/12 12:00"')
        ->and($csv)->not->toContain('blog')
        ->and($csv)->not->toContain($this->active->token);

    $activity = Activity::query()->where('log_name', 'newsletter')->sole();
    expect($activity->description)->toBe('newsletter.exported')
        ->and($activity->event)->toBe('exported')
        ->and($activity->causer_id)->toBe($editor->id)
        ->and($activity->properties['rows'])->toBe(3);
});

it('exports only the filtered subscribers', function (): void {
    $response = $this->actingAs(newsletterAdmin())->get(SubscriberResource::getUrl('export', ['status' => 'active']));
    $csv = $response->streamedContent();

    expect($csv)->toContain('active@example.com')
        ->and($csv)->not->toContain('pending@example.com')
        ->and($csv)->not->toContain('gone@example.com');

    $csv = $this->get(SubscriberResource::getUrl('export', ['search' => 'GONE']))->streamedContent();
    expect($csv)->toContain('gone@example.com')->and($csv)->not->toContain('active@example.com');

    expect(Activity::query()->where('description', 'newsletter.exported')->count())->toBe(2)
        ->and(Activity::query()->where('description', 'newsletter.exported')->latest('id')->first()?->properties['search'])->toBe('GONE');
});
